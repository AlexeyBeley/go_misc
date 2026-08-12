package human_api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AlexeyBeley/go_misc/azure_devops_api"
	"github.com/AlexeyBeley/go_misc/common_utils"
	config_pol "github.com/AlexeyBeley/go_misc/configuration_policy"
	human_api_types "github.com/AlexeyBeley/go_misc/human_api_types/v1"
)

type Configuration struct {
	SprintName                       string `json:"SprintName"`
	ReportsDirPath                   string `json:"ReportsDirPath"`
	WorkerId                         string `json:"WorkerId"`
	AzureDevopsConfigurationFilePath string `json:"AzureDevopsConfigurationFilePath"`
}

const preReportFileName = "pre_report.json"
const inputFileName = "input.hapi"
const baseFileName = "base.hapi"
const postReportFileName = "post_report.json"

func check(e error) {
	if e != nil {
		strErr := fmt.Sprintf("%v", e)
		data := []byte(strErr)
		err := os.WriteFile("/tmp/hapi.log", data, 0644) // 0644 are file permissions
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
		panic(e)
	}
}

func check_ng(prefixData string, e error) {
	if e != nil {
		strErr := fmt.Sprintf("%s, %v", prefixData, e)
		data := []byte(strErr)
		err := os.WriteFile("/tmp/hapi.log", data, 0644) // 0644 are file permissions
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
		panic(fmt.Errorf("%s: %v", prefixData, e))
	}
}

func DailyRoutine(configFilePath string) error {
	/*
		if _, err:= os.Stat(reportFilePath) ; err == nil {
			fmt.Println("File exists")
		} else if os.IsNotExist(err) {
			fmt.Println("File does not exist")
		} else {
			fmt.Println("Error checking file existence:", err)
		}

	*/
	fmt.Println("starting daily routine")
	config, err := loadConfiguration(configFilePath)
	if err != nil {
		log.Printf("Failed with error: %v\n", err)
		return err
	}
	fmt.Println("Loaded config")

	now := time.Now()
	dateDirName := now.Format("2006_01_02")

	dateDirPath := filepath.Join(config.ReportsDirPath, config.SprintName, dateDirName)
	fmt.Println("Generated new directory path: " + dateDirPath)

	curDir, err := os.Getwd()
	check(err)
	fmt.Printf("Current workind dir: %v\n", curDir)

	dateDirFullPath := filepath.Join(config.ReportsDirPath, config.SprintName, dateDirName)
	err = os.MkdirAll(dateDirFullPath, 0755)
	if err != nil {
		fmt.Printf("was not able to create '%v'\n", dateDirPath)
		return err
	}

	fmt.Println("Created new directory path: " + dateDirFullPath)

	preReportFilePath := filepath.Join(dateDirFullPath, preReportFileName)
	inputFilePath := filepath.Join(dateDirFullPath, inputFileName)
	baseFilePath := filepath.Join(dateDirFullPath, baseFileName)
	postReportFilePath := filepath.Join(dateDirFullPath, postReportFileName)

	if _, err := os.Stat(postReportFilePath); err == nil {
		return fmt.Errorf("post report file exists. The routine finished: %v", dateDirFullPath)
	}

	azure_devops_config, err := azure_devops_api.LoadConfig(config.AzureDevopsConfigurationFilePath)
	if err != nil {
		return err
	}
	log.Printf("inputFilePath: %v\n", inputFilePath)
	if !checkFileExists(inputFilePath) {
		return DailyRoutineExtract(config, azure_devops_config, preReportFilePath, inputFilePath, baseFilePath, postReportFilePath)
	}
	if !checkFileExists(preReportFilePath) ||
		!checkFileExists(inputFilePath) ||
		!checkFileExists(baseFilePath) ||
		checkFileExists(postReportFilePath) {
		return fmt.Errorf("undefined status: %s", postReportFilePath)
	}
	return DailyRoutineSubmit(azure_devops_config, inputFilePath, baseFilePath, postReportFilePath)

}

func DailyRoutineExtract(config Configuration, azureDevopsConfig azure_devops_api.Configuration, preReportFilePath, inputFilePath, baseFilePath, postReportFilePath string) (err error) {
	if !checkFileExists(preReportFilePath) {
		if checkFileExists(inputFilePath) {
			return fmt.Errorf("pre report file does not exist. Input file exists '%v'", inputFilePath)
		}
		if checkFileExists(baseFilePath) {
			return fmt.Errorf("pre report file does not exist. Base file exists '%v'", baseFilePath)
		}
		DownloadAllWits(azureDevopsConfig, preReportFilePath)
	}

	if !checkFileExists(inputFilePath) {

		GenerateDailyReport(config, preReportFilePath, baseFilePath)
		if err != nil {
			fmt.Println("Error copying file:", err)
			return err
		}
		//_, err = ConvertDailyJsonToHR(dailyJSONFilePath, baseFilePath)
		//check(err)

		err = copyFile(baseFilePath, inputFilePath)
		if err != nil {
			fmt.Println("Error copying file:", err)
			return err
		}
		return nil
	} else if checkFileExists(baseFilePath) {
		return fmt.Errorf("input file does not exist. Base file exists '%v'", baseFilePath)
	}

	if _, err := os.Stat(preReportFilePath); err == nil {
		fmt.Println("File exists")
	} else if os.IsNotExist(err) {
		fmt.Println("File does not exist")

		//ConvertToHapi(filepath.Dir(reportFilePath))
	} else {
		fmt.Println("Error checking file existence:", err)
	}
	return nil
}

func GenerateDailyReport(config Configuration, statusFilePath string, dstFilePath string) {
	wobjects, err := ConvertAzureDevopsStatusToWobjects(statusFilePath)
	check(err)
	GenerateDailyReportFromWobjects(config, wobjects, dstFilePath)
	//WorkerDailyReport{}
}

