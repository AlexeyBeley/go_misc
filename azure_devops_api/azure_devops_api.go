package azure_devops_api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	common_utils "github.com/AlexeyBeley/go_misc/common_utils"
	config_pol "github.com/AlexeyBeley/go_misc/configuration_policy"
	human_api_types "github.com/AlexeyBeley/go_misc/human_api_types/v1"
	logger "github.com/AlexeyBeley/go_misc/logger"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/graph"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/work"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

var lg = logger.Logger{Level: logger.INFO}

type Configuration struct {
	PersonalAccessToken    string                       `json:"PersonalAccessToken"`
	OrganizationName       string                       `json:"OrganizationName"`
	TeamName               string                       `json:"TeamName"`
	ProjectName            string                       `json:"ProjectName"`
	SprintName             string                       `json:"SprintName"`
	AreaPath               string                       `json:"AreaPath"`
	SystemAreaID           string                       `json:"SystemAreaID"`
	AreaPathByUserId       map[string]string            `json:"AreaPathByUserId"`
	TeamIdByUserId         map[string]string            `json:"TeamIdByUserId"`
	PerTypeProvisionKeyVal map[string]map[string]string `json:"PerTypeProvisionKeyVal"`
}

type WorkItem struct {
	ID        int                    `json:"id"`
	Rev       int                    `json:"rev"`
	Fields    map[string]interface{} `json:"fields"`
	Relations []struct {
		Rel        string                 `json:"rel"`
		URL        string                 `json:"url"`
		Attributes map[string]interface{} `json:"attributes"`
	}
}

type witWorkItemRelation struct {
	Rel        *string                `json:"rel"`
	Url        *string                `json:"url"`
	Attributes map[string]interface{} `json:"attributes"`
}

type witWorkItemQueryResult struct {
	WorkItems         *[]witWorkItemReference `json:"workItems"`
	WorkItemRelations *[]witWorkItemRelation  `json:"workItemRelations"`
}

type witWorkItemReference struct {
	Id *int `json:"id"`
}

