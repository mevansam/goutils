package entry

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/mevansam/goutils/logger"
)

// This structure defines metadata for an input field
// in a data input group. If a field has a required
// element then input for the required field will be
// collected first in a wizard like flow. It implements
// the Input abstraction.
type InputField struct {
	InputGroup

	inputType InputType

	valueFromFile bool
	envVars       []string
	defaultValue  *string

	hasValue bool
	inputSet bool

	valueRef interface{}

	sensitive bool

	acceptedValues             *[]string
	acceptedValueSet           map[string]bool
	acceptedValuesErrorMessage string

	inclusionFilter,
	exclusionFilter *regexp.Regexp

	inclusionFilterErrorMessage,
	exclusionFilterErrorMessage string
}

// in: inclusionFilter - field value must match this regex
// in: inclusionFilterErrorMessage - the message to display if inclusion filter does not match
func (f *InputField) SetInclusionFilter(
	inclusionFilter, inclusionFilterErrorMessage string,
) error {

	var (
		err error
	)

	if f.inclusionFilter, err = regexp.Compile(inclusionFilter); err != nil {
		return err
	}
	f.inclusionFilterErrorMessage = inclusionFilterErrorMessage
	return nil
}

// in: exclusionFilter - field value should not match this regex
// in: exclusionFilterErrorMessage - the message to display if exclusion filter matches
func (f *InputField) SetExclusionFilter(
	exclusionFilter, exclusionFilterErrorMessage string,
) error {

	var (
		err error
	)

	if f.exclusionFilter, err = regexp.Compile(exclusionFilter); err != nil {
		return err
	}
	f.exclusionFilterErrorMessage = exclusionFilterErrorMessage
	return nil
}

// in: acceptedValues - list of acceptable values for field
func (f *InputField) SetAcceptedValues(
	acceptedValues *[]string,
	acceptedValuesErrorMessage string,
) {
	f.acceptedValues = acceptedValues
	f.acceptedValuesErrorMessage = acceptedValuesErrorMessage

	if f.acceptedValues != nil {
		f.acceptedValueSet = make(map[string]bool)
		for _, v := range *acceptedValues {
			f.acceptedValueSet[v] = true
		}
	} else {
		f.acceptedValueSet = nil
	}
}

// out: list of acceptable values for field
func (f *InputField) AcceptedValues() *[]string {
	return f.acceptedValues
}

// in: sensitive - indicates if the field value should be masked
func (f *InputField) SetSensitive(sensitive bool) {
	f.sensitive = sensitive
}

// out: whether to mask the field value
func (f *InputField) Sensitive() bool {
	return f.sensitive
}

// out: whether the field is optional as it has a default value
func (f *InputField) Optional() bool {
	return f.defaultValue != nil
}

// out: environment variables associated with this field
func (f *InputField) EnvVars() []string {
	return f.envVars
}

// out: the long description of the group
func (f *InputField) LongDescription() string {

	var (
		buf strings.Builder
	)

	buf.WriteString(f.description)

	l := len(f.envVars)
	if l > 0 {
		buf.WriteString(" It will be sourced from the environment variable")
		if l > 1 {
			buf.WriteString("s")
		}
		buf.WriteString(" ")

		for i, v := range f.envVars {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(v)
		}
		buf.WriteString(" if not provided.")
	}

	return buf.String()
}

// out: returns input type of the field
func (f *InputField) Type() InputType {
	return f.inputType
}

// out: whether a value can be returned for this input
func (f *InputField) HasValue() bool {

	if f.hasValue {
		return true

	} else {
		if len(f.envVars) > 0 {
			for _, e := range f.envVars {
				if _, exists := os.LookupEnv(e); exists {
					return true
				}
			}
		}
		return false
	}
}

