package mocks

import (
	"github.com/mevansam/goutils/config"
	"github.com/mevansam/goutils/forms"

	. "github.com/onsi/gomega"
)

type FakeConfig struct {
	inputGroup *forms.InputGroup
	values     map[string]*valueRef
}

type valueRef struct {
	value *string
}

func (f *FakeConfig) InitConfig(name, description string) {

	f.inputGroup = forms.
		NewInputCollection().
		NewGroup(name, description)
	f.values = make(map[string]*valueRef)
}

func (f *FakeConfig) AddInputField(
	name,
	displayName,
	description,
	defaultValue string,
	envVars []string,
) {

	var (
		err error

		field forms.Input
	)

	if len(defaultValue) == 0 {
		field, err = f.inputGroup.NewInputField(
			name,
			displayName,
			description,
			forms.String,
			false,
			envVars,
			[]string{},
		)
		Expect(err).NotTo(HaveOccurred())

	} else {
		field, err = f.inputGroup.NewInputFieldWithDefaultValue(
			name,
			displayName,
			description,
			forms.String,
			false,
			defaultValue,
			envVars,
			[]string{},
		)
		Expect(err).NotTo(HaveOccurred())
	}

	v := valueRef{nil}
	f.values[name] = &v

	err = field.(*forms.InputField).SetValueRef(&v.value)
	Expect(err).NotTo(HaveOccurred())
}

func (f *FakeConfig) GetInternalValue(name string) (*string, bool) {

	var (
		exists bool
	)

	v, exists := f.values[name]
	return v.value, exists
}

func (f *FakeConfig) InputForm() (forms.InputForm, error) {
	return f.inputGroup, nil
}

func (f *FakeConfig) GetValue(name string) (*string, error) {
	return f.inputGroup.GetFieldValue(name)
}

func (f *FakeConfig) Copy() (config.Configurable, error) {
	return nil, nil
}

func (f *FakeConfig) IsValid() bool {
	return true
}

func (f *FakeConfig) Reset() {
	f.InitConfig(f.inputGroup.Name(), f.inputGroup.Description())
}
