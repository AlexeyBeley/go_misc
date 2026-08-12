package azure_devops_api

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/AlexeyBeley/go_misc/common_utils"
	"github.com/AlexeyBeley/go_misc/human_api_types/v1"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

type WorkItemTrackingClient struct {
	Client        workitemtracking.Client
	Configuration *Configuration
}

func WorkItemTrackingClientNew(Configuration *Configuration, context context.Context, connection *azuredevops.Connection) (*WorkItemTrackingClient, error) {

	workItemTrackingClient, err := workitemtracking.NewClient(context, connection)
	if err != nil {
		log.Fatalf("Failed to create Git client: %v", err)
		return nil, err
	}
	ret := &WorkItemTrackingClient{Configuration: Configuration, Client: workItemTrackingClient}

	return ret, nil
}

func (workItemTrackingClient *WorkItemTrackingClient) GetTeamSprints(teamId *string) ([]human_api_types.Sprint, error) {
	ret := []human_api_types.Sprint{}

	depth := 15
	rootNode, err := workItemTrackingClient.Client.GetClassificationNode(context.Background(), workitemtracking.GetClassificationNodeArgs{
		Project:        &workItemTrackingClient.Configuration.ProjectName,
		StructureGroup: &workitemtracking.TreeStructureGroupValues.Iterations, // Specify Iterations
		Depth:          &depth,
	})
	if err != nil {
		log.Fatalf("Error getting iteration paths: %v", err)
	}

	for _, rootChild := range *rootNode.Children {
		for _, subChild := range *rootChild.Children {
			if subChild.Attributes == nil {
				fmt.Printf("attributes is nil at rootChild: %s, subChild: %s\n", *rootChild.Name, *subChild.Name)
				continue
			}

			startDateStringAny, ok := (*subChild.Attributes)["startDate"]
			if !ok {
				continue
			}
			finishDateStringAny, ok := (*subChild.Attributes)["finishDate"]
			if !ok {
				continue
			}

			startDateString, ok := startDateStringAny.(string)
			if !ok {
				return nil, fmt.Errorf("error converting startDateStringAny: %v", startDateStringAny)
			}

			startDate, err := common_utils.StringToDate(startDateString)
			if err != nil {
				return nil, err
			}

			finishDateString, ok := finishDateStringAny.(string)
			if !ok {
				return nil, fmt.Errorf("error converting finishDateStringAny: %v", finishDateStringAny)
			}

			finishDate, err := common_utils.StringToDate(finishDateString)
			if err != nil {
				return nil, err
			}

			sprint := human_api_types.Sprint{Id: strings.Replace(*subChild.Path, "\\Iteration\\", "\\", 1), Name: *subChild.Name, DateStart: *startDate, DateEnd: *finishDate}
			ret = append(ret, sprint)
		}
	}

	return ret, nil
}

func (workItemTrackingClient *WorkItemTrackingClient) CreateWit(WitType *string, Document *[]webapi.JsonPatchOperation) (*workitemtracking.WorkItem, error) {

	workItem, err := workItemTrackingClient.Client.CreateWorkItem(context.Background(), workitemtracking.CreateWorkItemArgs{Document: Document,
		Project: &workItemTrackingClient.Configuration.ProjectName,
		Type:    WitType})

	return workItem, err

}

func (workItemTrackingClient *WorkItemTrackingClient) GetWit(Id *int) (*workitemtracking.WorkItem, error) {

	workItem, err := workItemTrackingClient.Client.GetWorkItem(context.Background(), workitemtracking.GetWorkItemArgs{Project: &workItemTrackingClient.Configuration.ProjectName,
		Id: Id})

	return workItem, err

}

func (workItemTrackingClient *WorkItemTrackingClient) AddWitComment(workItemID int, Comment string) error {
	var Document []webapi.JsonPatchOperation
	path := "/fields/System.History"
	Document = append(Document, webapi.JsonPatchOperation{
		Op:    &webapi.OperationValues.Add,
		Path:  &path,
		Value: Comment,
	})

	// 2. Call the API to update the work item.
	_, err := workItemTrackingClient.Client.UpdateWorkItem(context.Background(), workitemtracking.UpdateWorkItemArgs{
		Document: &Document,
		Id:       &workItemID,
	})
	if err != nil {
		return fmt.Errorf("error updating work item: %v", err)
	}
	return nil
}

