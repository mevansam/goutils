package forms

import "fmt"

type InputCursor struct {
	parents []*InputCursor
	group   Input
	index   int
}

func NewInputCursorFromCollection(
	groupName string,
	collection *InputCollection,
) (*InputCursor, error) {

	input := collection.Group(groupName)
	if input == nil {
		return nil, fmt.Errorf(
			"group '%s' not found in collection",
			groupName)
	}

	return NewInputCursor(input), nil
}

func NewInputCursor(
	input *InputGroup,
) *InputCursor {

	return &InputCursor{
		parents: []*InputCursor{},
		group:   input,
		index:   -1,
	}
}

// advances the cursor to the next input
func (c *InputCursor) NextInput() *InputCursor {

	cursor := c
	for {

		cursor.index++
		if cursor.index == len(cursor.group.Inputs()) {

			if len(cursor.parents) == 0 {
				// we have reached the end of all possible inputs
				cursor = nil
				break
			}
			// resume cursor at parent
			cursor = cursor.parents[0]

		} else {
			break
		}
	}
	return cursor
}

// out: input at current cursor position
func (c *InputCursor) GetCurrentInput() (Input, error) {
	if c.index == -1 {
		return nil, fmt.Errorf("cursor is needs to be advanced before retrieving input")
	} else {
		return c.group.Inputs()[c.index], nil
	}
}

// sets value of input at current cursor position
// and updates state if input has dependent inputs
//
// in: name - of input to set value of
// in: value - value to set. if nil default value will be used.
// out: c
func (c *InputCursor) SetInput(name, value string) (*InputCursor, error) {
	return c.setInput(name, &value)
}

// sets default value of input at current cursor position
// and updates state if input has dependent inputs
//
// in: name - of input to set value of
// out: c
func (c *InputCursor) SetDefaultInput(name string) (*InputCursor, error) {
	return c.setInput(name, nil)
}

// sets value of input at current cursor position
// and updates state if input has dependent inputs
//
// in: name - of input to set value of
// in: value - value to set. if nil default value will be used.
// out: c
func (c *InputCursor) setInput(name string, value *string) (*InputCursor, error) {

	var (
		err error

		cursor *InputCursor
		inputs []Input

		currInput,
		selectedInput Input

		inputField *InputField
	)

	cursor = c
	inputs = cursor.group.Inputs()
	currInput = inputs[cursor.index]

	if currInput.Type() == Container {

		selectedInput = nil
		for _, i := range currInput.Inputs() {

			if name == i.Name() {
				selectedInput = i
				break
			}
		}
		if selectedInput == nil {
			return cursor, fmt.Errorf(
				"unable to find input '%s' within 'Container' of mutually exclusive inputs '%s",
				name, cursor.group.Inputs()[cursor.index].Name())
		}
		currInput = selectedInput

	} else if name != currInput.Name() {

		return cursor, fmt.Errorf(
			"cursor is at input '%s' which is different from provided input name '%s' to set value of",
			currInput.Name(), name)
	}

	inputField = currInput.(*InputField)
	inputField.SetInput()
	if value != nil {
		if err = inputField.SetValue(value); err != nil {
			return cursor, err
		}

	} else if !inputField.HasValue() {
		return cursor, fmt.Errorf(
			"no default value for input name '%s' could be determined",
			name)
	}

	if len(currInput.Inputs()) > 0 {

		// input for which value was set has dependents.
		// so update cursor to point to the dependents.
		cursor = &InputCursor{
			parents: append([]*InputCursor{c}, c.parents...),
			group:   currInput,
			index:   -1,
		}
	}
	return cursor, nil
}
