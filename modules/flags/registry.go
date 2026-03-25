package flags

import (
	"fmt"
	"sort"
	"sync"
)

type FlagKey string

type FlagDefinition struct {
	Key         FlagKey
	Description string
	Default     bool
}

var (
	registryMu sync.RWMutex
	registry   = map[FlagKey]FlagDefinition{}
)

func Register(def FlagDefinition) FlagKey {
	if def.Key == "" {
		panic("flags: register requires non-empty key")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[def.Key]; exists {
		panic(fmt.Sprintf("flags: duplicate registration for key %q", def.Key))
	}
	registry[def.Key] = def
	return def.Key
}

func All() []FlagDefinition {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make([]FlagDefinition, 0, len(registry))
	for _, def := range registry {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out
}

func Lookup(key FlagKey) (FlagDefinition, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	def, ok := registry[key]
	return def, ok
}

func resetRegistryForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[FlagKey]FlagDefinition{}
}

