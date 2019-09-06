package defparam

import (
	"github.com/optherium/cckit/router"
	"github.com/optherium/cckit/router/param"
)

func Proto(target interface{}, argPoss ...int) router.MiddlewareFunc {
	return param.Proto(router.DefaultParam, target, argPoss...)
}