func GenerateDailyReportFromWobjects(config Configuration, wobjects map[string]*human_api_types.Wobject, dstFilePath string) (reportFilePath string) {
	log.Printf("filtering relevant wobkjects: %v\n", len(wobjects))
	wobjectsRelevant := FilterRelevantDailyReportWobjects(config, wobjects)
	new := []WorkerWobjReport{}
	active := []WorkerWobjReport{}
	blocked := []WorkerWobjReport{}
	closed := []WorkerWobjReport{}

	reports := []WorkerDailyReport{}
	var workerID string
	for wobjid, wobject := range wobjectsRelevant {
		if wobjid == "-1" {
			continue
		}
		var parentPointer *human_api_types.Wobject
		var childPointer *human_api_types.Wobject

		if len(*wobject.ChildrenIDs) != 0 {
			log.Printf("skipping Wobject with children: %v\n", wobjid)
			continue
		}

		parentPointer, childPointer = GenerateParentAndChildFromParentlessWobject(wobject, wobjectsRelevant)

		workerID = wobject.WorkerID

		report := WorkerWobjReport{Parent: []string{parentPointer.Type, parentPointer.Id, parentPointer.Title},
			Child: []string{childPointer.Type, childPointer.Id, childPointer.Title}}
		switch wobject.Status {
		case "New":
			new = append(new, report)
		case "Closed":
			closed = append(closed, report)
		case "Active":
			active = append(active, report)
		case "Blocked":
			blocked = append(blocked, report)
		default:
			check(fmt.Errorf("invalid wobject.Status: %v", wobject.Status))
		}
	}

	if len(new) == 0 {
		check(fmt.Errorf("new wobjects are empty: %v", new))
	}

	workerDailyReport := WorkerDailyReport{WorkerID: workerID,
		New:     new,
		Active:  active,
		Blocked: blocked,
		Closed:  closed,
	}
	reports = append(reports, workerDailyReport)
	WriteDailyToHRFile(reports, dstFilePath)
	return reportFilePath
}

// Generate Parent and child for Wobject that has not explicit parent.
// The wobject can become either Parent from new qobject or a Child with undefind (-1) Parent
func GenerateParentAndChildFromParentlessWobject(wobject *human_api_types.Wobject, wobjectsRelevant map[string]*human_api_types.Wobject) (parent, child *human_api_types.Wobject) {
	if wobject.Type == "Task" || wobject.Type == "Bug" {
		if wobject.ParentID == "" {
			wobject.ParentID = "-1"
		}
		if wobjectsRelevant[wobject.ParentID].ChildrenIDs == nil {
			wobjectsRelevant[wobject.ParentID].ChildrenIDs = new([]string)
		}
		*(wobjectsRelevant[wobject.ParentID].ChildrenIDs) = append(*(wobjectsRelevant[wobject.ParentID].ChildrenIDs), wobject.Id)
		parent = wobjectsRelevant[wobject.ParentID]
		child = wobject
	} else {
		wobject.ChildrenIDs = &[]string{"-1"}
		child = wobjectsRelevant["-1"]
		parent = wobject
	}

	return parent, child
}

func FilterRelevantDailyReportWobjects(config Configuration, wobjects map[string]*human_api_types.Wobject) map[string]*human_api_types.Wobject {
	log.Printf("filtering relevant wobjects: %v\n", len(wobjects))
	wobjectsRelevantById := make(map[string]*human_api_types.Wobject)

	wobjectsRelevantById["-1"] = &human_api_types.Wobject{Id: "-1",
		Description: "-1",
		Title:       "-1",
	}

	for _, wobject := range wobjects {
		if wobject.WorkerID != config.WorkerId {
			continue
		}
		if wobject.Sprint != config.SprintName {
			continue
		}
		if _, exists := wobjectsRelevantById[wobject.Id]; exists {
			continue
		}
		wobjectsRelevantById[wobject.Id] = wobject

		if _, existsInRelevant := wobjectsRelevantById[wobject.ParentID]; !existsInRelevant {
			if parent, existsInWobjects := wobjects[wobject.ParentID]; existsInWobjects {
				wobjectsRelevantById[wobject.ParentID] = parent
			}
		}
	}

	return wobjectsRelevantById
}

func ConvertAzureDevopsStatusToWobjects(filePath string) (wobjects map[string]*human_api_types.Wobject, err error) {
	wits, err := azure_devops_api.ReadWitsFromFile(filePath)
	wobjects = make(map[string]*human_api_types.Wobject)

	check(err)
	//log.Printf("todo: %v\n", wits)
	for _, wit := range wits {
		wobject, err := ConvertWitToWobject(wit)
		check(err)
		wobjects[wobject.Id] = &wobject
	}
	for wobjId, wobject := range wobjects {
		if wobject.ParentID != "" && wobject.ParentID != "-1" {
			parent, ok := wobjects[wobject.ParentID]
			if !ok {
				continue
			}
			*(parent.ChildrenIDs) = append(*(parent.ChildrenIDs), wobjId)
		}
	}
	return wobjects, nil
}

func ConvertWitToWobject(wit azure_devops_api.WorkItem) (wobject human_api_types.Wobject, err error) {
	wobject.ParentID = extractFloat64String(wit, "System.Parent")
	wobject.Id = strconv.Itoa(wit.ID)
	wobject.Title = wit.Fields["System.Title"].(string)
	wobject.Priority = extractFloat64Int(wit, "Microsoft.VSTS.Common.Priority")

	wobject.WorkerID = extractWorkerID(wit)
	wobject.ChildrenIDs = &[]string{}

	wobject.Status = extractStatus(wit)
	SprintParts := strings.Split(wit.Fields["System.IterationPath"].(string), "\\")
	wobject.Sprint = SprintParts[len(SprintParts)-1]
	wobject.Type = strings.Replace(wit.Fields["System.WorkItemType"].(string), " ", "", -1)
	return wobject, nil
}

