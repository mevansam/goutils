package config

import (
	"github.com/mevansam/goutils/data/entry"
)

type Configurable interface {
	// the title of the configuration
	Name() string
	// the description of the configuration
	Description() string

	// the configuration form
	InputForm() (entry.InputForm, error)

	// retrieves the value of a key in the form. this will search
	// all groups within the form for the given key and will return
	// the value of the first input with the given key.
	GetValue(name string) (*string, error)

	// Get a copy of this Configurable instance
	Copy() (Configurable, error)

	// reset all configuration values to their defaults
	Reset()
}
