package human_api_types

import (
	types "github.com/AlexeyBeley/go_misc/human_api_types"
)

type ProjectManager interface {
	ProvisionWobject(types.Wobject) error
}
