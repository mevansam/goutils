package entry

import (
	"fmt"
)

// Input types
type InputType int

const (
	String InputType = iota
	Number
	FilePath
	HttpUrl
	EmailAddress
	JsonInput
	Container
)

// Input abstraction
type Input interface {
	Name() string
	DisplayName() string
	Description() string
	LongDescription() string

	Type() InputType
	Inputs() []Input

	getGroupId() int
}

// InputForm abstraction
type InputForm interface {
	Input

	GetInputField(name string) (*InputField, error)

	GetFieldValue(name string) (*string, error)
	SetFieldValue(name string, value string) error

	InputFields() []*InputField
	InputValues() map[string]string
}

// This structure is a container for a collection of
// inputs which implements the InputForm abstraction
type InputGroup struct {
	name        string
	description string

	displayName string

	groupId int
	inputs  []Input

	containers   map[int]*InputGroup
	fieldNameSet map[string]Input
}

// in: name        - name of the container
// in: displayName - the name to display when requesting input
// in: description - a long description which can also be
//                     the help text for the container
// out: An initialized instance of an InputGroup of type "Container" structure
func (g *InputGroup) NewInputContainer(
	name, displayName, description string,
	groupId int,
) Input {

	container := &InputGroup{
		name:        name,
		description: description,
		groupId:     groupId,

		displayName: displayName,

		containers:   g.containers,
		fieldNameSet: g.fieldNameSet,
	}
	g.containers[groupId] = container

	return container
}

// in: name          - name of the field
// in: displayName   - the name to display when requesting
//                     input
// in: description   - a long description which can also be
//                     the help text for the field
// in: inputType     - type of the input used for validation
// in: valueFromFile - if true then the input value should be
//                     a file which will be read as the value
//                     of the field
// in: envVars       - any environment variables the value for
//                     this input can be sourced from
// in: dependsOn     - inputs that this input depends. this
//                     helps define the input flow
//
// out: An initialized instance of an InputField structure
func (g *InputGroup) NewInputField(
	name, displayName, description string,
	inputType InputType,
	valueFromFile bool,
	envVars []string,
	dependsOn []string,
) (Input, error) {

	return g.newInputField(
		name,
		displayName,
		description,
		0,
		inputType,
		valueFromFile,
		nil,
		envVars,
		dependsOn,
	)
}

// in: name          - name of the field
// in: displayName   - the name to display when requesting
//                     input
// in: description   - a long description which can also be
//                     the help text for the field
// in: inputType     - type of the input used for validation
// in: valueFromFile - if true then the input value should be
//                     a file which will be read as the value
//                     of the field
// in: defaultValue  - a default value
// in: dependsOn     - inputs that this input depends. this
//                     helps define the input flow
// in: envVars       - any environment variables the value for
//                     this input can be sourced from
//
// out: An initialized instance of an InputField structure
func (g *InputGroup) NewInputFieldWithDefaultValue(
	name, displayName, description string,
	inputType InputType,
	valueFromFile bool,
	defaultValue string,
	envVars []string,
	dependsOn []string,
) (Input, error) {

	return g.newInputField(
		name,
		displayName,
		description,
		0,
		inputType,
		valueFromFile,
		&defaultValue,
		envVars,
		dependsOn,
	)
}

// in: name          - name of the field
// in: displayName   - the name to display when requesting
//                     input
// in: description   - a long description which can also be
//                     the help text for the field
// in: groupId       - defines a group id. all fields having
//                     the same group id will be added to
//                     a "Container" input group where only
//                     one input in the "Container" will
//                     be collected. an id of 0 flags the
//                     field as not belonging to Container
//                     group
// in: inputType     - type of the input used for validation
// in: valueFromFile - if true then the input value should be
//                     a file which will be read as the value
//                     of the field
// in: envVars       - any environment variables the value for
//                     this input can be sourced from
// in: dependsOn     - inputs that this input depends. this
//                     helps define the input flow
//
// out: An initialized instance of an InputField structure
func (g *InputGroup) NewInputGroupField(
	name, displayName, description string,
	groupId int,
	inputType InputType,
	valueFromFile bool,
	envVars []string,
	dependsOn []string,
) (Input, error) {

	return g.newInputField(
		name,
		displayName,
		description,
		groupId,
		inputType,
		valueFromFile,
		nil,
		envVars,
		dependsOn,
	)
}

