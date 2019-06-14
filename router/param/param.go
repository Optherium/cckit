package param

import (
	"github.com/optherium/cckit/convert"
	"github.com/optherium/cckit/router"
)

// String creates middleware for converting to string chaincode method parameter
func String(name string, argPoss ...int) router.MiddlewareFunc {
	return Param(name, convert.TypeString, argPoss...)
}

func Strings(name string, argPoss ...int) router.MiddlewareFunc {
	return Param(name, []string{}, argPoss...)
}

// Int creates middleware for converting to integer chaincode method parameter
func Int(name string, argPoss ...int) router.MiddlewareFunc {
	return Param(name, convert.TypeInt, argPoss...)
}

// Bool creates middleware for converting to bool chaincode method parameter
func Bool(name string, argPoss ...int) router.MiddlewareFunc {
	return Param(name, convert.TypeBool, argPoss...)
}

// Struct creates middleware for converting to struct chaincode method parameter
func Struct(name string, target interface{}, argPoss ...int) router.MiddlewareFunc {
	return Param(name, target, argPoss...)
}

// Bytes creates middleware for converting to []byte chaincode method parameter
func Bytes(name string, argPoss ...int) router.MiddlewareFunc {
	return Param(name, []byte{}, argPoss...)
}

// StrictKnown allows passing arguments to chaincode func only if parameters are defined in router
func StrictKnown(next router.HandlerFunc, pos ...int) router.HandlerFunc {
	return func(c router.Context) (interface{}, error) {
		return next(c)
	}
}
