package forms

import (
	"github.com/mevansam/goutils/forms"

	. "github.com/onsi/gomega"
)

func NewTestInputCollection() *forms.InputCollection {

	var (
		err error
		ic  *forms.InputCollection
		ig  *forms.InputGroup
	)

	ic = forms.NewInputCollection()

	ig = ic.NewGroup("input-form", "test group description")
	ic.NewGroup("input-form2", "input form 2 description")
	ic.NewGroup("input-form3", "input form 3 description")

	// Input Paths (name group)
	//
	// attrib11 1 -> X
	// attrib12 1 -> attrib121 2 -> X
	//            -> attrib122 2 -> attrib1221 -> 0 X
	//            -> attrib131 0 -> attrib1311 -> 0 X
	//                           -> attrib1312 -> 0 X
	// attrib13 1 -> attrib131 0 -> attrib1311 -> 0 X
	//                           -> attrib1312 -> 0 X
	//            -> attrib132 3 -> X
	//            -> attrib133 3 -> X
	// attrib14 0 -> X

	ig.NewInputContainer(
		/* name */ "group1",
		/* displayName */ "Group 1",
		/* description */ "description for group 1",
		/* groupId */ 1,
	)
	ig.NewInputContainer(
		/* name */ "group2",
		/* displayName */ "Group 2",
		/* description */ "description for group 2",
		/* groupId */ 2,
	)
	ig.NewInputContainer(
		/* name */ "group3",
		/* displayName */ "Group 3",
		/* description */ "description for group 3",
		/* groupId */ 3,
	)

	_, err = ig.NewInputGroupField(
		/* name */ "attrib11",
		/* displayName */ "Attrib 11",
		/* description */ "description for attrib11.",
		/* groupId */ 1,
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{
			"ATTRIB11_ENV1",
			"ATTRIB11_ENV2",
			"ATTRIB11_ENV3",
		},
		/* dependsOn */ []string{},
	)
	Expect(err).NotTo(HaveOccurred())
	_, err = ig.NewInputGroupField(
		/* name */ "attrib12",
		/* displayName */ "Attrib 12",
		/* description */ "description for attrib12.",
		/* groupId */ 1,
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{
			"ATTRIB12_ENV1",
		},
		/* dependsOn */ []string{},
	)
	Expect(err).NotTo(HaveOccurred())
	_, err = ig.NewInputGroupField(
		/* name */ "attrib13",
		/* displayName */ "Attrib 13",
		/* description */ "description for attrib13.",
		/* groupId */ 1,
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{
			"ATTRIB13_ENV1",
			"ATTRIB13_ENV2",
		},
		/* dependsOn */ []string{},
	)
	Expect(err).NotTo(HaveOccurred())
	_, err = ig.NewInputFieldWithDefaultValue(
		/* name */ "attrib14",
		/* displayName */ "Attrib 14",
		/* description */ "description for attrib14.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* defaultValue */ "default value for attrib14",
		/* envVars */ []string{},
		/* dependsOn */ []string{},
	)
	Expect(err).NotTo(HaveOccurred())
	_, err = ig.NewInputGroupField(
		/* name */ "attrib121",
		/* displayName */ "Attrib 121",
		/* description */ "description for attrib121.",
		/* groupId */ 2,
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{},
		/* dependsOn */ []string{"attrib12"},
	)
	Expect(err).NotTo(HaveOccurred())
	_, err = ig.NewInputGroupField(
		/* name */ "attrib122",
		/* displayName */ "Attrib 122",
		/* description */ "description for attrib122.",
		/* groupId */ 2,
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{},
		/* dependsOn */ []string{"attrib12"},
	)
	Expect(err).NotTo(HaveOccurred())
	_, err = ig.NewInputField(
		/* name */ "attrib131",
		/* displayName */ "Attrib 131",
		/* description */ "description for attrib131.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{},
		/* dependsOn */ []string{"attrib12", "attrib13"},
	)
	Expect(err).NotTo(HaveOccurred())
	_, err = ig.NewInputGroupField(
		/* name */ "attrib132",
		/* displayName */ "Attrib 132",
		/* description */ "description for attrib132.",
		/* groupId */ 3,
		/* inputType */ forms.String,
		/* valueFromFile */ true,
		/* envVars */ []string{"ATTRIB132"},
		/* dependsOn */ []string{"attrib13"},
	)
	Expect(err).NotTo(HaveOccurred())
	_, err = ig.NewInputGroupFieldWithDefaultValue(
		/* name */ "attrib133",
		/* displayName */ "Attrib 133",
		/* description */ "description for attrib133.",
		/* groupId */ 3,
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* defaultValue */ "default value for attrib133",
		/* envVars */ []string{},
		/* dependsOn */ []string{"attrib13"},
	)
	Expect(err).NotTo(HaveOccurred())

	_, err = ig.NewInputField(
		/* name */ "attrib1221",
		/* displayName */ "Attrib 1221",
		/* description */ "description for attrib1221.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{},
		/* dependsOn */ []string{"attrib122"},
	)
	Expect(err).NotTo(HaveOccurred())
	_, err = ig.NewInputField(
		/* name */ "attrib1311",
		/* displayName */ "Attrib 1311",
		/* description */ "description for attrib1311.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{},
		/* dependsOn */ []string{"attrib131"},
	)
	Expect(err).NotTo(HaveOccurred())
	_, err = ig.NewInputField(
		/* name */ "attrib1312",
		/* displayName */ "Attrib 1312",
		/* description */ "description for attrib1312.",
		/* inputType */ forms.String,
		/* valueFromFile */ false,
		/* envVars */ []string{},
		/* dependsOn */ []string{"attrib131"},
	)
	Expect(err).NotTo(HaveOccurred())

	return ic
}