func extractStatus(workItem azure_devops_api.WorkItem) string {
	SystemState := workItem.Fields["System.State"].(string)
	switch SystemState {
	case "New":
		return "New"
	case "Closed":
		return "Closed"
	case "Resolved":
		return "Closed"
	case "Removed":
		return "Closed"
	case "Active":
		return "Active"
	case "Blocked":
		return "Blocked"
	default:
		log.Printf("invalid State: %v, using default\n", SystemState)
		return "Blocked"
	}
}

func extractWorkerID(workItem azure_devops_api.WorkItem) string {
	var data string
	if workItem.Fields["System.AssignedTo"] != nil {
		data = workItem.Fields["System.AssignedTo"].(map[string]interface{})["uniqueName"].(string)
	} else {
		data = workItem.Fields["System.CreatedBy"].(map[string]interface{})["uniqueName"].(string)
	}

	return strings.Split(data, "@")[0]
}

func extractFloat64Int(workItem azure_devops_api.WorkItem, FieldKey string) int {
	var retVal int
	if workItem.Fields[FieldKey] == nil {
		return retVal
	}

	value, ok := workItem.Fields[FieldKey]
	if !ok {
		check(fmt.Errorf("extractFloat64Int: Was not able to Extract %v, %v, %v", FieldKey, value, workItem))
	}
	retVal, err := strconv.Atoi(strconv.FormatFloat(value.(float64), 'f', 0, 64))
	check(err)
	return retVal
}

func extractFloat64String(workItem azure_devops_api.WorkItem, FieldKey string) string {
	var retVal string
	if workItem.Fields[FieldKey] == nil {
		return retVal
	}

	value, ok := workItem.Fields[FieldKey]
	if !ok {
		check(fmt.Errorf("extractFloat64String: Was not able to Extract %v, %v, %v", FieldKey, value, workItem))
	}
	retValtmp, err := strconv.Atoi(strconv.FormatFloat(value.(float64), 'f', 0, 64))
	check(err)
	retVal = strconv.Itoa(retValtmp)
	return retVal
}

func DailyRoutineSubmit(config azure_devops_api.Configuration, inputFilePath, baseFilePath, postReportFilePath string) (err error) {
	inputWobjects := GetWobjectsFromReportFile(config, inputFilePath)
	baseWobjects := GetWobjectsFromReportFile(config, baseFilePath)

	err = CleanWobjectsUserInput(inputWobjects)
	if err != nil {
		return err
	}

	err = ValidateWobjectsUserInput(baseWobjects, inputWobjects)
	if err != nil {
		return err
	}

	wobjects := FilterChangedWobjects(baseWobjects, inputWobjects)

	requestDicts := GenerateDictsFromWobjects(wobjects)
	err = azure_devops_api.SubmitSprintStatus(config, requestDicts)
	return err
}

func GetWobjectsFromReportFile(config azure_devops_api.Configuration, filePath string) map[string]*human_api_types.Wobject {
	inputJsonFilePath := strings.Replace(filepath.Base(filePath), ".hapi", "_hapi.json", 1)

	reports, err := ConvertHRToDailyJson(filePath, inputJsonFilePath)
	check(err)

	return GenerateWobjectsFromDailyReports(config, reports)

}

func CleanWobjectsUserInput(inputWobjects map[string]*human_api_types.Wobject) error {
	for _, wobject := range inputWobjects {
		cutset := " "
		wobject.Title = strings.TrimRight(strings.TrimLeft(wobject.Title, cutset), cutset)
		wobject.WorkerID = strings.TrimRight(strings.TrimLeft(wobject.WorkerID, cutset), cutset)
		wobject.Description = strings.TrimRight(strings.TrimLeft(wobject.Description, cutset), cutset)
	}

	return nil
}

func ValidateWobjectsUserInput(baseById map[string]*human_api_types.Wobject, inputWobjects map[string]*human_api_types.Wobject) error {
	errorPrefix := "[human_api:ValidateWobjectsUserInput]"
	errors := []string{}
	for _, wobject := range inputWobjects {

		cutset := "\t\r\n"
		cutset_readable := strings.ReplaceAll(cutset, "\t", "\\t, ")
		cutset_readable = strings.ReplaceAll(cutset_readable, "\n", "\\n, ")
		cutset_readable = strings.ReplaceAll(cutset_readable, "\r", "\\r, ")

		if strings.ContainsAny(wobject.Title, cutset) {
			errors = append(errors, fmt.Sprintf("wobject title '%s' contains one of invalid characters: [%s]", wobject.Title, cutset_readable))
		}

		cutset = "\t \r\n"
		cutset_readable = strings.ReplaceAll(cutset, " ", "\\s, ")
		cutset_readable = strings.ReplaceAll(cutset_readable, "\t", "\\t, ")
		cutset_readable = strings.ReplaceAll(cutset_readable, "\n", "\\n, ")
		cutset_readable = strings.ReplaceAll(cutset_readable, "\r", "\\r, ")
		if strings.ContainsAny(wobject.WorkerID, cutset) {
			errors = append(errors, fmt.Sprintf("wobject WorkerID '%s' contains one of invalid characters: [%s]", wobject.WorkerID, cutset_readable))
		}
		if wobject.Id == "" {
			errors = append(errors, fmt.Sprintf("wobject Id is empty'%s'.>", wobject.Title))
		}

		if wobject.Id != "0" {
			_, ok := baseById[wobject.Id]
			if !ok {
				errors = append(errors, fmt.Sprintf("wobject Id '%s' from input does not exist in base file", wobject.Id))
			}
		}
		errors = append(errors, ValidateWobjectUserInput(wobject)...)
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s input Validation errors:\n%v", errorPrefix, strings.Join(errors, "\n"))
	}
	return nil
}

