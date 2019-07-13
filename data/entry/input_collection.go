package entry

type InputCollection struct {
	groups map[string]*InputGroup
}

func NewInputCollection() *InputCollection {
	return &InputCollection{
		groups: make(map[string]*InputGroup),
	}
}

func (ic *InputCollection) NewGroup(
	name string,
	description string,
) *InputGroup {

	ig := &InputGroup{
		name:        name,
		description: description,
		inputs:      []Input{},

		containers:   make(map[int]*InputGroup),
		fieldNameSet: make(map[string]Input),
	}
	ic.groups[name] = ig
	return ig
}

func (ic *InputCollection) HasGroup(name string) bool {
	_, exists := ic.groups[name]
	return exists
}

func (ic *InputCollection) Group(name string) *InputGroup {
	return ic.groups[name]
}

func (ic *InputCollection) Groups() []*InputGroup {

	groupList := []*InputGroup{}
	for _, g := range ic.groups {
		groupList = append(groupList, g)
	}
	return groupList
}
