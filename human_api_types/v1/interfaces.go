package human_api_types

type ProjectManager interface {
	ProvisionWobject(*Wobject) error
	GetWorker(Name *string) (*Worker, error)
}