func GetAllWits(config Configuration) error {
	ctx := context.Background()

	// Fetch work item IDs in batches using WIQL
	ids, err := getWorkItemIDs(config, ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("fetched %d\n", len(ids))
	return nil
}

func GetCoreClientAndCtx(config Configuration) (core.Client, context.Context, error) {
	organizationUrl := "https://dev.azure.com/" + config.OrganizationName // todo: replace value with your organization url

	// Create a connection to your organization
	connection := azuredevops.NewPatConnection(organizationUrl, config.PersonalAccessToken)

	ctx := context.Background()

	// Create a client to interact with the Core area
	coreClient, err := core.NewClient(ctx, connection)

	if err != nil {
		return coreClient, ctx, err
	}

	return coreClient, ctx, nil
}

// Helper function to create basic authentication header
func basicAuth(pat string) string {
	return base64.StdEncoding.EncodeToString([]byte(":" + pat))
}

func GetWorkClientAndCtx(config Configuration) (work.Client, context.Context, error) {
	organizationUrl := "https://dev.azure.com/" + config.OrganizationName

	// Create a connection to your organization
	connection := azuredevops.NewPatConnection(organizationUrl, config.PersonalAccessToken)

	ctx := context.Background()

	// Create a client to interact with the Work area
	Client, err := work.NewClient(ctx, connection)

	if err != nil {
		return Client, ctx, err
	}

	return Client, ctx, nil
}

func ValidateConfig(config Configuration) error {
	if config.OrganizationName == "" {
		return fmt.Errorf("parameter OrganizationName was not set in config")
	}
	return nil
}

func GetWorkItemTrackingClientAndCtx(config Configuration) (workitemtracking.Client, context.Context, error) {
	err := ValidateConfig(config)
	if err != nil {
		return nil, nil, err
	}
	organizationUrl := "https://dev.azure.com/" + config.OrganizationName

	// Create a connection to your organization
	connection := azuredevops.NewPatConnection(organizationUrl, config.PersonalAccessToken)

	ctx := context.Background()
	if ctx == nil {
		log.Fatal("Can not allocate context.Background")
	}

	// Create a client to interact with the Work area
	Client, err := workitemtracking.NewClient(ctx, connection)

	if err != nil {
		log.Printf("was not able to create new workitemtracking client")
		return Client, ctx, err
	}

	return Client, ctx, nil
}

func GetTeamUuid(config Configuration) (id uuid.UUID, err error) {
	WorkClient, ctx, err := GetWorkClientAndCtx(config)
	fmt.Printf("%v, %v, %v\n", WorkClient, ctx, err)
	CoreClient, ctx, err := GetCoreClientAndCtx(config)
	if err != nil {
		return id, err
	}
	WebApiTeams, err := CoreClient.GetAllTeams(ctx, core.GetAllTeamsArgs{})
	if err != nil {
		return id, err
	}

	for _, WebApiTeam := range *WebApiTeams {
		if config.TeamName == *WebApiTeam.Name {
			return *WebApiTeam.Id, nil
		}
	}

	return id, err
}

func GetIteration(config Configuration) (iteration work.TeamSettingsIteration, err error) {
	WorkClient, ctx, err := GetWorkClientAndCtx(config)

	if err != nil {
		return iteration, err
	}

	TeamSettingsIterations, err := WorkClient.GetTeamIterations(ctx, work.GetTeamIterationsArgs{Project: &(config.ProjectName)})

	if TeamSettingsIterations == nil {
		return iteration, err
	}
	for _, TeamSettingsIteration := range *TeamSettingsIterations {
		if *TeamSettingsIteration.Name == config.SprintName {
			return TeamSettingsIteration, nil
		}
	}
	return iteration, fmt.Errorf("was not able to find Iteration by name: %s", config.SprintName)
}

func CallGetWorkItems(config Configuration, ctx context.Context, WorkItemTrackingClient workitemtracking.Client, WitIds []int, ch chan *[]workitemtracking.WorkItem) (err error) {
	Fields := []string{"System.State", "System.Id", "System.CreatedBy", "System.CreatedDate"}
	args := workitemtracking.GetWorkItemsArgs{Project: &config.ProjectName, Ids: &WitIds, Fields: &Fields}
	IterationWorkItems, err := WorkItemTrackingClient.GetWorkItems(ctx, args)
	if err != nil {
		return err
	}

	ch <- IterationWorkItems
	close(ch)
	return nil
}

func GetAllFields() error {
	//todo: replace with real implementation
	connection := azuredevops.NewPatConnection("organizationUrl", "config.PersonalAccessToken")

	ctx := context.Background()

	// Create a client to interact with the Core area
	WorkItemTrackingClient, err := workitemtracking.NewClient(ctx, connection)
	variable := "&config.ProjectName"
	argsNew := workitemtracking.GetWorkItemFieldsArgs{Project: &variable}
	WorkItemField2, err := WorkItemTrackingClient.GetWorkItemFields(ctx, argsNew)
	if err != nil {
		return err
	}
	log.Printf("WorkItemField2: %v", (*WorkItemField2)[0])
	return nil
}

func CacheToFile(IterationWorkItems *[]workitemtracking.WorkItem, dstFilePath string) (err error) {
	jsonData, err := json.MarshalIndent(IterationWorkItems, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(dstFilePath, jsonData, 0644)
	if err != nil {
		return err
	}
	return nil

}

func LoadConfig(configFilePath string) (config Configuration, err error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func getWorkItemIDs(config Configuration, ctx context.Context) ([]int, error) {
	client := http.Client{Timeout: 10 * time.Second}
	requestUrl := "https://dev.azure.com/" + config.OrganizationName + "/" + config.ProjectName + "/_apis/wit/wiql?api-version=7.0"
	wiqlData := fmt.Sprintf(`{"query": "SELECT [System.Id] FROM WorkItems Where [System.TeamProject] = '%s' AND [System.AreaId] = %s"}`, config.ProjectName, config.SystemAreaID)
	AuthHeaderValue := "Basic " + basicAuth(config.PersonalAccessToken)

	jsonBody := []byte(wiqlData)
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestUrl, bodyReader)
	if err != nil {
		return []int{}, err
	}

	// Set the Authorization header
	req.Header.Set("Authorization", AuthHeaderValue)
	req.Header.Set("Content-Type", "application/json")

	// Send the request

	resp, err := client.Do(req)
	if err != nil {

		return nil, err
	}
	defer resp.Body.Close()

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status error: %d %s", resp.StatusCode, resp.Status)
	}

	// Decode the JSON response
	var queryResult witWorkItemQueryResult

	err = json.NewDecoder(resp.Body).Decode(&queryResult)
	if err != nil {
		return nil, err
	}

	// Extract work item IDs
	lenIds := len(*queryResult.WorkItems)
	if lenIds == 0 {
		return []int{}, fmt.Errorf("was not able to fetch Work Item Ids, check the quert: %v", wiqlData)
	}

	//allIDs := [lenIds]int[]{}
	var allIDs [20000]int
	if queryResult.WorkItems != nil {
		for i, workItem := range *queryResult.WorkItems {
			allIDs[i] = *workItem.Id
		}
	} else {
		log.Fatal("Can not fetch work item ids")
	}

	// Check if there are more results
	if queryResult.WorkItemRelations != nil && len(*queryResult.WorkItemRelations) != 0 {
		log.Fatal("Unexpected status: Length of the WorkItemRelations is not 0")
	}

	return allIDs[0:lenIds], nil
}
func getClient() http.Client {
	return http.Client{Timeout: 10 * time.Second}
}

func createRequest(config Configuration, ctx context.Context, RequestPath string, httpMethod string, body io.Reader, contentType string) (*http.Request, error) {

	requestUrl := "https://dev.azure.com/" + config.OrganizationName + "/" + config.ProjectName + "/_apis/" + RequestPath
	AuthHeaderValue := "Basic " + basicAuth(config.PersonalAccessToken)

	req, err := http.NewRequestWithContext(ctx, httpMethod, requestUrl, body)
	if err != nil {
		return req, err
	}

	// Set the Authorization header
	req.Header.Set("Authorization", AuthHeaderValue)
	req.Header.Set("Content-Type", contentType)
	return req, nil
}

func DownloadAllWits(config Configuration, dstFilePath string) error {
	ctx := context.Background()

	// Fetch work item IDs in batches using WIQL
	WitIds, err := getWorkItemIDs(config, ctx)
	if err != nil {
		return err
	}

	BulckSize := 50
	WitCount := len(WitIds)
	channelsCount := WitCount / BulckSize
	if BulckSize*channelsCount < WitCount {
		channelsCount += 1
	}

	channels := make([]chan *[]workitemtracking.WorkItem, channelsCount)
	for chanIndex := range channels {
		channels[chanIndex] = make(chan *[]workitemtracking.WorkItem)
	}

	i := 0
	for i < WitCount {
		log.Printf("Entering loop with i: %d\n", i)
		endIndex := i + BulckSize
		log.Printf("loop i:%d, endIndex: %d\n", i, endIndex)
		if i+BulckSize >= WitCount {
			endIndex = WitCount - 1
		}

		log.Printf("loop i:%d, endIndex after cahnge: %d\n", i, endIndex)
		log.Printf("loop i: %d, endIndex:%d, i/BulckSize: %d\n", i, endIndex, i/BulckSize)
		if i/BulckSize == 8 {
			log.Printf("Problem loop i: %d, endIndex:%d, i/BulckSize: %d\n", i, endIndex, i/BulckSize)

		}
		WitIdsSlice := WitIds[i:endIndex]
		chanIndex := i / BulckSize
		go func() {
			GetWorkItemsBySlice(config, ctx, WitIdsSlice, channels[chanIndex])
		}()

		i += BulckSize
	}

	//fmt.Printf("queryResult.WorkItems: %v\n", *(.Id)
	//*[]workitemtracking.WorkItem

	AllWits := []workitemtracking.WorkItem{}
	for j, ch := range channels {
		fmt.Printf("fetched from chanel %d out of %d channels\n", j, len(channels))
		IterationWorkItems := <-ch
		AllWits = append(AllWits, *IterationWorkItems...)

	}
	fmt.Printf("IterationWorkItems: %d\n", len(AllWits))
	err = CacheToFile(&AllWits, dstFilePath)
	if err != nil {
		return err
	}
	return nil
}

func GetWorkItemsBySlice(config Configuration, ctx context.Context, WitIds []int, ch chan *[]workitemtracking.WorkItem) error {
	retWorkItems := []workitemtracking.WorkItem{}

	for i, WitId := range WitIds {
		fmt.Printf("fetched witid  : %d/%d\n", i, len(WitIds))
		req, err := createRequest(config, ctx, fmt.Sprintf("wit/workitems/%d?$expand=all&api-version=7.0", WitId), http.MethodGet, nil, "application/json")
		if err != nil {
			return err
		}
		client := getClient()

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("received error in HTTP clinet request: %v", err)
			return err
		}
		defer resp.Body.Close()

		// Check the status code
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP status error: %d %s", resp.StatusCode, resp.Status)
		}

		// Decode the JSON response
		//var queryResult witWorkItemQueryResult
		var wit workitemtracking.WorkItem
		err = json.NewDecoder(resp.Body).Decode(&wit)
		if err != nil {
			return err
		}
		retWorkItems = append(retWorkItems, wit)
	}

	ch <- &retWorkItems
	return nil
}

