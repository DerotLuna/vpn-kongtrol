package vpn

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	adapters = make(map[string]func() VPNAdapter)
)

// Register registers a VPN adapter factory under name.
// Called from each adapter package's init() function.
// Panics if the same name is registered twice (programming error).
func Register(name string, factory func() VPNAdapter) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := adapters[name]; exists {
		panic(fmt.Sprintf("vpn: adapter %q already registered", name))
	}
	adapters[name] = factory
}

// New creates a new adapter instance by name.
// Returns an error if the name is unknown — which usually means the adapter
// package was not imported (blank-import it in the binary's main.go).
func New(name string) (VPNAdapter, error) {
	mu.RLock()
	factory, ok := adapters[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("vpn: unknown adapter %q — import the adapter package in main.go", name)
	}
	return factory(), nil
}

// Registered returns the sorted list of registered adapter names.
func Registered() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(adapters))
	for name := range adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