func (workItemTrackingClient *WorkItemTrackingClient) UpdateWit(workItemID *int, Document *[]webapi.JsonPatchOperation) (*workitemtracking.WorkItem, error) {

	workItem, err := workItemTrackingClient.Client.UpdateWorkItem(context.Background(), workitemtracking.UpdateWorkItemArgs{
		Document: Document,
		Id:       workItemID,
	})
	if err != nil {
		return nil, fmt.Errorf("error updating work item: %v", err)
	}

	return workItem, err

}

// todo:
func (workItemTrackingClient *WorkItemTrackingClient) GetIterationBySrpint(sprint *human_api_types.Sprint) (*Iteration, error) {
	errorSuffix := "[work_item_tracking_client->GetIterationBySrpint]"

	depth := 15
	rootNode, err := workItemTrackingClient.Client.GetClassificationNode(context.Background(), workitemtracking.GetClassificationNodeArgs{
		Project:        &workItemTrackingClient.Configuration.ProjectName,
		StructureGroup: &workitemtracking.TreeStructureGroupValues.Iterations, // Specify Iterations
		Depth:          &depth,
	})
	if err != nil {
		return nil, fmt.Errorf("%s Error getting iteration paths: %v", errorSuffix, err)
	}

	for _, rootChild := range *rootNode.Children {
		for _, subChild := range *rootChild.Children {
			if sprint.Name != *subChild.Name {
				continue
			}
			iteration := Iteration{Id: subChild.Id,
				Identifier: subChild.Identifier,
				Path:       subChild.Path,
				Name:       subChild.Name,
				Attributes: subChild.Attributes}

			return &iteration, nil
		}
	}

	return nil, fmt.Errorf("%s Was not able to find Iteration by Sprint", errorSuffix)
}

func (workItemTrackingClient *WorkItemTrackingClient) GetWorkerIterationWorkItems(workerEmail string, iterationPath string) ([]*WorkItem, error) {

	errorSuffix := "[work_item_tracking_client->GetWorkerIterationWorkItems]"
	//Do you know why? Because microsoft is peice of shit!
	//the real iteration path "\\ProjName\\Iteration\\year\\id" is here must be represented as "ProjName\\year\\id"
	iterationPath = strings.Replace(iterationPath, "\\Iteration\\", "\\", 1)
	iterationPath = strings.TrimLeft(iterationPath, "\\")

	queryString := fmt.Sprintf(`
		SELECT [System.Id], [System.Title], [System.State] 
		FROM WorkItems 
		WHERE [System.AssignedTo] CONTAINS '%s'
		AND [System.IterationPath] = '%s'
		AND [System.State] <> 'Removed'`, workerEmail, iterationPath)

	wiql := workitemtracking.QueryByWiqlArgs{
		Wiql: &workitemtracking.Wiql{
			Query: &queryString,
		},
	}

	res, err := workItemTrackingClient.Client.QueryByWiql(context.Background(), wiql)
	if err != nil {
		log.Fatal(err)
	}
	wits := []*WorkItem{}
	for _, witResponse := range *res.WorkItems {
		wit := &WorkItem{ID: *witResponse.Id}
		updateSucceeded, err := workItemTrackingClient.UpdateWitInformation(wit)
		if err != nil {
			return nil, fmt.Errorf("%s was not able to update wits from the Wit Ids\n%v", errorSuffix, err)
		}
		if !updateSucceeded {
			return nil, fmt.Errorf("%s was not able to update wits from the Wit Ids", errorSuffix)
		}
		wits = append(wits, wit)

	}
	return wits, nil
}

func (workItemTrackingClient *WorkItemTrackingClient) UpdateWitInformation(wit *WorkItem) (bool, error) {
	var expand workitemtracking.WorkItemExpand
	expand = "All"
	params := workitemtracking.GetWorkItemArgs{Id: &wit.ID, Expand: &expand}
	azureDevopsWorkItem, err := workItemTrackingClient.Client.GetWorkItem(context.Background(), params)
	if err != nil {
		return false, fmt.Errorf("error updating work item information: %v", err)
	}
	wit.Fields = *azureDevopsWorkItem.Fields

	if azureDevopsWorkItem.Relations != nil {
		wit.Relations = []struct {
			Rel        string         `json:"rel"`
			URL        string         `json:"url"`
			Attributes map[string]any `json:"attributes"`
		}{}
		for _, rel := range *azureDevopsWorkItem.Relations {
			wit.Relations = append(wit.Relations, struct {
				Rel        string         `json:"rel"`
				URL        string         `json:"url"`
				Attributes map[string]any `json:"attributes"`
			}{Rel: *rel.Rel, URL: *rel.Url, Attributes: *rel.Attributes})
		}
	}

	return true, nil
}
