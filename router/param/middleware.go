package param

import (
	"github.com/optherium/cckit/router"
)

// StrictKnown allows passing arguments to chaincode func only if parameters are defined in router
func StrictKnown(next router.HandlerFunc, pos ...int) router.HandlerFunc {
	return func(c router.Context) (interface{}, error) {
		return next(c)
	}
}