func ValidateWobjectUserInput(wobject *human_api_types.Wobject) (errors []string) {
	//new task/bug wobject
	if wobject.Id == "-1" {
		return errors
	}

	if len(*wobject.ChildrenIDs) == 0 {
		if wobject.Type != "Task" && wobject.Type != "Bug" {
			errors = append(errors, fmt.Sprintf("[%s][%s] - unsupported Wobject Type %s. Use one of ['Task', 'Bug']", wobject.Id, wobject.Title, wobject.Type))
		}

		//new Task/Bug
		if wobject.Id == "0" {
			if wobject.LeftTime == -1 {
				errors = append(errors, fmt.Sprintf("[%s][%s] - must provide LeftTime for new %s ", wobject.Id, wobject.Title, wobject.Type))
			}
			if wobject.InvestedTime == -1 {
				errors = append(errors, fmt.Sprintf("[%s][%s] - must provide InvestedTime for new %s ", wobject.Id, wobject.Title, wobject.Type))
			}

		}

		if wobject.LeftTime == 0 && wobject.Status != "Closed" {
			errors = append(errors, fmt.Sprintf("[%s][%s] - if LeftTime == 0, Status can not be Closed", wobject.Id, wobject.Title))
		}

	}

	return errors
}

func FilterChangedWobjects(baseById map[string]*human_api_types.Wobject, inputWobjects map[string]*human_api_types.Wobject) (wobjectsRet []*human_api_types.Wobject) {
	for _, inputWobject := range inputWobjects {
		if inputWobject.Id == "-1" {
			continue
		}

		if inputWobject.Id == "" {
			check(fmt.Errorf("filterning failed on empty wobject Id : %v", inputWobject))
		}

		//New wobject
		if inputWobject.Id == "0" {
			wobjectsRet = append(wobjectsRet, inputWobject)
			continue
		}

		baseWobject, ok := baseById[inputWobject.Id]
		if !ok {
			check(fmt.Errorf("input Wobject ID '%v' does not exist in base.haphi ", inputWobject.Id))
			continue
		}

		if inputWobject.Description == baseWobject.Description &&
			inputWobject.InvestedTime == baseWobject.InvestedTime &&
			inputWobject.LeftTime == baseWobject.LeftTime &&
			inputWobject.Status == baseWobject.Status {
			continue
		}
		wobjectsRet = append(wobjectsRet, inputWobject)
	}
	return wobjectsRet
}

func GenerateWobjectsFromDailyReports(cofig azure_devops_api.Configuration, reports []WorkerDailyReport) map[string]*human_api_types.Wobject {
	wobjectById := make(map[string]*human_api_types.Wobject)
	for _, report := range reports {
		for _, wobjectReport := range report.New {
			GenerateWobjectsFromWobjectReport(cofig, wobjectById, report.WorkerID, "New", wobjectReport)
		}

		for _, wobjectReport := range report.Active {
			GenerateWobjectsFromWobjectReport(cofig, wobjectById, report.WorkerID, "Active", wobjectReport)
		}

		for _, wobjectReport := range report.Blocked {
			GenerateWobjectsFromWobjectReport(cofig, wobjectById, report.WorkerID, "Blocked", wobjectReport)
		}
		for _, wobjectReport := range report.Closed {
			GenerateWobjectsFromWobjectReport(cofig, wobjectById, report.WorkerID, "Closed", wobjectReport)
		}
	}
	return wobjectById
}

func GenerateWobjectsFromWobjectReport(cofig azure_devops_api.Configuration, wobjectById map[string]*human_api_types.Wobject, WorkerID string, status string, wobjectReport WorkerWobjReport) {
	//{type, id, title}

	if wobjectReport.Parent[1] != "-1" {
		if _, ok := wobjectById[wobjectReport.Parent[1]]; !ok {
			wobjParent := human_api_types.Wobject{Id: wobjectReport.Parent[1],
				Title:        wobjectReport.Parent[2],
				WorkerID:     WorkerID,
				ChildrenIDs:  &[]string{},
				Priority:     -1,
				InvestedTime: -1,
				LeftTime:     -1,
				Status:       status,
				Sprint:       cofig.SprintName,
				Type:         wobjectReport.Parent[0],
				ParentID:     "-1",
			}
			wobjectById[wobjParent.Id] = &wobjParent
		}
		wobjParent := wobjectById[wobjectReport.Parent[1]]
		*wobjParent.ChildrenIDs = append(*wobjParent.ChildrenIDs, wobjectReport.Child[1])
	}

	if value, seenBefore := wobjectById[wobjectReport.Child[1]]; seenBefore {
		if value.Id == "-1" {
			return
		}
		check(fmt.Errorf("reported child wobject ID '%v' already appeared in a report with title %v", value.Id, value.Title))
	}

	var childId string

	if wobjectReport.Child[1] != "" {
		childId = wobjectReport.Child[1]
	} else {
		childId = "0"
	}

	wobj := human_api_types.Wobject{Id: childId,
		Title:        wobjectReport.Child[2],
		WorkerID:     WorkerID,
		ChildrenIDs:  &[]string{},
		Priority:     -1,
		Status:       status,
		Sprint:       cofig.SprintName,
		InvestedTime: wobjectReport.InvestedTime,
		LeftTime:     wobjectReport.LeftTime,
		Description:  wobjectReport.Comment,
		Type:         wobjectReport.Child[0],
		ParentID:     wobjectReport.Parent[1],
	}

	wobjectById[wobj.Id] = &wobj
}