// in: name          - name of the field
// in: displayName   - the name to display when requesting
//                     input
// in: description   - a long description which can also be
//                     the help text for the field
// in: groupId       - defines a group id. all fields having
//                     the same group id will be added to
//                     a "Container" input group where only
//                     one input in the "Container" will
//                     be collected. an id of 0 flags the
//                     field as not belonging to Container
//                     group
// in: inputType     - type of the input used for validation
// in: valueFromFile - if true then the input value should be
//                     a file which will be read as the value
//                     of the field
// in: defaultValue  - a default value
// in: dependsOn     - inputs that this input depends. this
//                     helps define the input flow
// in: envVars       - any environment variables the value for
//                     this input can be sourced from
//
// out: An initialized instance of an InputField structure
func (g *InputGroup) NewInputGroupFieldWithDefaultValue(
	name, displayName, description string,
	groupId int,
	inputType InputType,
	valueFromFile bool,
	defaultValue string,
	envVars []string,
	dependsOn []string,
) (Input, error) {

	return g.newInputField(
		name,
		displayName,
		description,
		groupId,
		inputType,
		valueFromFile,
		&defaultValue,
		envVars,
		dependsOn,
	)
}

// in: name          - name of the field
// in: displayName   - the name to display when requesting
//                     input
// in: description   - a long description which can also be
//                     the help text for the field
// in: groupId       - defines a group id. all fields having
//                     the same group id will be added to
//                     a "Container" input group where only
//                     one input in the "Container" will
//                     be collected. an id of 0 flags the
//                     field as not belonging to Container
//                     group
// in: inputType     - type of the input used for validation
// in: valueFromFile - if true then the input value should be
//                     a file which will be read as the value
//                     of the field
// in: defaultValue  - a default value. nil if no default value
// in: envVars       - any environment variables the value for
//                     this input can be sourced from
// in: dependsOn     - inputs that this input depends. this
//                     helps define the input flow
//
// out: An initialized instance of an InputField structure
func (g *InputGroup) newInputField(
	name, displayName, description string,
	groupId int,
	inputType InputType,
	valueFromFile bool,
	defaultValue *string,
	envVars []string,
	dependsOn []string,
) (Input, error) {

	var (
		err    error
		exists bool

		field *InputField
	)

	// Do not allow adding duplicate fields
	if _, exists = g.fieldNameSet[name]; exists {
		return nil, fmt.Errorf(
			"a field with name '%s' has already been added",
			name)
	}

	field = &InputField{
		InputGroup: InputGroup{
			name:        name,
			description: description,
			groupId:     groupId,

			displayName: displayName,

			containers:   g.containers,
			fieldNameSet: g.fieldNameSet,
		},
		inputType: inputType,

		valueFromFile: valueFromFile,
		envVars:       envVars,
		defaultValue:  defaultValue,

		inputSet: false,
		valueRef: nil,

		acceptedValues:  nil,
		inclusionFilter: nil,
		exclusionFilter: nil,
	}
	g.fieldNameSet[name] = field

	if len(dependsOn) > 0 {

		// recursively add field to all
		// inputs that it depends on

		var (
			addToDepends func(
				input Input,
				names map[string]bool,
			) (bool, error)

			names map[string]bool
			added bool
		)

		addToDepends = func(
			input Input,
			names map[string]bool,
		) (bool, error) {
			for _, i := range input.Inputs() {

				if _, exists := names[i.Name()]; exists && i.Type() != Container {
					f := i.(*InputField)
					if err = f.addInputField(field); err != nil {
						return false, err
					}

					delete(names, i.Name())
					if len(names) == 0 {
						return true, nil
					}

				} else if len(i.Inputs()) > 0 {
					if added, err = addToDepends(i, names); added || err != nil {
						return added, err
					}
				}
			}
			return false, nil
		}

		names = make(map[string]bool)
		for _, n := range dependsOn {
			names[n] = true
		}
		if added, err = addToDepends(g, names); !added && err == nil {
			err = fmt.Errorf(
				"unable to add field '%s' as one or more dependent fields %v not found",
				field.name, dependsOn)
		}

	} else {
		err = g.addInputField(field)
	}
	return field, err
}

