package utils

import (
	"fmt"
	"io"
	"strings"
)

func JoinListAsSentence(format string, list []string, quoteListItems bool) string {

	var (
		listAsString strings.Builder
	)

	joinItem := func(item string) {

		if quoteListItems {
			listAsString.WriteByte('"')
		}
		listAsString.WriteString(item)
		if quoteListItems {
			listAsString.WriteByte('"')
		}
	}

	l := len(list)
	if l > 0 {
		l--

		for i, v := range list {

			if i == 0 {
				joinItem(v)
			} else {
				if i == l {
					listAsString.WriteString(" and ")
				} else {
					listAsString.WriteString(", ")
				}
				joinItem(v)
			}
		}
	}

	return fmt.Sprintf(format, listAsString.String())
}

func SplitString(input string, indent, width int, indentFirst bool) (string, bool) {

	var (
		lines []string

		ch          byte
		currentLine string

		lastLine, lineLength, splitAt, nextAt int

		out strings.Builder
	)

	lastLine = 0
	lines = []string{input}

	isWhitespace := func(ch byte) bool {
		return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
	}

	for len(lines[lastLine]) > width {

		splitAt = width
		currentLine = lines[lastLine]
		lineLength = len(currentLine)

		// Find first white space char before or at `width`
		ch = currentLine[splitAt]
		for splitAt > 0 && !isWhitespace(ch) {
			splitAt--
			ch = currentLine[splitAt]
		}
		if splitAt > 0 {

			// Find next non-whitespace char to break out new line
			// This loop will be invoked only if `width` happens to
			// fall within a sequence of whitespaces
			nextAt = splitAt
			ch = ' '
			for isWhitespace(ch) {
				nextAt++
				if nextAt == lineLength {
					break
				}
				ch = currentLine[nextAt]
			}

			// Find non-whitespace character where the current line
			// should break at. This loop will be invoked only if
			// there exists a sequence of whitespaces before `splitAt`.
			ch = currentLine[splitAt]
			for splitAt > 0 && isWhitespace(ch) {
				splitAt--
				ch = currentLine[splitAt]
			}

			if splitAt == 0 {
				// Current line to break is a cotiguous sequence of
				// white spaces
				if nextAt < lineLength {
					lines[lastLine] = currentLine[nextAt:]
				}
			} else {
				// Split current line
				lines[lastLine] = currentLine[:splitAt+1]

				// Break out next line
				if nextAt < lineLength {
					lines = append(lines, currentLine[nextAt:])
					lastLine++
				}
			}

		} else {
			// Break line exactly at width as line is a contiguos
			// sequence of non-whitespace characters.
			lines = append(lines, currentLine[width:])
			lines[lastLine] = currentLine[:width]
			lastLine++
		}
	}

	lastLine = len(lines) - 1
	for i, l := range lines {
		if indent > 0 && (indentFirst || i > 0) {
			out.WriteString(strings.Repeat(" ", indent))
		}
		out.WriteString(l)
		if i != lastLine {
			out.WriteString("\n")
		}
	}
	return out.String(), len(lines) > 1
}

func RepeatString(s string, n int, out io.Writer) {

	outSequence := []byte(s)
	for i := 0; i < n; i++ {
		if _, err := out.Write(outSequence); err != nil {
			panic(err)
		}
	}
}
