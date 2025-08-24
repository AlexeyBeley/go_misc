package human_api_types

type ProjectManager interface {
	ProvisionWobject(*Wobject) error
	GetWorkerId(Name *string) (*string, error)
}