func GenerateDictsFromWobjects(wobjects []*human_api_types.Wobject) (lstRet [](*map[string]string)) {
	for _, wobject := range wobjects {
		dictRequest := make(map[string]string)

		var err error

		if _, err = strconv.Atoi(wobject.ParentID); err != nil {
			check_ng(fmt.Sprintf("wobject [%s] [%s] ParentID:", wobject.Id, wobject.Title), err)
		}

		dictRequest["Id"] = wobject.Id
		dictRequest["ParentID"] = wobject.ParentID
		dictRequest["Priority"] = GuessPriorityForRequestDict(*wobject)
		dictRequest["Title"] = wobject.Title
		dictRequest["Description"] = wobject.Description
		dictRequest["LeftTime"] = strconv.Itoa(wobject.LeftTime)
		dictRequest["InvestedTime"] = strconv.Itoa(wobject.InvestedTime)
		dictRequest["WorkerID"] = wobject.WorkerID
		dictRequest["ChildrenIDs"] = strings.Join(*wobject.ChildrenIDs, ",")
		dictRequest["Sprint"] = wobject.Sprint
		dictRequest["Status"] = wobject.Status
		dictRequest["Type"] = wobject.Type

		lstRet = append(lstRet, &dictRequest)

	}

	return lstRet
}

func GuessPriorityForRequestDict(wobject human_api_types.Wobject) string {
	if wobject.Priority != -1 {
		return strconv.Itoa(wobject.Priority)
	}

	if wobject.Id != "0" {
		return "-1"
	}

	if wobject.Status == "Active" {
		return "1"
	}

	return "2"
}

func loadConfiguration(filePath string) (config Configuration, err error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func DownloadAllWits(config azure_devops_api.Configuration, dstFilePath string) (err error) {
	log.Printf("downloadAllWits: %v, %v\n", config, dstFilePath)
	err = azure_devops_api.DownloadAllWits(config, dstFilePath)
	return err
}

// Return True if exists, False if not or fails on error.
func checkFileExists(path string) (exists bool) {
	_, err := os.Stat(path)
	if err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	}

	log.Fatalf("Failed checking file exists: %v", err)
	return false
}

func copyFile(srcFilePath, dstFilePath string) error {
	// Open the source file for reading
	srcFile, err := os.Open(srcFilePath)
	if err != nil {
		fmt.Println("Error opening source file:", err)
		return err
	}
	defer srcFile.Close()

	// Create the destination file (with 0644 permissions)
	dstFile, err := os.Create(dstFilePath)
	if err != nil {
		fmt.Println("Error creating destination file:", err)
		return err
	}
	defer dstFile.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		fmt.Println("Error copying file:", err)
		return err
	}

	return nil
}

func logWithLineNumber(message string) {
	// Get the caller's file name and line number
	_, file, line, ok := runtime.Caller(1) // 1 skips the current function
	if !ok {
		file = "???"
		line = 0
	}

	// Format the log message with line number
	logMessage := fmt.Sprintf("%s:%d: %s", file, line, message)

	// Print the log message
	fmt.Println(logMessage)
}

type HumanAPIConfiguration struct {
	ApplicationRootDiriectoryPath string `json:"ApplicationRootDiriectoryPath,omitempty"`
	TicketDefaultValuesFilePath   string `json:"TicketDefaultValuesFilePath,omitempty"`
	WorkerName                    string `json:"WorkerName,omitempty"`
	DailiesReportsPath            string `json:"DailiesReportsPath,omitempty"`
}

type HumanAPI struct {
	Configuration     *HumanAPIConfiguration
	ProjectManagerAPI *human_api_types.ProjectManager
}

func WithProjectManagerAPI(ProjectManagerAPI human_api_types.ProjectManager) func(api config_pol.Configurable, APIConfiguration any) error {
	return func(api config_pol.Configurable, APIConfiguration any) error {
		humanAPI, ok := api.(*HumanAPI)
		if !ok {
			return fmt.Errorf("%v not HumanAPI", api)
		}

		humanAPI.ProjectManagerAPI = &ProjectManagerAPI
		err := api.SetConfiguration(APIConfiguration)
		if err != nil {
			return err
		}
		return nil
	}
}

func HumanAPINew(options ...config_pol.Option) (*HumanAPI, error) {
	humanAPI := &HumanAPI{}
	config := &HumanAPIConfiguration{}
	for _, option := range options {
		option(humanAPI, config)
	}

	humanAPI.initConfigDefaults()

	return humanAPI, nil
}

/*
ApplicationRootDiriectoryPath string `json:"ApplicationRootDiriectoryPath,omitempty"`

	TicketDefaultValuesFilePath   string `json:"TicketDefaultValuesFilePath,omitempty"`
	WorkerName                    string `json:"WorkerName,omitempty"`
	DailiesReportsPath            string `json:"DailiesReportsPath,omitempty"`
*/
func (humanAPI *HumanAPI) initConfigDefaults() {

	if humanAPI.Configuration.ApplicationRootDiriectoryPath == "" {
		humanAPI.Configuration.ApplicationRootDiriectoryPath = "/opt/human_api"
	}

	if humanAPI.Configuration.DailiesReportsPath == "" {
		humanAPI.Configuration.DailiesReportsPath = filepath.Join(humanAPI.Configuration.ApplicationRootDiriectoryPath, "daily")
	}

}

func (humanAPI *HumanAPI) TicketAction() error {

	return nil

}

func (humanAPI *HumanAPI) CreateTicket(Type, Title, Description, WorkerName string, Priority int) error {

	Worker, err := humanAPI.GetWorker(&WorkerName)
	if err != nil {
		return err
	}

	WorkerSprint, err := humanAPI.GetWorkerSprint(Worker)
	if err != nil {
		return err
	}

	wobj := &human_api_types.Wobject{
		Priority:    Priority,
		Type:        Type,
		Title:       Title,
		Description: Description,
		WorkerID:    (*Worker).Id,
		Sprint:      (*WorkerSprint).Id,
	}

	humanAPI.ProvisionWobject(wobj)
	return nil

}