func (g *InputGroup) addInputField(
	field *InputField,
) error {

	var (
		ig     *InputGroup
		exists bool
	)

	add := true
	for _, f := range g.inputs {

		if field.groupId > 0 && field.groupId == f.getGroupId() {

			if f.Type() != Container {
				return fmt.Errorf("invalid internal input container state")
			}

			// append to existing group
			ig = f.(*InputGroup)
			ig.inputs = append(ig.inputs, field)

			add = false
			break
		}
	}
	if add {

		if field.groupId > 0 {
			// retrieve group container
			// to add new field to
			if ig, exists = g.containers[field.groupId]; !exists {
				return fmt.Errorf(
					"unable to add field '%s' as its group '%d' was not found",
					field.name, field.groupId)
			}
			ig.inputs = append(ig.inputs, field)
			g.inputs = append(g.inputs, ig)
		} else {
			g.inputs = append(g.inputs, field)
		}
	}
	return nil
}

// interface: Input

// out: the name of the group
func (g *InputGroup) Name() string {
	return g.name
}

// out: the display name of the group
func (g *InputGroup) DisplayName() string {
	return g.displayName
}

// out: the description of the group
func (g *InputGroup) Description() string {
	return g.description
}

// out: the long description of the group
func (g *InputGroup) LongDescription() string {
	return g.description
}

// out: returns input type of "Container"
func (g *InputGroup) Type() InputType {
	return Container
}

// out: a list of all inputs for the group
func (g *InputGroup) Inputs() []Input {
	return g.inputs
}

// out: return the group id
func (g *InputGroup) getGroupId() int {
	return g.groupId
}

// interface: InputForm

// in: the name of the input field to retrieve
// out: the input field with the given name
func (g *InputGroup) GetInputField(name string) (*InputField, error) {

	var (
		input Input
		field *InputField
		ok    bool
	)

	if input, ok = g.fieldNameSet[name]; !ok {
		return nil, fmt.Errorf("field '%s' was not found in form", name)
	}
	if field, ok = input.(*InputField); !ok {
		return nil, fmt.Errorf("internal state error retrieving field '%s'", name)
	}
	return field, nil
}

// in: the name of the input field whose value should be retrieved
// out: a reference to the value of the input field
func (g *InputGroup) GetFieldValue(name string) (*string, error) {

	var (
		err   error
		field *InputField
	)

	if field, err = g.GetInputField(name); err != nil {
		return nil, err
	}
	return field.Value(), nil
}

// in: the name of the input field to set the value of
// in: a reference to the value to set. if nil the value is cleared
func (g *InputGroup) SetFieldValue(name string, value string) error {

	var (
		err   error
		field *InputField
	)

	if field, err = g.GetInputField(name); err != nil {
		return err
	}
	return field.SetValue(&value)
}

// out: a list of all fields for the group
func (g *InputGroup) InputFields() []*InputField {
	return g.inputFields(make(map[string]bool))
}

// in: set of added fields
// out: a list of all fields for the group
func (g *InputGroup) inputFields(added map[string]bool) []*InputField {

	fields := []*InputField{}
	for _, f := range g.inputs {
		if f.Type() == Container {
			// recursively retrieve fields from groups
			ig := f.(*InputGroup)
			fields = append(fields, ig.inputFields(added)...)

		} else if _, exists := added[f.Name()]; !exists {
			fields = append(fields, f.(*InputField))
			added[f.Name()] = true

			ig := f.(*InputField)
			// recursively retrieve fields from groups
			fields = append(fields, ig.inputFields(added)...)
		}
	}
	return fields
}

// out: map of name-values of all inputs entered
func (g *InputGroup) InputValues() map[string]string {

	var (
		val *string
	)

	valueMap := make(map[string]string)
	inputFields := g.InputFields()

	for _, f := range inputFields {
		if f.InputSet() {
			val = f.Value()
			valueMap[f.Name()] = *val
		}
	}
	return valueMap
}
