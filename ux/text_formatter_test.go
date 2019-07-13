package ux_test

import (
	"strings"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/ux"
	. "github.com/onsi/ginkgo"

	test_data "github.com/mevansam/goutils/test/data"
	. "github.com/onsi/gomega"
)

var _ = Describe("Text Formatting tests", func() {

	var (
		outputBuffer strings.Builder
	)

	BeforeEach(func() {
		outputBuffer.Reset()
	})

	It("outputs a detailed input data form reference", func() {

		ic := test_data.NewTestInputCollection()

		tf := ux.NewTextFormatter()
		tf.ShowDataInputForm(
			"Input Data Form Reference for 'input-form'",
			"CONFIGURATION DATA INPUT REFERENCE",
			2, 2, 80, ic.Group("input-form"), &outputBuffer)

		output := outputBuffer.String()
		logger.DebugMessage("\n%s\n", output)
		Expect(output).To(Equal(testInputDataReferenceOutput))
	})
})

const testInputDataReferenceOutput = `  Input Data Form Reference for 'input-form'
  ==========================================

  test group description

  CONFIGURATION DATA INPUT REFERENCE

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
