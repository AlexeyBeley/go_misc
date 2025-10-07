package human_api_types

import (
	"strconv"
	"strings"
	"time"
)

type Wobject struct {
	Id           string    `json:"Id"`
	Title        string    `json:"Title"`
	Description  string    `json:"Description"`
	LeftTime     int       `json:"LeftTime"`
	InvestedTime int       `json:"InvestedTime"`
	WorkerID     string    `json:"WorkerID"`
	ChildrenIDs  *[]string `json:"ChildrenIDs"`
	ParentID     string    `json:"ParentID"`
	Priority     int       `json:"Priority"`
	Status       string    `json:"Status"`
	Sprint       string    `json:"Sprint"`
	Type         string    `json:"Type"`
	Link         string    `json:"Link"`
}

func (wobject *Wobject) GuessPriorityForRequestDict() string {
	if wobject.Priority != -1 {
		return strconv.Itoa(wobject.Priority)
	}

	if !strings.HasPrefix(wobject.Id, "CreatePlease:") {
		return "-1"
	}

	if wobject.Status == "Active" {
		return "1"
	}

	return "2"
}

func (wobject *Wobject) ConverttotMap() (map[string]string, error) {
	ret := map[string]string{}
	if wobject.Id != "" {
		ret["Id"] = wobject.Id
	}
	if wobject.ParentID != "" {
		ret["ParentID"] = wobject.ParentID
	}
	ret["Priority"] = wobject.GuessPriorityForRequestDict()
	ret["Title"] = wobject.Title
	ret["Description"] = wobject.Description
	ret["LeftTime"] = strconv.Itoa(wobject.LeftTime)
	ret["InvestedTime"] = strconv.Itoa(wobject.InvestedTime)
	ret["WorkerID"] = wobject.WorkerID
	if wobject.ChildrenIDs != nil && len(*wobject.ChildrenIDs) > 0 {
		ret["ChildrenIDs"] = strings.Join(*wobject.ChildrenIDs, ",")
	}
	if wobject.Sprint != "" {
		ret["Sprint"] = wobject.Sprint
	}
	ret["Status"] = wobject.Status
	ret["Type"] = wobject.Type
	if wobject.Link != "" {
		ret["Link"] = wobject.Link
	}

	return ret, nil
}

type Worker struct {
	Id         string `json:"Id"`
	Name       string `json:"Name"`
	SystemName string
}

type Sprint struct {
	Id        string    `json:"Id"`
	Name      string    `json:"Name"`
	DateStart time.Time `json:"DateStart"`
	DateEnd   time.Time `json:"DateEnd"`
}
