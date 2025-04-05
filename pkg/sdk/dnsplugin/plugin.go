package dnsplugin

import (
	"fmt"
	"github.com/libdns/libdns"
	"sync"
)

type Provider interface {
	libdns.RecordGetter
	libdns.RecordAppender
	libdns.RecordSetter
	libdns.RecordDeleter
}

type Plugin interface {
	Plugin() PluginInfo
}

type PluginInfo struct {
	ID  string
	New func() Provider
}

type Registry struct {
	mu      sync.RWMutex
	plugins map[string]PluginInfo
}

func Register(plugin Plugin) {
	register(plugin.Plugin())
}

func RegisterFunc(id string, g func() Provider) {
	register(PluginInfo{ID: id, New: g})
}

func Get(name string) *PluginInfo {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	plugin, exists := registry.plugins[name]
	if !exists {
		return nil
	}

	return &plugin
}

func register(info PluginInfo) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, exists := registry.plugins[info.ID]; exists {
		panic(fmt.Sprintf("dns plugin already registered: %s", info.ID))
	}

	registry.plugins[info.ID] = info
}

var (
	registry = &Registry{
		plugins: make(map[string]PluginInfo),
	}
)
