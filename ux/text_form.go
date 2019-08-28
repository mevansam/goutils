package ux

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/peterh/liner"

	"github.com/mevansam/goutils/data/entry"
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
		err error

		doubleDivider, singleDivider string

		nameLen, l, j int

		cursor     *entry.InputCursor
		inputField *entry.InputField
		input      entry.Input

		prompt,
		response,
		suggestion string
		value *string
	)

	line := liner.NewLiner()
	line.SetCtrlCAborts(true)

	defer func() {
		line.Close()
	}()

	doubleDivider = strings.Repeat("=", width)
	singleDivider = strings.Repeat("-", width)

	tf.printFormHeader("", width)
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
			if response, err = line.Prompt("Please select one of the above ? "); err != nil {
				return err
			}
			if j, err = strconv.Atoi(response); err != nil {
				return err
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
		if value, err = inputField.Value(); err != nil {
			return err
		}

		line.SetCompleter(func(line string) []string {
			if acceptedValues := inputField.AcceptedValues(); acceptedValues == nil {
				if value == nil {
					return []string{}
				} else {
					// if a input has a value then return it as a completion
					// value so the user can retrieve it by tabbing
					return []string{*value, ""}
				}

			} else {
				// if input has a list of accepted values then allow completion on those values
				return *acceptedValues
			}
		})

		// if input already has a value then prompt with
		// the current value as a suggestion as default
		if value == nil {
			suggestion = ""
		} else {
			suggestion = *value
		}
		if response, err = line.PromptWithSuggestion(prompt, suggestion, -1); err != nil {
			return err
		}

		// set input with entered value
		if cursor, err = cursor.SetInput(input.Name(), response); err != nil {
			return err
		}

		fmt.Println()
		cursor = cursor.NextInput()
	}

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
	fmt.Print("\n")

	fmt.Print(padding)
	utils.RepeatString("=", len(tf.title), os.Stdout)
	fmt.Print("\n\n")

	fmt.Print(padding)
	fmt.Print(tf.inputGroup.Description())
	fmt.Print("\n\n")

	l := len(padding)
	s, _ := utils.SplitString(tf.heading, l, width-l, true)
	fmt.Print(s)

	fmt.Print("\n")
}

func (tf *TextForm) getInputLongDescription(
	input entry.Input,
	showDefaults bool,
	nameLen int,
	padding, bullet string,
	indent, width int,
) string {
	var (
		err error
		ok  bool

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
			if value, err = field.Value(); value != nil && err == nil {
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