func ReadWitsFromFile(filePath string) (wits []WorkItem, err error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &wits)
	if err != nil {
		return nil, err
	}
	return wits, nil
}

func SubmitSprintStatus(config Configuration, requestDicts []*(map[string]string)) error {
	// Provision parents
	// todo: Clean new identical parents by title
	for _, requestDict := range requestDicts {
		if (*requestDict)["ChildrenIDs"] != "" || (*requestDict)["ParentID"] == "-1" {
			continue
		}
		if _, err := strconv.Atoi((*requestDict)["ParentID"]); err != nil {
			return err
		}
	}
	workerIds := make(map[string]string)

	for _, requestDict := range requestDicts {

		strWorkerValue, ok := workerIds[(*requestDict)["WorkerID"]]
		if !ok {
			strWorker, strWorkerOK := (*requestDict)["WorkerID"]
			if !strWorkerOK {
				return fmt.Errorf("no Key WorkerID in %v", *requestDict)
			}
			if strWorker == "" {
				return fmt.Errorf("no WorkerID provided in %v", *requestDict)
			}
			workerIds[(*requestDict)["WorkerID"]] = getWorker(strWorker)
			strWorkerValue = workerIds[(*requestDict)["WorkerID"]]
		}
		(*requestDict)["WorkerID"] = strWorkerValue

		if (*requestDict)["ChildrenIDs"] == "" {
			continue
		}
		err := ProvisionWitFromDict(config, requestDict)
		if err != nil {
			return err
		}
	}

	for _, requestDict := range requestDicts {
		if (*requestDict)["ChildrenIDs"] != "" {
			continue
		}

		err := ProvisionWitFromDict(config, requestDict)
		if err != nil {
			return err
		}
		if (*requestDict)["ParentID"] == "-1" {
			continue
		}

		err = setWitParentFromWit(config, requestDict)
		if err != nil {
			return err
		}
	}
	return nil
}

func ProvisionWitFromDict(config Configuration, requestDict *(map[string]string)) error {
	// provision_work_item_from_dict

	if (*requestDict)["Id"] == "-1" {
		return nil
	}

	if strings.HasPrefix((*requestDict)["Id"], "CreatePlease:") {
		return CreateWit(config, requestDict)
	}
	return UpdateWit(config, (*requestDict))
}

func CreateWit(config Configuration, requestDict *(map[string]string)) error {
	req, err := GenerateCreateWitRequest(config, requestDict)
	if err != nil {
		log.Printf("received error in Generate Create Wit Request: %v", err)
		return err
	}
	client := getClient()

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("received error in HTTP clinet request: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status error: %d %s", resp.StatusCode, resp.Status)
	}

	// Decode the JSON response
	//var queryResult witWorkItemQueryResult
	var wit workitemtracking.WorkItem
	err = json.NewDecoder(resp.Body).Decode(&wit)
	if err != nil {
		return err
	}
	(*requestDict)["Id"] = strconv.Itoa(*wit.Id)

	return nil
}

func GenerateCreateWitRequest(config Configuration, requestDict *(map[string]string)) (*http.Request, error) {
	/*
		dictRequest["Id"] = wobject.Id
		dictRequest["ParentID"] = wobject.ParentID
		dictRequest["Priority"] = strconv.Itoa(wobject.Priority)
		dictRequest["Title"] = wobject.Title
		dictRequest["Description"] = wobject.Description
		dictRequest["LeftTime"] = strconv.Itoa(wobject.LeftTime)
		dictRequest["InvestedTime"] = strconv.Itoa(wobject.InvestedTime)
		dictRequest["WorkerID"] = wobject.WorkerID
		dictRequest["ChildrenIDs"] = strings.Join(*wobject.ChildrenIDs, ",")
		dictRequest["Sprint"] = wobject.Sprint
		dictRequest["Status"] = wobject.Status
		dictRequest["Type"] = wobject.Type
	*/

	if config.AreaPath == "" {
		return nil, fmt.Errorf("error config.AreaPath is empty, %v", config)
	}

	if value, err := strconv.Atoi((*requestDict)["Priority"]); err != nil || value == -1 {
		return nil, fmt.Errorf("creating Wobject has malformed Prioriy: %v, %v", value, err)
	}

	ctx := context.Background()
	postList := []map[string]string{}

	var witUrlType string
	switch {
	case (*requestDict)["Type"] == "UserStory":
		witUrlType = "$User%20Story"
	case (*requestDict)["Type"] == "Task" || (*requestDict)["Type"] == "Bug":
		witUrlType = "$" + (*requestDict)["Type"]
	default:
		return nil, fmt.Errorf("unknown WIT Type: %s", (*requestDict)["Type"])
	}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.AreaPath",
		"value": config.AreaPath,
	})

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.Title",
		"value": (*requestDict)["Title"],
	})

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.Description",
		"value": (*requestDict)["Description"],
	})

	iteration, err := GetIteration(config)
	if err != nil {
		return nil, err
	}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.IterationPath",
		"value": *iteration.Path,
	})

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/Microsoft.VSTS.Common.Priority",
		"value": (*requestDict)["Priority"],
	})

	err = fillCreateWitRequestTimes(&postList, requestDict)
	if err != nil {
		return nil, err
	}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.AssignedTo",
		"value": (*requestDict)["WorkerID"],
	})

	fmt.Printf("Creating new Azure Devops WorkITem  : %v\n", requestDict)

	postData, err := json.Marshal(postList)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %v", err)
	}
	req, err := createRequest(config, ctx, fmt.Sprintf("wit/workitems/%s?api-version=7.0", witUrlType), http.MethodPost, bytes.NewBuffer(postData), "application/json-patch+json")

	return req, err
}

