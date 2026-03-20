package emailsubscriptions

import (
	"fmt"
	"strings"
)

type ListSpec struct {
	Key    List
	Active bool
}

type ListCatalog interface {
	ListByKey(key List) (ListSpec, bool)
	Lists() []ListSpec
}

type StaticListCatalog struct {
	byKey map[List]ListSpec
	all   []ListSpec
}

func NewStaticListCatalog(specs []ListSpec) (*StaticListCatalog, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("list catalog cannot be empty")
	}
	byKey := make(map[List]ListSpec, len(specs))
	all := make([]ListSpec, 0, len(specs))
	for _, spec := range specs {
		key := NormalizeList(spec.Key)
		if key == "" {
			return nil, fmt.Errorf("list key cannot be empty")
		}
		spec.Key = key
		if _, exists := byKey[key]; exists {
			return nil, fmt.Errorf("duplicate list key %q", key)
		}
		byKey[key] = spec
		all = append(all, spec)
	}
	return &StaticListCatalog{byKey: byKey, all: all}, nil
}

func (c *StaticListCatalog) ListByKey(key List) (ListSpec, bool) {
	spec, ok := c.byKey[NormalizeList(key)]
	return spec, ok
}

func (c *StaticListCatalog) Lists() []ListSpec {
	out := make([]ListSpec, len(c.all))
	copy(out, c.all)
	return out
}

func NormalizeList(list List) List {
	return List(strings.ToLower(strings.TrimSpace(string(list))))
}
