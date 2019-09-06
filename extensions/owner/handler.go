package owner

import (
	"github.com/optherium/cckit/router"
)

const QueryMethod = `owner`

// FromState returns raw data ( serialized Grant ) of current chain code owner
func Query(c router.Context) (interface{}, error) {
	return c.State().Get(OwnerStateKey)
}

// InvokeSetFromCreator sets tx creator as chaincode owner, if owner not previously setted
func InvokeSetFromCreator(c router.Context) (interface{}, error) {
	return SetFromCreator(c)
}

// InvokeSetFromArgs gets owner data fron args[0] (Msp Id) and arg[1] (cert)
func InvokeSetFromArgs(c router.Context) (interface{}, error) {
	return SetFromArgs(c)
}
