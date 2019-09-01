package ux

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/peterh/liner"

	"github.com/mevansam/goutils/data/entry"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
)

type TextForm struct {
	title,
	heading string

	inputGroup *entry.InputGroup
}

func NewTextForm(
	title, heading string,
	input entry.Input,
) (*TextForm, error) {

	var (
		ok         bool
		inputGroup *entry.InputGroup
	)

	if inputGroup, ok = input.(*entry.InputGroup); !ok {
		return nil, fmt.Errorf("input is not of type entry.InputGroup: %#v", input)
	}

	return &TextForm{
		title:   title,
		heading: heading,

		inputGroup: inputGroup,
	}, nil
}

func (tf *TextForm) GetInput(
	showDefaults bool,
	indentSpaces, width int,
) error {

	var (
		err    error
		exists bool

		nameLen, l, j int

		cursor     *entry.InputCursor
		inputField *entry.InputField
		input      entry.Input

		doubleDivider, singleDivider,
		prompt, response, suggestion,
		envVal string

		value *string

		valueFromFile bool
		filePaths,
		hintValues,
		fieldHintValues []string
	)

	line := liner.NewLiner()
	line.SetCtrlCAborts(true)

	defer func() {
		line.Close()
	}()

	doubleDivider = strings.Repeat("=", width)
	singleDivider = strings.Repeat("-", width)

	tf.printFormHeader("", width)
	fmt.Println(doubleDivider)
	fmt.Println()

	cursor = entry.NewInputCursor(tf.inputGroup)
	cursor = cursor.NextInput()

	for cursor != nil {
		if input, err = cursor.GetCurrentInput(); err != nil {
			return err
		}

		if input.Type() == entry.Container {

			fmt.Println(input.Description())
			fmt.Println(doubleDivider)

			// normalize display name length
			// of all input group fields
			nameLen = 0
			for _, ii := range input.Inputs() {
				l = len(ii.DisplayName())
				if nameLen < l {
					nameLen = l
				}
			}

			// show list of possible inputs an prompt
			// which input should be requested
			inputs := input.Inputs()
			options := make([]string, len(inputs))

			for i, ii := range inputs {

				options[i] = strconv.Itoa(i + 1)
				fmt.Println(tf.getInputLongDescription(
					ii, showDefaults, l, "",
					fmt.Sprintf("%s. ", options[i]), 0, width,
				))
				fmt.Println(singleDivider)
			}

			line.SetCompleter(func(line string) (c []string) {
				// allow selection of options using tab
				return options
			})
			for {
				if response, err = line.Prompt("Please select one of the above ? "); err != nil {
					return err
				}
				if j, err = strconv.Atoi(response); err == nil {
					break
				}
			}

			fmt.Println(singleDivider)
			input = inputs[j-1]
			prompt = input.DisplayName() + " : "

		} else {

			fmt.Println(tf.getInputLongDescription(
				input, showDefaults, len(input.DisplayName()),
				"", "", 0, width,
			))
			fmt.Println(singleDivider)
			prompt = ": "
		}

		inputField = input.(*entry.InputField)
		value = inputField.Value()

		valueFromFile, filePaths = inputField.ValueFromFile()
		if valueFromFile {

			// if value for the field is sourced from a file then
			// create a list of auto-completion hints with default
			// values from the environment
			if value != nil {
				hintValues = append(filePaths, []string{"", "[saved]"}...)
			} else {
				hintValues = append(filePaths, "")
			}
			suggestion = hintValues[len(hintValues)-1]

		} else {

			if values := inputField.AcceptedValues(); values != nil {
				// if values are restrcted to a given list then
				// create a list of auto-completion hints only
				// with those values
				hintValues = append(*values)
				if value != nil {
					suggestion = *value
				} else {
					suggestion = ""
				}

			} else {
				// create a list of auto-completion hints from
				// the environment variable associated with the
				// input field along with any values retrieved
				// from any field hints set in the input group.
				hintValues = []string{}

				// set of added values used to ensure
				// the same values are not added twice
				valueSet := map[string]bool{"": true}
				if value != nil && len(*value) > 0 {
					valueSet[*value] = true
				}

				// add values sourced from environment to completion list
				for _, e := range inputField.EnvVars() {
					if envVal, exists = os.LookupEnv(e); exists {
						if _, exists = valueSet[envVal]; !exists {
							hintValues = append(hintValues, envVal)
							valueSet[envVal] = true
						}
					}
				}

				// add values sourced from hints to completion list
				if fieldHintValues, err = tf.inputGroup.GetFieldValueHints(input.Name()); err != nil {
					logger.DebugMessage(
						"Error retrieving hint values for field '%s': '%s'",
						input.Name(), err.Error())
				}
				hintValues = append(append(hintValues, fieldHintValues...), "")
				if value != nil {
					hintValues = append(hintValues, *value)
				}
				suggestion = hintValues[len(hintValues)-1]
			}
		}

		line.SetCompleter(func(line string) []string {
			return hintValues
		})
		if response, err = line.PromptWithSuggestion(prompt, suggestion, -1); err != nil {
			return err
		}

		// set input with entered value
		if valueFromFile && response == "[saved]" {
			if cursor, err = cursor.SetDefaultInput(input.Name()); err != nil {
				return err
			}
		} else {
			if cursor, err = cursor.SetInput(input.Name(), response); err != nil {
				return err
			}
		}
		if valueFromFile {
			fmt.Printf("Value from file: \n%s\n", *inputField.Value())
		}

		fmt.Println()
		cursor = cursor.NextInput()
	}

	fmt.Println(doubleDivider)
	return nil
}

