package collections

import (
	"fmt"
	"strconv"

	"github.com/mevansam/goutils/data/persistence"
)

type Array []interface{}

func NewArray() Array {
	return []interface{}{}
}

func (a Array) Unmarshal(
	path []string,
	key string,
	elemType persistence.ElementType,
	value interface{},
) (
	persistence.Unmarshaller,
	persistence.Unmarshaller,
	error,
) {

	switch elemType {
	case persistence.EtObject:
		mm := NewMap()
		return mm, append(a, mm), nil

	case persistence.EtArray:
		aa := NewArray()
		return aa, append(a, aa), nil

	case persistence.EtValue, persistence.EtArrayValue:
		return a, append(a, value), nil

	default:
		return nil, nil, fmt.Errorf("invalid type for Array container: %#v", elemType)
	}
}

func (a Array) Finalize(
	path []string,
	key string,
	node persistence.Unmarshaller,
) error {

	var (
		err        error
		arrayIndex int
	)

	switch node.(type) {
	case Array:
		if arrayIndex, err = strconv.Atoi(key[1:]); err != nil {

			return fmt.Errorf("expected array element to be finalized but index '%s' was invalid: %s",
				key, err.Error())
		}
		a[arrayIndex] = node
	}
	return nil
}
