package entry_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/mevansam/goutils/data/entry"
	"github.com/mevansam/goutils/logger"
	test_data "github.com/mevansam/goutils/test/data"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Input Collection", func() {

	var (
		err error
		ic  *entry.InputCollection
		ig  *entry.InputGroup
	)

	BeforeEach(func() {
		ic = test_data.NewTestInputCollection()
		ig = ic.Group("input-form")
	})

	Context("collection group creation", func() {

		It("returns groups that have have been created correctly", func() {

			expectedGroups := map[string]string{
				"input-form":  "test group description",
				"input-form2": "input form 2 description",
				"input-form3": "input form 3 description",
			}
			groups := ic.Groups()
			Expect(len(groups)).To(Equal(3))

			for _, g := range groups {
				Expect(expectedGroups[g.Name()]).To(Equal(g.Description()))
			}
		})
	})

	Context("input group creation and access", func() {

		It("input group hierarchy is as expected", func() {

			hierarchyMap := map[string][]string{
				"input-form": []string{"group1", "attrib14"},
				"group1":     []string{"attrib11", "attrib12", "attrib13"},
				"group2":     []string{"attrib121", "attrib122"},
				"group3":     []string{"attrib132", "attrib133"},
				"attrib11":   []string{},
				"attrib12":   []string{"group2", "attrib131"},
				"attrib13":   []string{"attrib131", "group3"},
				"attrib14":   []string{},
				"attrib121":  []string{},
				"attrib122":  []string{"attrib1221"},
				"attrib131":  []string{"attrib1311", "attrib1312"},
				"attrib1221": []string{},
				"attrib1311": []string{},
				"attrib1312": []string{},
			}

			var (
				validateInput func(indent int, input entry.Input)
			)

			validateInput = func(indent int, input entry.Input) {

				names := hierarchyMap[input.Name()]

				var out strings.Builder
				for i := 0; i < indent; i++ {
					out.WriteByte(' ')
				}
				out.WriteString("- ")
				out.WriteString(input.Name())
				out.WriteString(fmt.Sprintf(" : should contain %v", names))
				logger.DebugMessage(out.String())

				for i, ii := range input.Inputs() {
					validateInput(indent+2, ii)
					Expect(ii.Name()).To(Equal(names[i]))
				}
			}
			logger.DebugMessage("\n*** Validating input group hierarchy ***\n\n")
			validateInput(0, ig)
		})

		It("returns an ordered unique list of all attributes", func() {

			// Expected order is. Traversal of input fields will be
			// in order of least significant field in the hierarchy
			//
			// 1.  attrib11
			// 2.  attrib12
			// 3.  attrib121
			// 4.  attrib122
			// 5.  attrib1221
			// 6.  attrib131
			// 7.  attrib1311
			// 8.  attrib1312
			// 9.  attrib13
			// 10. attrib14

			expectedOrder := []string{
				"attrib11",
				"attrib12",
				"attrib121",
				"attrib122",
				"attrib1221",
				"attrib131",
				"attrib1311",
				"attrib1312",
				"attrib13",
				"attrib132",
				"attrib133",
				"attrib14",
			}

			ff := ig.InputFields()
			Expect(len(ff)).To(Equal(len(expectedOrder)))
			for i, f := range expectedOrder {
				Expect(ff[i].Name()).To(Equal(f))
			}
		})
	})

	Context("gather inputs", func() {

		var (
			input  entry.Input
			cursor *entry.InputCursor

			advanceCursorPositionAndValidate func(cursor *entry.InputCursor, inputName string) *entry.InputCursor
		)

		BeforeEach(func() {

			advanceCursorPositionAndValidate = func(cursor *entry.InputCursor, inputName string) *entry.InputCursor {

				cursor = cursor.NextInput()
				Expect(cursor).NotTo(BeNil())

				input, err = cursor.GetCurrentInput()
				Expect(err).ToNot(HaveOccurred())
				Expect(input.Name()).To(Equal(inputName))

				return cursor
			}

			os.Setenv("ATTRIB13_ENV2", "value for attrib13 from env")

			// Bind fields to map of values so
			// that form values can be saved
			inputValues := make(map[string]*string)
			for _, f := range ig.InputFields() {
				s := new(string)
				inputValues[f.Name()] = s
				err = f.SetValueRef(s)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		AfterEach(func() {
			os.Unsetenv("ATTRIB13_ENV2")
		})

		It("navigates path option #1", func() {

			expectedValues := map[string]string{
				"attrib11": "value for attrib11",
				"attrib14": "value for attrib14",
			}

			cursor, err = entry.NewInputCursorFromCollection("input-form", ic)
			Expect(err).NotTo(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "group1")
			cursor, err = cursor.SetInput("attrib10", "value for attrib10")
			Expect(err).To(HaveOccurred())
			cursor, err = cursor.SetInput("attrib11", "value for attrib11")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "attrib14")
			cursor, err = cursor.SetInput("attrib11", "value for attrib11")
			Expect(err).To(HaveOccurred())
			cursor, err = cursor.SetInput("attrib14", "value for attrib14")
			Expect(err).ToNot(HaveOccurred())

			values := ig.InputValues()
			Expect(len(values)).To(Equal(len(expectedValues)))
			Expect(reflect.DeepEqual(expectedValues, values)).To(BeTrue())
		})

		It("navigates path option #2", func() {

			expectedValues := map[string]string{
				"attrib12":   "value for attrib12",
				"attrib122":  "value for attrib122",
				"attrib1221": "value for attrib1221",
				"attrib131":  "value for attrib131",
				"attrib1311": "value for attrib1311",
				"attrib1312": "value for attrib1312",
				"attrib14":   "default value for attrib14",
			}

			cursor, err = entry.NewInputCursorFromCollection("input-form", ic)
			Expect(err).NotTo(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "group1")
			cursor, err = cursor.SetInput("attrib12", "value for attrib12")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "group2")
			cursor, err = cursor.SetInput("attrib122", "value for attrib122")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "attrib1221")
			cursor, err = cursor.SetInput("attrib1221", "value for attrib1221")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "attrib131")
			cursor, err = cursor.SetInput("attrib131", "value for attrib131")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "attrib1311")
			cursor, err = cursor.SetInput("attrib1311", "value for attrib1311")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "attrib1312")
			cursor, err = cursor.SetInput("attrib1312", "value for attrib1312")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "attrib14")
			cursor, err = cursor.SetDefaultInput("attrib14")
			Expect(err).ToNot(HaveOccurred())

			values := ig.InputValues()
			Expect(len(values)).To(Equal(len(expectedValues)))
			Expect(reflect.DeepEqual(expectedValues, values)).To(BeTrue())
		})

		It("navigates path option #3", func() {

			expectedValues := map[string]string{
				"attrib13":   "value for attrib13 from env",
				"attrib131":  "value for attrib131",
				"attrib1311": "value for attrib1311",
				"attrib1312": "value for attrib1312",
				"attrib132":  "value for attrib132 from file",
				"attrib14":   "value for attrib14",
			}

			cursor, err = entry.NewInputCursorFromCollection("input-form", ic)
			Expect(err).NotTo(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "group1")
			cursor, err = cursor.SetDefaultInput("attrib13")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "attrib131")
			cursor, err = cursor.SetInput("attrib131", "value for attrib131")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "attrib1311")
			cursor, err = cursor.SetInput("attrib1311", "value for attrib1311")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "attrib1312")
			cursor, err = cursor.SetInput("attrib1312", "value for attrib1312")
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "group3")
			attrib132FilePath, err := filepath.Abs(workingDirectory + "/../../test/fixtures/data/attrib132")
			Expect(err).NotTo(HaveOccurred())
			cursor, err = cursor.SetInput("attrib132", attrib132FilePath)
			Expect(err).ToNot(HaveOccurred())

			cursor = advanceCursorPositionAndValidate(cursor, "attrib14")
			cursor, err = cursor.SetInput("attrib14", "value for attrib14")
			Expect(err).ToNot(HaveOccurred())

			values := ig.InputValues()
			Expect(len(values)).To(Equal(len(expectedValues)))
			Expect(reflect.DeepEqual(expectedValues, values)).To(BeTrue())
		})
	})
})