func (humanAPI *HumanAPI) GetWorker(Name *string) (*human_api_types.Worker, error) {
	worker, err := (*humanAPI.ProjectManagerAPI).GetWorker(Name)
	if err != nil {
		return nil, err
	}
	return worker, nil
}
func (humanAPI *HumanAPI) GetWorkerSprint(Worker *human_api_types.Worker) (*human_api_types.Sprint, error) {
	baseError := "[human_api->GetWorkerSprint]"
	sprint, err := (*humanAPI.ProjectManagerAPI).GetWorkerSprint(Worker)
	if err != nil {
		return nil, fmt.Errorf("%s error fetching worker sprint from ProjectManagerAPI\n %v", baseError, err)
	}
	return sprint, nil
}

func (humanAPI *HumanAPI) SetConfiguration(Config any) error {
	HumanConfig, ok := Config.(*HumanAPIConfiguration)
	if !ok {
		return fmt.Errorf("was not able to convert %v to HumanAPIConfig", Config)
	}
	humanAPI.Configuration = HumanConfig
	return nil
}

func (humanAPI *HumanAPI) ProvisionWobject(WorkObject *human_api_types.Wobject) error {
	err := (*humanAPI.ProjectManagerAPI).ProvisionWobject(WorkObject)
	if err != nil {
		return err
	}
	return nil

}

type DailyConfig struct {
	Sprint           *human_api_types.Sprint `json:"Sprint"`
	DailyDirectory   string                  `json:"DailyDirectory"`
	Worker           *human_api_types.Worker `json:"Worker"`
	ReportFilePath   string                  `json:"ReportFilePath"`
	InputFilePath    string                  `json:"InputFilePath"`
	OutputFilePath   string                  `json:"OutputFilePath"`
	WobjectsFilePath string                  `json:"WobjectsFilePath"`
}

func (humanAPI *HumanAPI) DailyConfigNew(worker *human_api_types.Worker) (*DailyConfig, error) {
	dailyConfg := &DailyConfig{}
	sprint, err := humanAPI.GetWorkerSprint(worker)
	if err != nil {
		return nil, fmt.Errorf("[humanAPI->DailyConfigNew] Error getting worker sprint\n %v", err)
	}
	dailyConfg.Sprint = sprint
	dailyConfg.Worker = worker

	sprintDirPath := filepath.Join(humanAPI.Configuration.DailiesReportsPath, dailyConfg.Sprint.Name)

	now := time.Now()
	dateDirName := now.Format("2006_01_02")
	dateDirPath := filepath.Join(sprintDirPath, dateDirName)

	workerNameDirName := strings.Replace(worker.Name, " ", "_", -1)
	dailyConfg.DailyDirectory = filepath.Join(dateDirPath, workerNameDirName)

	fmt.Println("Generated new directory path: " + dateDirPath)

	err = os.MkdirAll(dailyConfg.DailyDirectory, 0755)

	dailyConfg.ReportFilePath = filepath.Join(dailyConfg.DailyDirectory, "report.hapi")
	dailyConfg.InputFilePath = filepath.Join(dailyConfg.DailyDirectory, "input.hapi")
	dailyConfg.OutputFilePath = filepath.Join(dailyConfg.DailyDirectory, "YTB.hapi")
	dailyConfg.WobjectsFilePath = filepath.Join(dailyConfg.DailyDirectory, "wobjects.json")

	return dailyConfg, err
}

func (humanAPI *HumanAPI) FetchDaily(worker *human_api_types.Worker) (err error) {
	errorPrefix := "[human_api:FetchDaily]"
	dailyConfig, err := humanAPI.DailyConfigNew(worker)
	if err != nil {
		return fmt.Errorf("%s Initializing DailyConfigNew\n%v", errorPrefix, err)
	}

	if !checkFileExists(dailyConfig.InputFilePath) {
		err = humanAPI.GenerateDailyReport(dailyConfig)
		if err != nil {
			return fmt.Errorf("%s Generating daily report\n%v", errorPrefix, err)
		}
		err = copyFile(dailyConfig.ReportFilePath, dailyConfig.InputFilePath)
		if err != nil {
			return fmt.Errorf("%s Error Copying report file to input file\n%v", errorPrefix, err)
		}
	}

	if !checkFileExists(dailyConfig.WobjectsFilePath) {
		return fmt.Errorf("%s Wobjects file is missing\n%v", errorPrefix, err)
	}

	fmt.Printf("SUCCESS!! %s\n", dailyConfig.InputFilePath)
	return nil
}

func (humanAPI *HumanAPI) GenerateDailyReport(dailyConfig *DailyConfig) error {
	errorPrefix := "[human_api:GenerateDailyReport]"

	if !checkFileExists(dailyConfig.WobjectsFilePath) {
		err := humanAPI.DownloadDailySprintWobjects(dailyConfig)
		if err != nil {
			return fmt.Errorf("%s Downloading daily sprint Wobjects\n%v", errorPrefix, err)
		}
	}

	wobjects, err := humanAPI.LoadWobjectsFromFile(dailyConfig.WobjectsFilePath)
	if err != nil {
		return fmt.Errorf("%s Marshaling wobjects\n%v", errorPrefix, err)
	}
	wobjects["-1"] = &human_api_types.Wobject{Id: "-1", Title: "AutoGenerated", Type: "UserStory"}

	new := []WorkerWobjReport{}
	active := []WorkerWobjReport{}
	blocked := []WorkerWobjReport{}
	closed := []WorkerWobjReport{}

	reports := []WorkerDailyReport{}
	var workerID string
	for wobjid, wobject := range wobjects {
		if wobjid == "-1" {
			continue
		}
		var parentPointer *human_api_types.Wobject
		var childPointer *human_api_types.Wobject

		if len(*wobject.ChildrenIDs) != 0 {
			log.Printf("skipping Wobject with children: %v\n", wobjid)
			continue
		}

		parentPointer, childPointer, err = humanAPI.GenerateParentAndChildForReport(wobject, wobjects)
		if err != nil {
			return fmt.Errorf("%s Desiding parent and child order\n%v", errorPrefix, err)
		}

		workerID = wobject.WorkerID

		report := WorkerWobjReport{Parent: []string{parentPointer.Type, parentPointer.Id, parentPointer.Title},
			Child: []string{childPointer.Type, childPointer.Id, childPointer.Title}}
		switch wobject.Status {
		case "New":
			new = append(new, report)
		case "Closed":
			closed = append(closed, report)
		case "Active":
			active = append(active, report)
		case "Blocked":
			blocked = append(blocked, report)
		default:
			return fmt.Errorf("%s invalid wobject.Status: %v", errorPrefix, wobject.Status)
		}
	}

	if len(new) == 0 {
		return fmt.Errorf("%s new wobjects are empty: %v", errorPrefix, new)
	}

	workerDailyReport := WorkerDailyReport{WorkerID: workerID,
		New:     new,
		Active:  active,
		Blocked: blocked,
		Closed:  closed,
	}
	reports = append(reports, workerDailyReport)
	WriteDailyToHRFile(reports, dailyConfig.ReportFilePath)
	return nil
}