// in: valueRef - pointer to a value or a pointer to a pointer to
//                a value. changing the contents of this pointer
//                will modify the value reference and hence the
//								contents of the field.
func (f *InputField) SetValueRef(valueRef interface{}) error {

	var (
		ptrValue, ptrToValue reflect.Value
	)

	ptrValue = reflect.ValueOf(valueRef) // pointer to the pointer of the value object
	if ptrValue.Kind() == reflect.Ptr {
		ptrToValue = reflect.Indirect(ptrValue) // value object or pointer to the value object

		if ptrToValue.Kind() == reflect.String {
			f.hasValue = (len(ptrToValue.Interface().(string)) > 0)
			if !f.hasValue && f.defaultValue != nil {
				ptrToValue.Set(reflect.ValueOf(*f.defaultValue))
				f.hasValue = true
			}

			logger.TraceMessage(
				"Binding input field '%s': 0x%x",
				f.name, ptrValue.Pointer())

		} else if ptrToValue.Kind() == reflect.Ptr {

			if reflect.Indirect(ptrToValue).Kind() == reflect.Invalid {

				if f.defaultValue != nil {
					value := *f.defaultValue
					ptrToValue.Set(reflect.ValueOf(&value))
					f.hasValue = true
				} else {
					f.hasValue = false
				}

			} else if reflect.Indirect(ptrToValue).Kind() == reflect.String {
				f.hasValue = true
			} else {
				return fmt.Errorf(
					"the field '%s' value object being bound must be of type string or nil",
					f.name)
			}

			logger.TraceMessage(
				"Binding input field '%s': 0x%x => 0x%x",
				f.name, ptrValue.Pointer(), ptrToValue.Pointer())

		} else {
			return fmt.Errorf(
				"the field '%s' value reference must be a pointer to string or a pointer to pointer to a string",
				f.name)
		}

	} else {
		return fmt.Errorf(
			"the field '%s' value reference must be a pointer to string or a pointer to pointer to a string",
			f.name)
	}

	f.valueRef = valueRef
	return nil
}

// in: input value
func (f *InputField) SetValue(value *string) error {

	var (
		ptrValue, ptrToValue reflect.Value
	)

	if f.valueRef == nil {
		return fmt.Errorf("field '%s' has not been bound to a value instance", f.name)
	}

	if f.acceptedValueSet != nil {
		if _, ok := f.acceptedValueSet[*value]; !ok {
			return fmt.Errorf(f.acceptedValuesErrorMessage)
		}
	}
	if f.inclusionFilter != nil && !f.inclusionFilter.MatchString(*value) {
		return fmt.Errorf(f.inclusionFilterErrorMessage)
	}
	if f.exclusionFilter != nil && f.exclusionFilter.MatchString(*value) {
		return fmt.Errorf(f.exclusionFilterErrorMessage)
	}

	ptrValue = reflect.ValueOf(f.valueRef)  // pointer to the pointer of the value object
	ptrToValue = reflect.Indirect(ptrValue) // value object or pointer to the value object

	// Update pointer for bound field
	// to the new value pointer
	if ptrToValue.Kind() == reflect.Ptr {
		ptrToValue.Set(reflect.ValueOf(value))

		logger.TraceMessage(
			"Input field '%s' bound to object at 0x%x has been updated to value at 0x%x.",
			f.name, ptrValue.Pointer(), ptrToValue.Pointer())

	} else {
		if value == nil {
			ptrToValue.Set(reflect.ValueOf(""))
		} else {
			ptrToValue.Set(reflect.ValueOf(*value))
		}

		logger.TraceMessage(
			"Input field '%s' bound to object at 0x%x has been updated.",
			f.name, ptrValue.Pointer())
	}

	f.hasValue = (value != nil)
	return nil
}

// flags field as having its input set
func (f *InputField) SetInput() {
	f.inputSet = true
}

// out: whether input has been set
func (f *InputField) InputSet() bool {
	return f.inputSet
}

// out: the value of the input
func (f *InputField) Value() (*string, error) {

	var (
		err error

		ptrValue, ptrToValue reflect.Value
		valueDeref, value    *string

		buf  []byte
		data string
	)
	value = nil

	if f.hasValue {

		ptrValue = reflect.ValueOf(f.valueRef)  // pointer to the pointer of the value object
		ptrToValue = reflect.Indirect(ptrValue) // value object or pointer to the value object
		if ptrToValue.Kind() == reflect.Ptr {
			valueDeref = ptrToValue.Interface().(*string)
		} else {
			valueDeref = ptrValue.Interface().(*string)
		}

		if f.valueFromFile {

			// extract value from file
			if buf, err = ioutil.ReadFile(*valueDeref); err != nil {
				return nil, err
			}
			data = string(buf)
			value = &data

		} else {
			value = valueDeref
		}

	} else {
		if len(f.envVars) > 0 {
			// extract value from environment
			for _, e := range f.envVars {
				if envVal, exists := os.LookupEnv(e); exists {

					logger.TraceMessage(
						"Value of input field '%s' has been sourced from the environment variable '%s'.",
						f.name, e)

					value = &envVal
					break
				}
			}
		}
	}
	return value, nil
}
