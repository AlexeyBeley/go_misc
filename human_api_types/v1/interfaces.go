package human_api_types

type ProjectManager interface {
	ProvisionWobject(*Wobject) error
	GetWorker(Name *string) (*Worker, error)
	GetWorkerSprint(*Worker) (*Sprint, error)
	GetWorkerSprintWobjects(sprint *Sprint, worker *Worker) ([]*Wobject, error)
	UpdateWobject(*Wobject) error
}
