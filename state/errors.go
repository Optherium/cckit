package state

import (
	"github.com/pkg/errors"
)

var (
	// ErrUnableToCreateStateKey can occurs while creating composite key for entry
	ErrUnableToCreateStateKey = errors.New(`unable to create state key`)

	// ErrUnableToCreateEventName can occurs while creating composite key for entry
	ErrUnableToCreateEventName = errors.New(`unable to create event name`)

	// ErrKeyAlreadyExists can occurs when trying to insert entry with existing key
	ErrKeyAlreadyExists = errors.New(`state key already exists`)

	// ErrKeyNotFound key not found in chaincode state
	ErrKeyNotFound = errors.New(`state entry not found`)

	// ErrAllowOnlyOneValue can occurs when trying to call Insert or Put with more than 2 arguments
	ErrAllowOnlyOneValue = errors.New(`allow only one value`)

	// ErrStateEntryNotSupportKeyerInterface can occurs when trying to Insert or Put struct
	// providing key and struct without Keyer interface support
	ErrStateEntryNotSupportKeyerInterface = errors.New(`state entry not support keyer interface`)

	ErrEventEntryNotSupportNamerInterface = errors.New(`event entry not support name interface`)

	// ErrKeyPartsLength can occurs when trying to create key consisting of zero parts
	ErrKeyPartsLength = errors.New(`key parts length must be greater than zero`)

	UnexpectedError = errors.New(`unexpected error`)

	SetGetError = errors.New(`set/get error`)

	KeyNotFoundError = errors.New("state entry not found")

	AlreadyExistsError = errors.New(`state key already exists`)

	// Rich query related errors
	NoQuerySelectorError = errors.New(`No selector provided for rich query`)

	InvalidSortQueryError = errors.New(`Invalid syntax for sort query`)
)
