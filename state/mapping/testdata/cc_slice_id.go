package testdata

import (
	"github.com/optherium/cckit/extensions/debug"
	"github.com/optherium/cckit/extensions/owner"
	"github.com/optherium/cckit/router"
	"github.com/optherium/cckit/router/param"
	"github.com/optherium/cckit/router/param/defparam"
	"github.com/optherium/cckit/state"
	m "github.com/optherium/cckit/state/mapping"
	"github.com/optherium/cckit/state/mapping/testdata/schema"
)

func NewSliceIdCC() *router.Chaincode {
	r := router.New(`complexId`)
	debug.AddHandlers(r, `debug`, owner.Only)

	// Mappings for chaincode state
	r.Use(m.MapStates(m.StateMappings{}.
		//key will be <`EntityWithSliceId`, {Id[0]}, {Id[1]},... {Id[len(Id)-1]} >
		Add(&schema.EntityWithSliceId{}, m.PKeyId())))

	r.Init(owner.InvokeSetFromCreator)

	r.Group(`entity`).
		Invoke(`List`, func(c router.Context) (interface{}, error) {
			return c.State().List(&schema.EntityWithSliceId{})
		}).
		Invoke(`Get`, func(c router.Context) (interface{}, error) {
			return c.State().Get(&schema.EntityWithSliceId{Id: state.StringsIdFromStr(c.ParamString(`Id`))})
		}, param.String(`Id`)).
		Invoke(`Insert`, func(c router.Context) (interface{}, error) {
			return nil, c.State().Insert(c.Param())
		}, defparam.Proto(&schema.EntityWithSliceId{}))

	return router.NewChaincode(r)
}