// Download the Wobjects and write them to file
func (humanAPI *HumanAPI) DownloadDailySprintWobjects(dailyConfig *DailyConfig) error {
	errorPrefix := "[human_api:DownloadDailySprintWobjects]"
	if dailyConfig.Sprint == nil {
		return fmt.Errorf("%s Sprint is nil", errorPrefix)
	}

	wobjectSlice, err := (*humanAPI.ProjectManagerAPI).GetWorkerSprintWobjects(dailyConfig.Sprint, dailyConfig.Worker)
	if err != nil {
		return fmt.Errorf("%s Fetching worker sprint wobjects\n%v", errorPrefix, err)
	}
	wobjects := map[string]*human_api_types.Wobject{}
	for _, wobject := range wobjectSlice {
		wobjects[wobject.Id] = wobject
	}

	jsonData, err := json.MarshalIndent(wobjects, "", "  ")
	if err != nil {
		return fmt.Errorf("%s Marshaling wobjects\n%v", errorPrefix, err)
	}

	err = os.WriteFile(dailyConfig.WobjectsFilePath, jsonData, 0644) // 0644 are file permissions
	if err != nil {
		return fmt.Errorf("%s Error writing to file\n%v", errorPrefix, err)
	}
	return nil

}

// Generate Parent and child for Wobject that has not explicit parent.
// The wobject can become either Parent from new qobject or a Child with undefind (-1) Parent
func (humanAPI *HumanAPI) GenerateParentAndChildForReport(wobject *human_api_types.Wobject, wobjectsRelevant map[string]*human_api_types.Wobject) (parent, child *human_api_types.Wobject, err error) {
	if wobject.Type == "Task" || wobject.Type == "Bug" {

		if wobject.ParentID == "" {
			wobject.ParentID = "-1"
		}

		parentWobject := wobjectsRelevant[wobject.ParentID]
		if parentWobject == nil {
			return nil, nil, fmt.Errorf("wobj id %s is missing", wobject.ParentID)
		}

		if wobjectsRelevant[wobject.ParentID].ChildrenIDs == nil {
			wobjectsRelevant[wobject.ParentID].ChildrenIDs = new([]string)
		}
		*(wobjectsRelevant[wobject.ParentID].ChildrenIDs) = append(*(wobjectsRelevant[wobject.ParentID].ChildrenIDs), wobject.Id)
		parent = wobjectsRelevant[wobject.ParentID]
		child = wobject
	} else {
		wobject.ChildrenIDs = &[]string{"-1"}
		child = wobjectsRelevant["-1"]
		parent = wobject
	}

	return parent, child, nil
}

func (humanAPI *HumanAPI) PushDaily(worker *human_api_types.Worker) (err error) {
	errorPrefix := "[human_api:PushDaily]"
	dailyConfig, err := humanAPI.DailyConfigNew(worker)
	if err != nil {
		return fmt.Errorf("%s Initializing DailyConfigNew\n%v", errorPrefix, err)
	}

	if !checkFileExists(dailyConfig.ReportFilePath) || !checkFileExists(dailyConfig.InputFilePath) {
		return fmt.Errorf("%s, pre report or input file does not exist '%v'", errorPrefix, dailyConfig.InputFilePath)
	}

	inputWobjects, err := humanAPI.LoadWobjectsFromReport(dailyConfig, dailyConfig.InputFilePath)
	if err != nil {
		return fmt.Errorf("%s Loading input wobjects from report \n%v", errorPrefix, err)
	}
	baseWobjects, err := humanAPI.LoadWobjectsFromReport(dailyConfig, dailyConfig.ReportFilePath)
	if err != nil {
		return fmt.Errorf("%s Loading report wobjects from report \n%v", errorPrefix, err)
	}

	err = CleanWobjectsUserInput(inputWobjects)
	if err != nil {
		return fmt.Errorf("%s Cleaning wobjects input \n%v", errorPrefix, err)
	}
	err = ValidateWobjectsUserInput(baseWobjects, inputWobjects)
	if err != nil {
		return fmt.Errorf("%s Validating wobjects input\n%v", errorPrefix, err)
	}

	wobjects := FilterChangedWobjects(baseWobjects, inputWobjects)
	humanAPI.ProvisionDailyWobjects(dailyConfig, wobjects)
	//requestDicts := GenerateDictsFromWobjects(wobjects)
	//err = azure_devops_api.SubmitSprintStatus(config, requestDicts)
	return nil
}

func (humanAPI *HumanAPI) LoadWobjectsFromFile(filePath string) (map[string]*human_api_types.Wobject, error) {
	errorPrefix := "[human_api:LoadWobjectsFromFile]"
	wobjects := map[string]*human_api_types.Wobject{}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%s Opening file\n%v", errorPrefix, err)
	}
	err = json.Unmarshal(data, &wobjects)
	if err != nil {
		return nil, fmt.Errorf("%s Unmarshallng file\n%v", errorPrefix, err)
	}

	return wobjects, nil
}

