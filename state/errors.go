package state

import (
	"github.com/pkg/errors"
)

var (
	// creating composite key for entry
	UnableToCreateKeyError = errors.New(`unable to create state key`)

	// insert or Put with more than 2 arguments
	AllowOnlyOneValueError = errors.New(`allow only one value`)

	//insert or Put struct without providing key and struct not support Keyer interface
	KeyNotSupportKeyerInterfaceError = errors.New(`key not support keyer interface`)

	//create key consisting of zero parts
	KeyPartsLengthError = errors.New(`key parts length must be greater than zero`)

	UnExpectedError = errors.New(`unexpected Error`)

	SetGetError = errors.New(`set/get Error`)

	AlreadyExistsError = errors.New(`state key already exists`)

	KeyNotFoundError = errors.New(`state entry not found`)

	// Rich query related errors
	NoQuerySelectorError = errors.New(`No selector provided for rich query`)

	InvalidSortQueryError = errors.New(`Invalid syntax for sort query`)
)
