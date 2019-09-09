package errors

import (
	"github.com/pkg/errors"
)

var (
	// Generic errors
	UnexpectedError = errors.New(`unexpected error`)

	// State errors
	ErrUnableToCreateStateKey             = errors.New(`unable to create state key`)
	ErrUnableToCreateEventName            = errors.New(`unable to create event name`)
	ErrKeyAlreadyExists                   = errors.New(`state key already exists`)
	ErrKeyNotFound                        = errors.New(`state entry not found`)
	ErrAllowOnlyOneValue                  = errors.New(`allow only one value`)
	ErrStateEntryNotSupportKeyerInterface = errors.New(`state entry not support keyer interface`)
	ErrEventEntryNotSupportNamerInterface = errors.New(`event entry not support name interface`)
	ErrKeyPartsLength                     = errors.New(`key parts length must be greater than zero`)
	SetGetError                           = errors.New(`set/get error`)
	NoQuerySelectorError                  = errors.New(`no selector provided for rich query`)
	InvalidSortQueryError                 = errors.New(`invalid syntax for sort query`)

	// Router errors
	ErrEmptyArgs       = errors.New(`empty args`)
	ErrMethodNotFound  = errors.New(`chaincode method not found`)
	ErrArgsNumMismatch = errors.New(`chaincode method args count mismatch`)
	ErrHandlerError    = errors.New(`router handler error`)

	// Identity errors
	CertificateError = errors.New(`certificate error`)
)

// GetErrorMappings - returns a map of all errors
func GetErrorMappings() map[error]int32 {
	return errMap
}

var errMap = map[error]int32{
	// Generic errors
	UnexpectedError: 599,

	// State errors
	ErrUnableToCreateStateKey:             599,
	ErrUnableToCreateEventName:            599,
	ErrKeyAlreadyExists:                   401,
	ErrKeyNotFound:                        402,
	ErrAllowOnlyOneValue:                  599,
	ErrStateEntryNotSupportKeyerInterface: 599,
	ErrEventEntryNotSupportNamerInterface: 599,
	ErrKeyPartsLength:                     599,
	SetGetError:                           500,
	NoQuerySelectorError:                  400,
	InvalidSortQueryError:                 400,

	// Router errors
	ErrEmptyArgs:       400,
	ErrMethodNotFound:  404,
	ErrArgsNumMismatch: 400,
	ErrHandlerError:    599,

	// Identity errors
	CertificateError: 400,
}
