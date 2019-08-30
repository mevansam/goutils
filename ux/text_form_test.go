package ux_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/mevansam/goutils/data/entry"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/ux"

	test_data "github.com/mevansam/goutils/test/data"
)

var _ = Describe("Text Formatting tests", func() {

	var (
		err error

		origStdin, stdInWriter,
		origStdout, stdOutReader,
		origStderr *os.File
	)

	BeforeEach(func() {

		// pipe output to be written to by form output
		origStdout = os.Stdout
		stdOutReader, os.Stdout, err = os.Pipe()
		Expect(err).ToNot(HaveOccurred())

		// redirect all output to stderr to new stdout
		origStderr = os.Stderr
		os.Stderr = os.Stdout

		// pipe input to be read in by form input
		origStdin = os.Stdin
		os.Stdin, stdInWriter, err = os.Pipe()
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		stdOutReader.Close()
		os.Stdout = origStdout
		os.Stderr = origStderr
		stdInWriter.Close()
		os.Stdin = origStdin
	})

	Context("Output", func() {

		It("outputs a detailed input data form reference", func() {

			// channel to signal when getting form input is done
			out := make(chan string)

			go func() {

				var (
					output bytes.Buffer
				)

				ic := test_data.NewTestInputCollection()
				tf, err := ux.NewTextForm(
					"Input Data Form for 'input-form'",
					"CONFIGURATION DATA INPUT",
					ic.Group("input-form"),
				)
				Expect(err).NotTo(HaveOccurred())
				tf.ShowInputReference(false, 2, 2, 80)

				// close piped output
				os.Stdout.Close()
				io.Copy(&output, stdOutReader)

				// signal end
				out <- output.String()
			}()

			// wait until signal is received

			output := <-out
			logger.DebugMessage("\n%s\n", output)
			Expect(output).To(Equal(testFormReferenceOutput))
		})
	})

	Context("Input", func() {

		var (
			inputGroup *entry.InputGroup
		)

		BeforeEach(func() {

			inputGroup = test_data.NewTestInputCollection().Group("input-form")

			// Bind fields to map of values so
			// that form values can be saved
			inputValues := make(map[string]*string)
			for _, f := range inputGroup.InputFields() {
				s := new(string)
				inputValues[f.Name()] = s
				err = f.SetValueRef(s)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("gathers intput for the form from stdin", func() {

			expectedValues := map[string]string{
				"attrib12":   "value for attrib12",
				"attrib122":  "value for attrib122",
				"attrib1221": "value for attrib1221",
				"attrib131":  "value for attrib131",
				"attrib1311": "value for attrib1311",
				"attrib1312": "value for attrib1312",
				"attrib14":   "value for attrib14",
			}

			go func() {

				tf, err := ux.NewTextForm(
					"Input Data Form for 'input-form'",
					"CONFIGURATION DATA INPUT",
					inputGroup,
				)
				if err == nil {
					err = tf.GetInput(false, 2, 80)
				}
				if err != nil {
					fmt.Println(err.Error())
				}
			}()

			outputReader := bufio.NewScanner(stdOutReader)
			expectReader := bufio.NewScanner(bytes.NewBufferString(testFormInputPrompts))
			for expectReader.Scan() {
				expected := expectReader.Text()

				if strings.HasPrefix(expected, "<<") {
					input := expected[2:] + "\n"
					fmt.Fprint(os.Stdout, input)
					stdInWriter.WriteString(input)

					if outputReader.Scan() {
						logger.TraceMessage("expect> %s\n", outputReader.Text())
					}

				} else {
					if !outputReader.Scan() {
						Fail(fmt.Sprintf("TextFrom GetInput() did not output expected string '%s'.", expected))
					}
					actual := outputReader.Text()
					logger.TraceMessage("expect> %s\n", actual)
					Expect(expected).To(Equal(actual))
				}
			}

			values := inputGroup.InputValues()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(values)).To(Equal(len(expectedValues)))
			Expect(reflect.DeepEqual(expectedValues, values)).To(BeTrue())
		})
	})
})

const testFormReferenceOutput = `  Input Data Form for 'input-form'
  ================================

  test group description

  CONFIGURATION DATA INPUT

  * Provide one of the following for:

    description for group 1

    * Attrib 11 - description for attrib11. It will be sourced from the
                  environment variables ATTRIB11_ENV1, ATTRIB11_ENV2,
                  ATTRIB11_ENV3 if not provided.

    OR

    * Attrib 12 - description for attrib12. It will be sourced from the
                  environment variable ATTRIB12_ENV1 if not provided.

    * Provide one of the following for:

      description for group 2

      * Attrib 121 - description for attrib121.

      OR

      * Attrib 122  - description for attrib122.
      * Attrib 1221 - description for attrib1221.

    * Attrib 131  - description for attrib131.
    * Attrib 1311 - description for attrib1311.
    * Attrib 1312 - description for attrib1312.

    OR

    * Attrib 13   - description for attrib13. It will be sourced from the
                    environment variables ATTRIB13_ENV1, ATTRIB13_ENV2 if not
                    provided.
    * Attrib 131  - description for attrib131.
    * Attrib 1311 - description for attrib1311.
    * Attrib 1312 - description for attrib1312.

    * Provide one of the following for:

      description for group 3

      * Attrib 132 - description for attrib131.

      OR

      * Attrib 133 - description for attrib131.

  * Attrib 14 - description for attrib14.`

const testFormInputPrompts = `Input Data Form for 'input-form'
================================

test group description

CONFIGURATION DATA INPUT
================================================================================

description for group 1
================================================================================
1. Attrib 11 - description for attrib11. It will be sourced from the environment
               variables ATTRIB11_ENV1, ATTRIB11_ENV2, ATTRIB11_ENV3 if not
               provided.
--------------------------------------------------------------------------------
2. Attrib 12 - description for attrib12. It will be sourced from the environment
               variable ATTRIB12_ENV1 if not provided.
--------------------------------------------------------------------------------
3. Attrib 13 - description for attrib13. It will be sourced from the environment
               variables ATTRIB13_ENV1, ATTRIB13_ENV2 if not provided.
--------------------------------------------------------------------------------
<<2
--------------------------------------------------------------------------------
<<value for attrib12

description for group 2
================================================================================
1. Attrib 121 - description for attrib121.
--------------------------------------------------------------------------------
2. Attrib 122 - description for attrib122.
--------------------------------------------------------------------------------
<<2
--------------------------------------------------------------------------------
<<value for attrib122

Attrib 1221 - description for attrib1221.
--------------------------------------------------------------------------------
<<value for attrib1221

Attrib 131 - description for attrib131.
--------------------------------------------------------------------------------
<<value for attrib131

Attrib 1311 - description for attrib1311.
--------------------------------------------------------------------------------
<<value for attrib1311

Attrib 1312 - description for attrib1312.
--------------------------------------------------------------------------------
<<value for attrib1312

Attrib 14 - description for attrib14.
--------------------------------------------------------------------------------
<<value for attrib14

================================================================================`