func fillCreateWitRequestTimes(postList *[]map[string]string, requestDict *map[string]string) error {
	if (*requestDict)["LeftTime"] == "-1" {
		return nil
	}

	*postList = append(*postList, map[string]string{
		"op":    "add",
		"path":  "/fields/Microsoft.VSTS.Scheduling.RemainingWork",
		"value": (*requestDict)["LeftTime"],
	})

	*postList = append(*postList, map[string]string{
		"op":    "add",
		"path":  "/fields/Microsoft.VSTS.Scheduling.CompletedWork",
		"value": (*requestDict)["InvestedTime"],
	})

	intLeftTime, err := strconv.Atoi((*requestDict)["LeftTime"])
	if err != nil {
		return err
	}

	intInvestedTime, err := strconv.Atoi((*requestDict)["InvestedTime"])
	if err != nil {
		return err
	}

	originalEstimate := strconv.Itoa(intLeftTime + intInvestedTime)

	*postList = append(*postList, map[string]string{
		"op":    "add",
		"path":  "/fields/Microsoft.VSTS.Scheduling.OriginalEstimate",
		"value": originalEstimate,
	})
	return nil
}

func UpdateWit(config Configuration, requestDict map[string]string) error {
	req, err := GenerateUpdateWitRequest(config, requestDict)
	if err != nil {
		log.Printf("received error in Generate Create Wit Request: %v", err)
		return err
	}
	return Patch(req)

}

func setWitParentFromWit(config Configuration, requestDict *map[string]string) error {
	/*
			            logger.info(f"Removing parent: {wit_id}")
		            request_remove_parent = [{"op": "remove", "path": "/relations/0"}]
		            self.patch(url, request_remove_parent)

		            logger.info(f"Adding parent: {wit_id}")
		            return self.patch(url, request_data)
	*/
	ctx := context.Background()
	postList := []map[string]any{}

	postList = append(postList, map[string]any{
		"op":   "add",
		"path": "/relations/-",
		"value": map[string]string{
			"rel": "System.LinkTypes.Hierarchy-Reverse",
			"url": fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/wit/workItems/%s", config.OrganizationName, config.ProjectName, (*requestDict)["ParentID"]),
		},
	})

	postData, err := json.Marshal(postList)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}
	fmt.Printf("Updating Azure Devops WorkITem  : %v\n", requestDict)

	req, err := createRequest(config, ctx, fmt.Sprintf("wit/workitems/%s?api-version=7.0", (*requestDict)["Id"]), http.MethodPatch, bytes.NewBuffer(postData), "application/json-patch+json")

	if err != nil {
		log.Printf("received error in Generate Create Wit Request: %v", err)
		return err
	}
	return Patch(req)

}
func Patch(req *http.Request) error {

	client := getClient()

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("received error in HTTP clinet request: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status error: %d %s", resp.StatusCode, resp.Status)
	}

	// Decode the JSON response
	// var queryResult witWorkItemQueryResult
	var wit workitemtracking.WorkItem
	err = json.NewDecoder(resp.Body).Decode(&wit)
	if err != nil {
		return err
	}

	return nil
}

func GenerateUpdateWitRequest(config Configuration, requestDict map[string]string) (*http.Request, error) {
	/*
		dictRequest["Id"] = wobject.Id
		dictRequest["ParentID"] = wobject.ParentID
		dictRequest["Priority"] = strconv.Itoa(wobject.Priority)
		dictRequest["Title"] = wobject.Title
		dictRequest["Description"] = wobject.Description
		dictRequest["LeftTime"] = strconv.Itoa(wobject.LeftTime)
		dictRequest["InvestedTime"] = strconv.Itoa(wobject.InvestedTime)
		dictRequest["WorkerID"] = wobject.WorkerID
		dictRequest["ChildrenIDs"] = strings.Join(*wobject.ChildrenIDs, ",")
		dictRequest["Sprint"] = wobject.Sprint
		dictRequest["Status"] = wobject.Status
		dictRequest["Type"] = wobject.Type
	*/

	if config.AreaPath == "" {
		return nil, fmt.Errorf("error config.AreaPath is empty, %v", config)
	}

	ctx := context.Background()
	postList := []map[string]string{}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.AreaPath",
		"value": config.AreaPath,
	})

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.Title",
		"value": requestDict["Title"],
	})

	if requestDict["Priority"] != "-1" {
		postList = append(postList, map[string]string{
			"op":    "add",
			"path":  "/fields/Microsoft.VSTS.Common.Priority",
			"value": requestDict["Priority"],
		})
	}

	iteration, err := GetIteration(config)
	if err != nil {
		return nil, err
	}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.IterationPath",
		"value": *iteration.Path,
	})

	err = fillUpdateWitRequestTimes(&postList, requestDict)
	if err != nil {
		return nil, err
	}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.AssignedTo",
		"value": requestDict["WorkerID"],
	})

	postData, err := json.Marshal(postList)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %v", err)
	}
	fmt.Printf("Updating Azure Devops WorkITem  : %v\n", requestDict)

	req, err := createRequest(config, ctx, fmt.Sprintf("wit/workitems/%s?api-version=7.0", requestDict["Id"]), http.MethodPatch, bytes.NewBuffer(postData), "application/json-patch+json")

	return req, err
}