func (humanAPI *HumanAPI) ProvisionDailyWobjects(dailyConfig *DailyConfig, wobjects []*human_api_types.Wobject) error {
	errorPrefix := "[human_api:ProvisionDailyWobjects]"
	currentWobjects, err := humanAPI.LoadWobjectsFromFile(dailyConfig.WobjectsFilePath)
	if err != nil {
		return fmt.Errorf("%s Loading cached wobjects from file %s\n%v", errorPrefix, dailyConfig.WobjectsFilePath, err)

	}

	for _, wobject := range wobjects {
		if wobject.Id == "0" {
			err = (*humanAPI.ProjectManagerAPI).ProvisionWobject(wobject)
			if err != nil {
				return fmt.Errorf("%s Creating wobject %s\n%v", errorPrefix, dailyConfig.WobjectsFilePath, err)

			}
		}

		currentWobject := currentWobjects[wobject.Id]
		if wobject.LeftTime != -1 {
			currentWobject.LeftTime = wobject.LeftTime
		}
		if wobject.InvestedTime != -1 {
			currentWobject.InvestedTime += wobject.InvestedTime
		}
		if wobject.Description != "" {
			currentWobject.Description = wobject.Description
		}
		if currentWobject.Status != wobject.Status {
			currentWobject.Status = wobject.Status
		}
		(*humanAPI.ProjectManagerAPI).UpdateWobject(currentWobject)
	}

	return nil
}

func (humanAPI *HumanAPI) LoadWobjectsFromReport(dailyConfig *DailyConfig, filePath string) (map[string]*human_api_types.Wobject, error) {
	errorPrefix := "[human_api:LoadWobjectsFromReport]"
	reports, err := ReadDailyFromHRFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%s Reading daily file\n%v", errorPrefix, err)
	}
	wobjects, err := humanAPI.GenerateWobjectsFromDailyReports(dailyConfig, reports)
	if err != nil {
		return nil, fmt.Errorf("%s Converting Report to Wobjects\n%v", errorPrefix, err)
	}
	return wobjects, nil
}

func (humanAPI *HumanAPI) GenerateWobjectsFromDailyReports(dailyConfig *DailyConfig, reports []WorkerDailyReport) (map[string]*human_api_types.Wobject, error) {
	errr := common_utils.ErrorCreator("[human_api:]GenerateWobjectsFromDailyReports")
	wobjectById := make(map[string]*human_api_types.Wobject)
	for _, report := range reports {
		for _, wobjectReport := range report.New {
			err := humanAPI.GenerateWobjectsFromWobjectReport(dailyConfig, wobjectById, "New", wobjectReport)
			if err != nil {
				return nil, errr("Generating Wobjects from report 'New'", err)
			}
		}

		for _, wobjectReport := range report.Active {
			err := humanAPI.GenerateWobjectsFromWobjectReport(dailyConfig, wobjectById, "Active", wobjectReport)
			if err != nil {
				return nil, errr("Generating Wobjects from report 'Active'", err)
			}
		}

		for _, wobjectReport := range report.Blocked {
			err := humanAPI.GenerateWobjectsFromWobjectReport(dailyConfig, wobjectById, "Blocked", wobjectReport)
			if err != nil {
				return nil, errr("Generating Wobjects from report 'Blocked'", err)
			}
		}
		for _, wobjectReport := range report.Closed {
			err := humanAPI.GenerateWobjectsFromWobjectReport(dailyConfig, wobjectById, "Closed", wobjectReport)
			if err != nil {
				return nil, errr("Generating Wobjects from report 'Closed'", err)
			}
		}
	}
	return wobjectById, nil
}

func (humanAPI *HumanAPI) GenerateWobjectsFromWobjectReport(dailyConfg *DailyConfig, wobjectById map[string]*human_api_types.Wobject, status string, wobjectReport WorkerWobjReport) error {

	//real parent
	if wobjectReport.Parent[1] != "-1" {
		wobjParent, ok := wobjectById[wobjectReport.Parent[1]]
		if !ok {
			wobjParent = &human_api_types.Wobject{Id: wobjectReport.Parent[1],
				Title:        wobjectReport.Parent[2],
				WorkerID:     dailyConfg.Worker.Id,
				ChildrenIDs:  &[]string{},
				Priority:     -1,
				InvestedTime: -1,
				LeftTime:     -1,
				Status:       status,
				Sprint:       dailyConfg.Sprint.Id,
				Type:         wobjectReport.Parent[0],
				ParentID:     "-1",
			}
			wobjectById[wobjParent.Id] = wobjParent
		}
		*wobjParent.ChildrenIDs = append(*wobjParent.ChildrenIDs, wobjectReport.Child[1])
	}

	if value, seenBefore := wobjectById[wobjectReport.Child[1]]; seenBefore {
		if value.Id == "-1" {
			return nil
		}
		return fmt.Errorf("reported child wobject ID '%v' already appeared in a report with title %v", value.Id, value.Title)
	}

	var childId string

	if wobjectReport.Child[1] != "" {

		childId = wobjectReport.Child[1]
	} else {
		childId = "0"
	}

	wobj := human_api_types.Wobject{Id: childId,
		Title:        wobjectReport.Child[2],
		WorkerID:     dailyConfg.Worker.Id,
		ChildrenIDs:  &[]string{},
		Priority:     -1,
		Status:       status,
		Sprint:       dailyConfg.Sprint.Id,
		InvestedTime: wobjectReport.InvestedTime,
		LeftTime:     wobjectReport.LeftTime,
		Description:  wobjectReport.Comment,
		Type:         wobjectReport.Child[0],
		ParentID:     wobjectReport.Parent[1],
	}

	wobjectById[wobj.Id] = &wobj
	return nil
}
