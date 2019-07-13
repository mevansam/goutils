package ux

import (
	"fmt"
	"io"
	"strings"

	"github.com/mevansam/goutils/data/entry"
	"github.com/mevansam/goutils/utils"
)

type TextFormatter struct {
	ShowDefaults bool
}

func NewTextFormatter() *TextFormatter {

	return &TextFormatter{
		ShowDefaults: false,
	}
}

func (tf *TextFormatter) ShowDataInputForm(
	title string,
	heading string,
	startIndent, indentSpaces, width int,
	input entry.Input,
	outputBuffer io.Writer,
) {

	var (
		padding             string
		printInputReference func(level int, input entry.Input)

		fieldLengths map[string]*int
	)

	fieldLengths = make(map[string]*int)
	calcNameLengths(input, fieldLengths, nil, true)
	padding = strings.Repeat(" ", startIndent)

	printInputReference = func(level int, input entry.Input) {

		var (
			err error
			ok  bool

			out strings.Builder

			levelIndent string
			name        string

			inputs        []entry.Input
			ii            entry.Input
			i, l, nameLen int

			field        *entry.InputField
			value        *string
			defaultValue string
		)

		if input.Type() == entry.Container {

			// output description of a group of
			// inputs which are mutually exclusive

			out.WriteString("\n")
			out.WriteString(padding)
			utils.RepeatString(" ", level*indentSpaces, &out)
			out.WriteString("* Provide one of the following for:\n\n")

			levelIndent = strings.Repeat(" ", (level+1)*indentSpaces)

			out.WriteString(padding)
			out.WriteString(levelIndent)
			out.WriteString(input.Description())

		} else {

			out.WriteString("\n")
			out.WriteString(padding)
			utils.RepeatString(" ", level*indentSpaces, &out)
			out.WriteString("* ")

			name = input.DisplayName()
			nameLen = *fieldLengths[name]
			out.WriteString(name)

			utils.RepeatString(" ", nameLen-len(name), &out)
			out.WriteString(" - ")

			l = len(out.String()) - 1
			description, _ := utils.SplitString(input.LongDescription(), l, width-l, false)
			out.WriteString(description)

			if tf.ShowDefaults {
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
		}
		fmt.Fprint(outputBuffer, out.String())

		inputs = input.Inputs()
		for i, ii = range inputs {

			if input.Type() == entry.Container {
				if i > 0 {
					fmt.Fprint(outputBuffer, "\n\n")
					fmt.Fprint(outputBuffer, padding)
					fmt.Fprint(outputBuffer, levelIndent)
					fmt.Fprint(outputBuffer, "OR\n")
				} else {
					fmt.Fprint(outputBuffer, "\n")
				}
				printInputReference(level+1, ii)

			} else {
				if ii.Type() == entry.Container {
					fmt.Fprint(outputBuffer, "\n")
				}
				printInputReference(level, ii)
			}
		}
		if input.Type() == entry.Container {

			inputs = ii.Inputs()
			l = len(inputs)

			// end group with new line. handle case where if last input
			// also had a container at the end of its inputs two newlines
			// will be outputwhen only on newline should have been output
			if l == 0 || inputs[l-1].Type() != entry.Container {
				fmt.Fprint(outputBuffer, "\n")
			}
		}
	}

	fmt.Fprint(outputBuffer, padding)
	fmt.Fprint(outputBuffer, title)
	fmt.Fprint(outputBuffer, "\n")

	fmt.Fprint(outputBuffer, padding)
	utils.RepeatString("=", len(title), outputBuffer)
	fmt.Fprint(outputBuffer, "\n\n")

	fmt.Fprint(outputBuffer, padding)
	fmt.Fprint(outputBuffer, input.Description())
	fmt.Fprint(outputBuffer, "\n\n")

	l := len(padding)
	s, _ := utils.SplitString(heading, l, width-l, true)
	fmt.Fprint(outputBuffer, s)

	fmt.Fprint(outputBuffer, "\n")

	for _, i := range input.Inputs() {
		printInputReference(0, i)
	}
}

func calcNameLengths(
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
				calcNameLengths(i, fieldLengths, length, false)
			}

			l := len(name)
			if l > *length {
				*length = l
			}
		} else {

			// reset length when a container is encountered
			ll := 0
			length = &ll

			calcNameLengths(i, fieldLengths, nil, false)
		}
	}
}