func fillUpdateWitRequestTimes(postList *[]map[string]string, requestDict map[string]string) error {

	if requestDict["LeftTime"] != "-1" {
		_, err := strconv.Atoi(requestDict["LeftTime"])
		if err != nil {
			return err
		}

		*postList = append(*postList, map[string]string{
			"op":    "add",
			"path":  "/fields/Microsoft.VSTS.Scheduling.RemainingWork",
			"value": requestDict["LeftTime"],
		})
	}

	if requestDict["InvestedTime"] != "-1" {
		_, err := strconv.Atoi(requestDict["InvestedTime"])
		if err != nil {
			return err
		}

		*postList = append(*postList, map[string]string{
			"op":    "add",
			"path":  "/fields/Microsoft.VSTS.Scheduling.CompletedWork",
			"value": requestDict["InvestedTime"],
		})
	}

	return nil
}

func getWorker(uniqueNamePart string) string {
	// data = workItem.Fields["System.AssignedTo"].(map[string]interface{})["uniqueName"].(string)
	ret := strings.Split(uniqueNamePart, ".")
	nameRunes := []rune(ret[0])
	nameRunes[0] = unicode.ToUpper(nameRunes[0])

	lastNameRunes := []rune(ret[1])
	lastNameRunes[0] = unicode.ToUpper(lastNameRunes[0])
	return string(nameRunes) + " " + string(lastNameRunes)

}

type AzureDevopsAPI struct {
	Configuration          *Configuration
	GitClient              GitClient
	BuildClient            BuildClient
	GraphClient            GraphClient
	CoreClient             CoreClient
	WorkClient             WorkClient
	WorkItemTrackingClient WorkItemTrackingClient
}

func validateConfig(config *Configuration) error {
	errors := []string{}
	if config.OrganizationName == "" {
		errors = append(errors, fmt.Sprintf("OrganizationName was not set"))
	}
	if config.PersonalAccessToken == "" {
		errors = append(errors, fmt.Sprintf("PersonalAccessToken was not set"))
	}
	if len(errors) == 0 {
		return nil
	}
	return fmt.Errorf("validating Azure Devops Configuration: %s", strings.Join(errors, "\n"))
}

func AzureDevopsAPINew(options ...config_pol.Option) (*AzureDevopsAPI, error) {
	config := &Configuration{}
	retAPI := &AzureDevopsAPI{}

	for _, option := range options {
		err := option(retAPI, config)
		if err != nil {
			return nil, err
		}
	}

	err := validateConfig(config)
	if err != nil {
		return nil, err
	}

	organizationUrl := "https://dev.azure.com/" + config.OrganizationName // todo: replace value with your organization url

	// Create a connection to your organization
	connection := azuredevops.NewPatConnection(organizationUrl, config.PersonalAccessToken)

	ctx := context.Background()

	gitClient, err := GitClientNew(config, ctx, connection)
	if err != nil {
		return nil, err
	}
	retAPI.GitClient = *gitClient

	BuildClient, err := BuildClientNew(config, ctx, connection)
	if err != nil {
		return nil, err
	}
	retAPI.BuildClient = *BuildClient

	GraphClient, err := GraphClientNew(config, ctx, connection)
	if err != nil {
		return nil, err
	}
	retAPI.GraphClient = *GraphClient

	CoreClient, err := CoreClientNew(config, ctx, connection)
	if err != nil {
		return nil, err
	}

	retAPI.CoreClient = *CoreClient
	WorkClient, err := WorkClientNew(config, ctx, connection)
	if err != nil {
		return nil, err
	}

	retAPI.WorkClient = *WorkClient

	workItemTrackingClientNew, err := WorkItemTrackingClientNew(config, ctx, connection)
	if err != nil {
		return nil, err
	}

	retAPI.WorkItemTrackingClient = *workItemTrackingClientNew

	return retAPI, nil
}

func (azureDevopsAPI *AzureDevopsAPI) SetConfiguration(Config any) error {
	AzureDevopsAPIConfig, ok := Config.(*Configuration)
	if !ok {
		return fmt.Errorf("was not able to convert %v to HumanAPIConfig", Config)
	}
	azureDevopsAPI.Configuration = AzureDevopsAPIConfig
	return nil
}

func (azureDevopsAPI *AzureDevopsAPI) GetRepositories() ([]git.GitRepository, error) {
	allRepositories, err := azureDevopsAPI.GitClient.GetRepositories()
	if err != nil {
		return nil, err
	}

	if len(allRepositories) == 0 {
		return nil, fmt.Errorf("no repositories found in project '%s'", azureDevopsAPI.Configuration.ProjectName)
	}

	return allRepositories, nil
}

func (azureDevopsAPI *AzureDevopsAPI) GetPipelineDefinitions() ([]build.BuildDefinitionReference, error) {
	Definitions, err := azureDevopsAPI.BuildClient.GetDefinitions()
	if err != nil {
		return nil, err
	}

	if len(Definitions) == 0 {
		return nil, fmt.Errorf("no repositories found in project '%s'", azureDevopsAPI.Configuration.ProjectName)
	}

	return Definitions, nil
}

func (azureDevopsAPI *AzureDevopsAPI) GetPipelineDefinition() ([]build.BuildDefinitionReference, error) {
	Definitions, err := azureDevopsAPI.BuildClient.GetDefinitions()
	if err != nil {
		return nil, err
	}

	for _, Definition := range Definitions {
		DefinitionFull, err := azureDevopsAPI.BuildClient.GetDefinition(Definition.Id)
		if err != nil {
			return nil, err
		}
		fmt.Printf("DefinitionFull: %v", DefinitionFull)
	}

	return Definitions, nil
}

