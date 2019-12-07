package utils_test

import (
	"github.com/mevansam/goutils/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("string utils tests", func() {

	Context("joining list of items to a sentance", func() {

		It("creates output of unquoted items", func() {

			s := utils.JoinListAsSentence(
				"John ate %s for breakfast.",
				[]string{"eggs", "bacon", "baked beans"},
				false,
			)

			Expect(s).To(Equal("John ate eggs, bacon and baked beans for breakfast."))
		})

		It("creates output of quoted items", func() {

			s := utils.JoinListAsSentence(
				"The attendees names were %s.",
				[]string{"Jack", "Jill", "Jane", "Mike"},
				true,
			)

			Expect(s).To(Equal("The attendees names were \"Jack\", \"Jill\", \"Jane\" and \"Mike\"."))
		})
	})

	Context("wrapping a long string with indentation", func() {

		It("splits and indents all lines except the first of a long string", func() {

			s, l := utils.SplitString(
				"Terraform is a tool for building, changing, and versioning infrastructure safely and efficiently. Terraform can manage existing and popular service providers as well as custom in-house solutions.",
				11, 80, false,
			)
			Expect(l).To(BeTrue())

			Expect(s).To(
				Equal(
					`Terraform is a tool for building, changing, and versioning infrastructure safely
           and efficiently. Terraform can manage existing and popular service providers as
           well as custom in-house solutions.`,
				),
			)
		})

		It("splits and indents all lines of a long string", func() {

			s, l := utils.SplitString(
				"Terraform is a tool for building, changing, and versioning infrastructure safely and efficiently. Terraform can manage existing and popular service providers as well as custom in-house solutions.",
				11, 80, true,
			)
			Expect(l).To(BeTrue())

			Expect(s).To(
				Equal(
					`           Terraform is a tool for building, changing, and versioning infrastructure safely
           and efficiently. Terraform can manage existing and popular service providers as
           well as custom in-house solutions.`,
				),
			)
		})

		It("removes whitespace at split", func() {

			s, l := utils.SplitString(
				"Terraform is a tool for building, changing, and versioning infrastructure            and efficiently. Terraform can manage existing and popular service providers as well as custom in-house solutions.",
				11, 80, false,
			)
			Expect(l).To(BeTrue())

			Expect(s).To(
				Equal(
					`Terraform is a tool for building, changing, and versioning infrastructure
           and efficiently. Terraform can manage existing and popular service providers as
           well as custom in-house solutions.`,
				),
			)
		})
	})
})
