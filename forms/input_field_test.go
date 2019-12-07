package forms_test

import (
	"os"
	"path/filepath"

	"github.com/mevansam/goutils/forms"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	test_data "github.com/mevansam/goutils/test/forms"
)

var _ = Describe("Input Fields", func() {

	var (
		err error
		ic  *forms.InputCollection
		ig  *forms.InputGroup
	)

	BeforeEach(func() {
		ic = test_data.NewTestInputCollection()
		ig = ic.Group("input-form")
	})

	Context("input field retrieval and updates", func() {

		It("can retrieve fields as expected", func() {

			var (
				field *forms.InputField
			)

			field, err = ig.GetInputField("attrib121")
			Expect(err).NotTo(HaveOccurred())
			Expect(field.Name()).To(Equal("attrib121"))

			field, err = ig.GetInputField("attrib1221")
			Expect(err).NotTo(HaveOccurred())
			Expect(field.Name()).To(Equal("attrib1221"))

			field, err = ig.GetInputField("attrib1312")
			Expect(err).NotTo(HaveOccurred())
			Expect(field.Name()).To(Equal("attrib1312"))

			_, err = ig.GetInputField("attrib1411")
			Expect(err).To(HaveOccurred())
		})

		It("gets and sets field bound to an external data structure", func() {

			var (
				field *forms.InputField
				value *string

				newValue string
			)

			attrib11Value := "attrib11 #1"

			data := struct {
				attrib11 *string
				attrib12 string
				attrib13 *string
			}{
				attrib11: &attrib11Value,
				attrib12: "attrib12 #2",
				attrib13: nil,
			}

			field, err = ig.GetInputField("attrib11")
			Expect(err).NotTo(HaveOccurred())
			err = field.SetValueRef(&data.attrib11)
			Expect(err).NotTo(HaveOccurred())
			value = field.Value()
			Expect(*value).To(Equal("attrib11 #1"))

			// value update in struct should reflect
			// when retrieved via InputForm
			attrib11Value = "attrib11 #2"
			Expect(*data.attrib11).To(Equal("attrib11 #2"))
			value = field.Value()
			Expect(*value).To(Equal("attrib11 #2"))

			// value update in input form
			// should reflect in struct
			newValue = "attrib11 #3"
			err = field.SetValue(&newValue)
			Expect(*data.attrib11).To(Equal("attrib11 #3"))

			field, err = ig.GetInputField("attrib12")
			Expect(err).NotTo(HaveOccurred())
			err = field.SetValueRef(&data.attrib12)
			Expect(err).NotTo(HaveOccurred())
			value = field.Value()
			Expect(*value).To(Equal("attrib12 #2"))

			// value update in struct should reflect
			// when retrieved via InputForm
			data.attrib12 = "attrib12 #2"
			value = field.Value()
			Expect(*value).To(Equal("attrib12 #2"))

			// value update in input form
			// should reflect in struct
			newValue = "attrib12 #3"
			err = field.SetValue(&newValue)
			Expect(data.attrib12).To(Equal("attrib12 #3"))

			field, err = ig.GetInputField("attrib13")
			Expect(err).NotTo(HaveOccurred())
			err = field.SetValueRef(&data.attrib13)
			Expect(err).NotTo(HaveOccurred())
			value = field.Value()
			Expect(value).To(BeNil())

			// value update in input form
			// should reflect in struct
			newValue = "attrib13 #1"
			err = field.SetValue(&newValue)
			Expect(err).NotTo(HaveOccurred())
			Expect(*data.attrib13).To(Equal("attrib13 #1"))

			data.attrib13 = nil
			value = field.Value()
			Expect(value).To(BeNil())

			// value update in struct should reflect
			// when retrieved via InputForm
			newValue = "attrib13 #2"
			data.attrib13 = &newValue
			value = field.Value()
			Expect(*value).To(Equal("attrib13 #2"))
		})

		It("sources field value from a file with path sourced from environment", func() {

			var (
				attrib132Value string
				value          *string
			)

			field, err := ig.GetInputField("attrib132")
			Expect(err).NotTo(HaveOccurred())
			err = field.SetValueRef(&attrib132Value)
			Expect(err).NotTo(HaveOccurred())

			attrib132FilePath, err := filepath.Abs(workingDirectory + "/../test/fixtures/data/attrib132")
			Expect(err).NotTo(HaveOccurred())
			os.Setenv("ATTRIB132", attrib132FilePath)

			valueFromFile, paths := field.ValueFromFile()
			Expect(valueFromFile).To(BeTrue())
			Expect(len(paths)).To(Equal(1))
			Expect(paths[0]).To(Equal(attrib132FilePath))

			err = field.SetValue(&paths[0])
			Expect(err).NotTo(HaveOccurred())
			value = field.Value()
			Expect(*value).To(Equal(`{"attrib132":"value for attrib132 from file"}`))
		})

		It("sources field value from a file with path sourced from environment", func() {

			var (
				attrib132Value string
				attrib133Value string
				value          *string
			)

			field, err := ig.GetInputField("attrib132")
			Expect(err).NotTo(HaveOccurred())
			err = field.SetValueRef(&attrib132Value)
			Expect(err).NotTo(HaveOccurred())

			field, err = ig.GetInputField("attrib133")
			Expect(err).NotTo(HaveOccurred())
			err = field.SetValueRef(&attrib133Value)
			Expect(err).NotTo(HaveOccurred())
			Expect(attrib133Value).To(Equal("default value for attrib133"))

			// hint is value of parsed json context of field 'attrib132' having key 'attrib132'
			err = ig.AddFieldValueHint("attrib133", "field://attrib132/attrib132")
			Expect(err).NotTo(HaveOccurred())

			attrib132FilePath, err := filepath.Abs(workingDirectory + "/../test/fixtures/data/attrib132")
			Expect(err).NotTo(HaveOccurred())
			err = ig.SetFieldValue("attrib132", attrib132FilePath)
			Expect(err).NotTo(HaveOccurred())

			hintValues, err := ig.GetFieldValueHints("attrib133")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(hintValues)).To(Equal(1))

			err = ig.SetFieldValue("attrib133", hintValues[0])
			Expect(err).NotTo(HaveOccurred())
			value, err = ig.GetFieldValue("attrib133")
			Expect(err).NotTo(HaveOccurred())
			Expect(*value).To(Equal("value for attrib132 from file"))
		})
	})

	Context("input field validation", func() {

		BeforeEach(func() {

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

		It("restricts field values to a list of accepted values", func() {

			var (
				field *forms.InputField
				value string
			)

			field, err = ig.GetInputField("attrib11")
			Expect(err).NotTo(HaveOccurred())

			acceptedValues := []string{"aa", "bb", "cc"}
			field.SetAcceptedValues(&acceptedValues, "error")

			value = "dd"
			err = field.SetValue(&value)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("error"))

			value = "bb"
			err = field.SetValue(&value)
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates field values using an inclusion filter", func() {

			var (
				field *forms.InputField
				value string
			)

			field, err = ig.GetInputField("attrib11")
			Expect(err).NotTo(HaveOccurred())

			err = field.SetInclusionFilter("(gopher){2}", "error")
			Expect(err).ToNot(HaveOccurred())

			value = "gopher"
			err = field.SetValue(&value)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("error"))

			value = "gophergophergopher"
			err = field.SetValue(&value)
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates field values using an exclusion filter", func() {

			var (
				field *forms.InputField
				value string
			)

			field, err = ig.GetInputField("attrib11")
			Expect(err).NotTo(HaveOccurred())

			err = field.SetExclusionFilter("(gopher){2}", "error")
			Expect(err).ToNot(HaveOccurred())

			value = "gophergophergopher"
			err = field.SetValue(&value)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("error"))

			value = "gopher"
			err = field.SetValue(&value)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