func (azureDevopsAPI *AzureDevopsAPI) ProvisionWitFromDict(requestDict *(map[string]string)) error {
	// provision_work_item_from_dict

	if (*requestDict)["Id"] == "-1" {
		return nil
	}

	return fmt.Errorf("deprecated, use provision_wobject instead ")

}

func (azureDevopsAPI *AzureDevopsAPI) GenerateCreateWitRequest(requestDict *(map[string]string)) (*http.Request, error) {
	/*
		dictRequest["Id"] = wobject.Id
		dictRequest["ParentID"] = wobject.ParentID
		dictRequest["Priority"] = strconv.Itoa(wobject.Priority)
		dictRequest["Title"] = wobject.Title
		dictRequest["Description"] = wobject.Description
		dictRequest["LeftTime"] = strconv.Itoa(wobject.LeftTime)
		dictRequest["InvestedTime"] = strconv.Itoa(wobject.InvestedTime)
		dictRequest["WorkerID"] = wobject.WorkerID
		dictRequest["ChildrenIDs"] = strings.Join(*wobject.ChildrenIDs, ",")
		dictRequest["Sprint"] = wobject.Sprint
		dictRequest["Status"] = wobject.Status
		dictRequest["Type"] = wobject.Type
	*/

	if value, err := strconv.Atoi((*requestDict)["Priority"]); err != nil || value == -1 {
		return nil, fmt.Errorf("creating Wobject has malformed Prioriy: %v, %v", value, err)
	}

	ctx := context.Background()
	postList := []map[string]string{}

	var witUrlType string
	switch {
	case (*requestDict)["Type"] == "UserStory":
		witUrlType = "$User%20Story"
	case (*requestDict)["Type"] == "Task" || (*requestDict)["Type"] == "Bug":
		witUrlType = "$" + (*requestDict)["Type"]
	default:
		return nil, fmt.Errorf("unknown WIT Type: %s", (*requestDict)["Type"])
	}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.AreaPath",
		"value": (*requestDict)["AreaPath"],
	})

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.Title",
		"value": (*requestDict)["Title"],
	})

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.Description",
		"value": (*requestDict)["Description"],
	})

	iteration, err := azureDevopsAPI.GetIteration()
	if err != nil {
		return nil, err
	}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.IterationPath",
		"value": *iteration.Path,
	})

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/Microsoft.VSTS.Common.Priority",
		"value": (*requestDict)["Priority"],
	})

	err = fillCreateWitRequestTimes(&postList, requestDict)
	if err != nil {
		return nil, err
	}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.AssignedTo",
		"value": (*requestDict)["WorkerID"],
	})

	fmt.Printf("Creating new Azure Devops WorkITem  : %v\n", requestDict)

	postData, err := json.Marshal(postList)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %v", err)
	}
	req, err := azureDevopsAPI.CreateRequest(ctx, fmt.Sprintf("wit/workitems/%s?api-version=7.0", witUrlType), http.MethodPost, bytes.NewBuffer(postData), "application/json-patch+json")

	return req, err
}

func (azureDevopsAPI *AzureDevopsAPI) GetIteration() (iteration work.TeamSettingsIteration, err error) {
	WorkClient, ctx, err := azureDevopsAPI.GetWorkClientAndCtx()

	if err != nil {
		return iteration, err
	}

	TeamSettingsIterations, err := WorkClient.GetTeamIterations(ctx, work.GetTeamIterationsArgs{Project: &(azureDevopsAPI.Configuration.ProjectName)})

	if TeamSettingsIterations == nil {
		return iteration, err
	}
	for _, TeamSettingsIteration := range *TeamSettingsIterations {
		if *TeamSettingsIteration.Name == azureDevopsAPI.Configuration.SprintName {
			return TeamSettingsIteration, nil
		}
	}
	return iteration, fmt.Errorf("was not able to find Iteration by name: %s", azureDevopsAPI.Configuration.SprintName)
}
func (azureDevopsAPI *AzureDevopsAPI) CreateRequest(ctx context.Context, RequestPath string, httpMethod string, body io.Reader, contentType string) (*http.Request, error) {

	requestUrl := "https://dev.azure.com/" + azureDevopsAPI.Configuration.OrganizationName + "/" + azureDevopsAPI.Configuration.ProjectName + "/_apis/" + RequestPath
	AuthHeaderValue := "Basic " + basicAuth(azureDevopsAPI.Configuration.PersonalAccessToken)

	req, err := http.NewRequestWithContext(ctx, httpMethod, requestUrl, body)
	if err != nil {
		return req, err
	}

	// Set the Authorization header
	req.Header.Set("Authorization", AuthHeaderValue)
	req.Header.Set("Content-Type", contentType)
	return req, nil
}

func (azureDevopsAPI *AzureDevopsAPI) GetWorkClientAndCtx() (work.Client, context.Context, error) {
	organizationUrl := "https://dev.azure.com/" + azureDevopsAPI.Configuration.OrganizationName

	// Create a connection to your organization
	connection := azuredevops.NewPatConnection(organizationUrl, azureDevopsAPI.Configuration.PersonalAccessToken)

	ctx := context.Background()

	// Create a client to interact with the Work area
	Client, err := work.NewClient(ctx, connection)

	if err != nil {
		return Client, ctx, err
	}

	return Client, ctx, nil
}

func (azureDevopsAPI *AzureDevopsAPI) GetWorker(Name *string) (*human_api_types.Worker, error) {
	NameParts, err := splitWorkerNameToParts(Name, []string{" ", ".", "-", "_", ","})
	if err != nil {
		return nil, err
	}

	users, err := azureDevopsAPI.GraphClient.ListUsers()
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		match, err := checkWorkerNamePartsMatch(user.DisplayName, NameParts)
		if err != nil {
			return nil, err
		}
		if match {
			lg.InfoF("test: %v", user)
			return &human_api_types.Worker{Id: *user.Descriptor, Name: *user.DisplayName, SystemName: *user.PrincipalName}, nil

		}
	}

	return nil, nil

}