func (tf *TextForm) ShowInputReference(
	showDefaults bool,
	startIndent, indentSpaces, width int,
) {

	var (
		padding    string
		printInput func(level int, input entry.Input)

		fieldLengths map[string]*int
	)

	padding = strings.Repeat(" ", startIndent)

	fieldLengths = make(map[string]*int)
	tf.calcNameLengths(tf.inputGroup, fieldLengths, nil, true)

	printInput = func(level int, input entry.Input) {

		var (
			levelIndent string

			inputs []entry.Input
			ii     entry.Input
			i, l   int
		)

		fmt.Println()
		if input.Type() == entry.Container {

			// output description of a group of
			// inputs which are mutually exclusive

			fmt.Printf(padding)
			utils.RepeatString(" ", level*indentSpaces, os.Stdout)
			fmt.Printf("* Provide one of the following for:\n\n")

			levelIndent = strings.Repeat(" ", (level+1)*indentSpaces)

			fmt.Printf(padding)
			fmt.Printf(levelIndent)
			fmt.Printf(input.Description())

		} else {

			fmt.Printf(tf.getInputLongDescription(
				input,
				showDefaults,
				*fieldLengths[input.DisplayName()],
				padding, "* ",
				level*indentSpaces, width,
			))
		}

		inputs = input.Inputs()
		for i, ii = range inputs {

			if input.Type() == entry.Container {
				if i > 0 {
					fmt.Print("\n\n")
					fmt.Print(padding)
					fmt.Print(levelIndent)
					fmt.Print("OR\n")
				} else {
					fmt.Print("\n")
				}
				printInput(level+1, ii)

			} else {
				if ii.Type() == entry.Container {
					fmt.Print("\n")
				}
				printInput(level, ii)
			}
		}
		if input.Type() == entry.Container {

			inputs = ii.Inputs()
			l = len(inputs)

			// end group with new line. handle case where if last input
			// also had a container at the end of its inputs two newlines
			// will be outputwhen only on newline should have been output
			if l == 0 || inputs[l-1].Type() != entry.Container {
				fmt.Print("\n")
			}
		}
	}

	tf.printFormHeader(padding, width)
	for _, i := range tf.inputGroup.Inputs() {
		printInput(0, i)
	}
}

func (tf *TextForm) printFormHeader(
	padding string,
	width int,
) {
	fmt.Print(padding)
	fmt.Print(tf.title)
	fmt.Println()

	fmt.Print(padding)
	utils.RepeatString("=", len(tf.title), os.Stdout)
	fmt.Print("\n\n")

	fmt.Print(padding)
	fmt.Print(tf.inputGroup.Description())
	fmt.Print("\n\n")

	l := len(padding)
	s, _ := utils.SplitString(tf.heading, l, width-l, true)
	fmt.Print(s)
	fmt.Println()
}

func (tf *TextForm) getInputLongDescription(
	input entry.Input,
	showDefaults bool,
	nameLen int,
	padding, bullet string,
	indent, width int,
) string {

	var (
		ok bool

		out strings.Builder

		name string
		l    int

		field        *entry.InputField
		value        *string
		defaultValue string
	)

	out.WriteString(padding)
	utils.RepeatString(" ", indent, &out)
	out.WriteString(bullet)

	name = input.DisplayName()
	out.WriteString(name)

	utils.RepeatString(" ", nameLen-len(name), &out)
	out.WriteString(" - ")

	l = len(out.String())
	description, _ := utils.SplitString(input.LongDescription(), l, width-l, false)
	out.WriteString(description)

	if showDefaults {
		if field, ok = input.(*entry.InputField); ok {
			if value = field.Value(); value != nil {
				out.WriteString("\n")
				out.WriteString(padding)

				if field.Sensitive() {
					defaultValue, _ = utils.SplitString("(Default value = '****')", l, width-l, true)
					out.WriteString(defaultValue)
				} else {
					defaultValue, _ = utils.SplitString(fmt.Sprintf("(Default value = '%s')", *value), l, width-l, true)
					out.WriteString(defaultValue)

				}
			}
		}
	}

	return out.String()
}

func (tf *TextForm) calcNameLengths(
	input entry.Input,
	fieldLengths map[string]*int,
	length *int,
	isRoot bool,
) {

	if length == nil {
		ll := 0
		length = &ll
	}

	for _, i := range input.Inputs() {

		if i.Type() != entry.Container {

			if !isRoot && input.Type() == entry.Container {
				// reset length for each input in a container
				ll := 0
				length = &ll
			}

			name := i.DisplayName()
			fieldLengths[name] = length

			ii := i.Inputs()
			if len(ii) > 0 {
				tf.calcNameLengths(i, fieldLengths, length, false)
			}

			l := len(name)
			if l > *length {
				*length = l
			}
		} else {

			// reset length when a container is encountered
			ll := 0
			length = &ll

			tf.calcNameLengths(i, fieldLengths, nil, false)
		}
	}
}
