package human_api_types

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
}


type Worker struct {
	Id           string    `json:"Id"`
	Name        string    `json:"Name"`
	SystemName string
}

type Sprint struct{
	Id string `json:"Id"`
}