func (azureDevopsAPI *AzureDevopsAPI) GetWorkerSprint(worker *human_api_types.Worker) (*human_api_types.Sprint, error) {
	teamID, err := azureDevopsAPI.GetWorkerTeamId(worker.Id)
	if err != nil {
		return nil, err
	}

	allSprints, err := azureDevopsAPI.WorkItemTrackingClient.GetSprints(&teamID)

	//itersold, err := azureDevopsAPI.WorkClient.GetIterations(&teamID)
	if err != nil {
		return nil, err
	}

	//now := time.Now()
	currentSprints := []human_api_types.Sprint{}
	nowTime := time.Now()
	for _, sprint := range allSprints {
		if nowTime.Before(sprint.DateEnd) && nowTime.After(sprint.DateStart) {
			currentSprints = append(currentSprints, sprint)
		}
	}

	if len(currentSprints) > 1 || len(currentSprints) == 0 {
		return nil, fmt.Errorf("expected to find single sprint, found: %d", len(currentSprints))
	}

	return &currentSprints[0], nil

}

func splitWorkerNameToParts(Name *string, Separators []string) ([]string, error) {
	ret := []string{*Name}
	for _, separator := range Separators {
		ret = splitSliceBySeparators(ret, separator)
	}
	return ret, nil
}

func splitSliceBySeparators(slice []string, separator string) []string {
	ret := []string{}

	for _, str := range slice {
		ret = append(ret, strings.Split(str, separator)...)
	}

	return ret
}

func checkWorkerNamePartsMatch(Name *string, parts []string) (bool, error) {
	if len(parts) == 0 {
		return false, fmt.Errorf("oarts are empty %s ", *Name)
	}
	for _, part := range parts {
		if !strings.Contains(*Name, part) {
			return false, nil
		}
	}

	return true, nil
}
func (azureDevopsAPI *AzureDevopsAPI) checkUserInputProvisionWobject(wobj *human_api_types.Wobject) error {
	acailableTypes := []string{"Bug", "Task"}
	if !slices.Contains(acailableTypes, wobj.Type) {
		return fmt.Errorf("wobject Type is '%s' not one of '%v'", wobj.Type, acailableTypes)
	}

	acailableStatuses := []string{"New", "Active", "Blocked", "Closed"}
	if !slices.Contains(acailableStatuses, wobj.Status) {
		return fmt.Errorf("wobject Status is '%s' not one of '%v'", wobj.Status, acailableStatuses)
	}
	return nil
}

func (azureDevopsAPI *AzureDevopsAPI) ProvisionWobject(wobj *human_api_types.Wobject) error {

	err := azureDevopsAPI.checkUserInputProvisionWobject(wobj)
	if err != nil {
		return err
	}
	//azureDevopsAPI.WorkItemTrackingClient.Client.CreateWorkItem(nil, workitemtracking.CreateWorkItemArgs{})
	requestDict, err := wobj.ConverttotMap()

	if err != nil {
		return err
	}

	Worker, err := azureDevopsAPI.GetWorkerByName(wobj.WorkerID)
	if err != nil {
		return err
	}

	path, err := azureDevopsAPI.GetAreaPath(Worker)
	if err != nil {
		return err
	}

	sprint, err := azureDevopsAPI.GetWorkerSprint(Worker)
	if err != nil {
		return err
	}

	if value, ok := (requestDict)["Id"]; !ok || value == "" {
		keyValMap := map[string]string{
			"/fields/System.IterationPath": sprint.Id,
			"/fields/System.AreaPath":      path,
			"/fields/System.Title":         wobj.Title,
			"/fields/System.Description":   wobj.Description,
		}

		Document := []webapi.JsonPatchOperation{}

		for Path, Value := range keyValMap {
			Document = append(Document, webapi.JsonPatchOperation{
				Op:    &webapi.OperationValues.Add,
				Path:  &Path,
				Value: Value,
			})
		}

		Document = append(Document, webapi.JsonPatchOperation{
			Op:    &webapi.OperationValues.Add,
			Path:  common_utils.StrPTR("/fields/System.AssignedTo"),
			Value: common_utils.StrPTR(Worker.Id),
		})

		keyVals, ok := azureDevopsAPI.Configuration.PerTypeProvisionKeyVal[wobj.Type]
		if ok {
			for Path, Value := range keyVals {
				Document = append(Document, webapi.JsonPatchOperation{
					Op:    &webapi.OperationValues.Add,
					Path:  &Path,
					Value: Value,
				})

			}
		}

		wit, err := azureDevopsAPI.WorkItemTrackingClient.CreateWit(&wobj.Type, &Document)
		if err != nil {
			return err
		}
		log.Printf("Created Wit: %d", *wit.Id)

		wobj.Id = strconv.Itoa(*wit.Id)
		wobj.Link = fmt.Sprintf("https://%s.visualstudio.com/%s/_workitems/edit/%d", strings.ToLower(azureDevopsAPI.Configuration.OrganizationName), azureDevopsAPI.Configuration.OrganizationName, *wit.Id)

		err = azureDevopsAPI.WorkItemTrackingClient.AddWitComment(*wit.Id, wobj.Description)
		if err != nil {
			return err
		}
	}

	err = azureDevopsAPI.UpdateWobject(wobj)
	if err != nil {
		return err
	}

	if wobj.Link == "" {
		return fmt.Errorf("wobject %s, '%s' Link was not set", wobj.Id, wobj.Title)
	}
	return nil
}

func (azureDevopsAPI *AzureDevopsAPI) UpdateWobject(wobj *human_api_types.Wobject) error {
	wobjID, err := strconv.Atoi(wobj.Id)
	if err != nil {
		return err
	}

	wit, err := azureDevopsAPI.WorkItemTrackingClient.GetWit(&wobjID)
	if err != nil {
		return err
	}
	wobj.Link = fmt.Sprintf("https://%s.visualstudio.com/%s/_workitems/edit/%d", strings.ToLower(azureDevopsAPI.Configuration.OrganizationName), azureDevopsAPI.Configuration.OrganizationName, *wit.Id)

	state, ok := (*wit.Fields)["System.State"].(string)
	if !ok {
		log.Fatalf("Could not find or assert System.State to a string for work item #%d.", *wit.Id)
	}
	keyValMap := map[string]string{}
	if wobj.Status != state {
		keyValMap["/fields/System.State"] = wobj.Status
	}

	Document := []webapi.JsonPatchOperation{}
	if len(keyValMap) == 0 {
		return nil
	}
	for Path, Value := range keyValMap {
		Document = append(Document, webapi.JsonPatchOperation{
			Op:    &webapi.OperationValues.Replace,
			Path:  &Path,
			Value: Value,
		})
	}
	_, err = azureDevopsAPI.WorkItemTrackingClient.UpdateWit(wit.Id, &Document)

	return err

}

func (azureDevopsAPI *AzureDevopsAPI) GetAreaPath(Worker *human_api_types.Worker) (string, error) {

	WorkerID := strings.ToLower(Worker.Id)
	path, ok := azureDevopsAPI.Configuration.AreaPathByUserId[WorkerID]
	if !ok {
		return "", fmt.Errorf("can not find area path for user %s", WorkerID)
	} else if ok {
		return path, nil
	}

	//todo: test
	Team, err := azureDevopsAPI.GetWorkerTeamId(WorkerID)
	if err != nil {
		return "", err
	}

	response, err := azureDevopsAPI.WorkClient.GetTeamFieldValues(&Team)
	if err != nil {
		return "", err
	}
	return *(*response.Values)[0].Value, nil
}

func (azureDevopsAPI *AzureDevopsAPI) GetWorkerByName(workerName string) (*human_api_types.Worker, error) {
	workerNameSpaceParts := strings.Split(workerName, " ")
	workerNameParts := []string{}
	for _, workerNameSpacePart := range workerNameSpaceParts {
		workerNameDotParts := strings.Split(workerNameSpacePart, ".")
		for _, workerNameDotPart := range workerNameDotParts {
			workerNameParts = append(workerNameParts, strings.ToLower(workerNameDotPart))
		}
	}

	workers, err := azureDevopsAPI.GraphClient.ListUsers()
	if err != nil {
		return nil, err
	}

	returnGraphUsers := []*graph.GraphUser{}
	for _, worker := range workers {
		lowerDisplayName := strings.ToLower(*worker.DisplayName)
		found := true
		for _, part := range workerNameParts {
			if !strings.Contains(lowerDisplayName, part) {
				found = false
				break
			}
		}
		if found {
			returnGraphUsers = append(returnGraphUsers, &worker)
		}
	}
	if len(returnGraphUsers) > 1 {
		userNames := []string{}
		for _, returnGraphUser := range returnGraphUsers {
			userNames = append(userNames, *returnGraphUser.DisplayName)
		}
		return nil, fmt.Errorf("found [%s] users using source name: '%s'", strings.Join(userNames, ", "), workerName)
	}

	if len(returnGraphUsers) == 0 {
		return nil, fmt.Errorf("can not find user using source name: '%s'", workerName)
	}

	ret := human_api_types.Worker{Id: *returnGraphUsers[0].MailAddress, Name: *returnGraphUsers[0].DisplayName, SystemName: *returnGraphUsers[0].PrincipalName}
	return &ret, nil
}

func (azureDevopsAPI *AzureDevopsAPI) GetWorkerTeamId(workerID string) (string, error) {
	workerID = strings.ToLower(workerID)
	value, ok := azureDevopsAPI.Configuration.TeamIdByUserId[workerID]
	if ok {
		return value, nil
	}

	fmt.Printf("Was not able to find team by worker id: %s", workerID)

	//todo:
	teams, err := azureDevopsAPI.CoreClient.GetTeams()
	if err != nil {
		return "", err
	}
	foundTeams := []string{}
	for _, team := range teams {
		members, err := azureDevopsAPI.CoreClient.GetTeamMembers(common_utils.StrPTR((*team.Id).String()))
		if err != nil {
			log.Printf("Warning: Could not get members for team '%s', '%s': %v", *team.Name, *team.Id, err)
			continue
		}

		// 3. Check if the workerID is in the member list
		for _, member := range *members {
			if member.Identity != nil && member.Identity.Id != nil && *member.Identity.Id == workerID {
				foundTeams = append(foundTeams, *team.Name)
				break // Found the user, no need to check other members of this team
			}
		}
	}

	return foundTeams[0], nil
}

func (azureDevopsAPI *AzureDevopsAPI) UpdateWit(requestDict map[string]string) error {
	req, err := azureDevopsAPI.GenerateUpdateWitRequest(requestDict)
	if err != nil {
		log.Printf("received error in Generate Create Wit Request: %v", err)
		return err
	}
	return Patch(req)

}

func (azureDevopsAPI *AzureDevopsAPI) GenerateUpdateWitRequest(requestDict map[string]string) (*http.Request, error) {

	ctx := context.Background()
	postList := []map[string]string{}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.AreaPath",
		"value": requestDict["AreaPath"],
	})

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.Title",
		"value": requestDict["Title"],
	})

	if requestDict["Priority"] != "-1" {
		postList = append(postList, map[string]string{
			"op":    "add",
			"path":  "/fields/Microsoft.VSTS.Common.Priority",
			"value": requestDict["Priority"],
		})
	}

	iteration, err := azureDevopsAPI.GetIteration()
	if err != nil {
		return nil, err
	}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.IterationPath",
		"value": *iteration.Path,
	})

	err = fillUpdateWitRequestTimes(&postList, requestDict)
	if err != nil {
		return nil, err
	}

	postList = append(postList, map[string]string{
		"op":    "add",
		"path":  "/fields/System.AssignedTo",
		"value": requestDict["WorkerID"],
	})

	postData, err := json.Marshal(postList)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %v", err)
	}
	fmt.Printf("Updating Azure Devops WorkITem  : %v\n", requestDict)

	req, err := azureDevopsAPI.CreateRequest(ctx, fmt.Sprintf("wit/workitems/%s?api-version=7.0", requestDict["Id"]), http.MethodPatch, bytes.NewBuffer(postData), "application/json-patch+json")

	return req, err
}
