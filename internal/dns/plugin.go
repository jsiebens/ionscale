package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-plugin"
	"github.com/jsiebens/ionscale/internal/util"
	dnsplugin "github.com/jsiebens/libdns-plugin"
	"github.com/libdns/libdns"
	"go.uber.org/zap"
	"os/exec"
	"sync"
	"time"
)

// pluginManager handles plugin lifecycle and resilience
type pluginManager struct {
	pluginPath string
	client     *plugin.Client
	instance   dnsplugin.Provider
	lock       sync.RWMutex
	logger     *zap.Logger

	zone   string
	config json.RawMessage
}

// NewPluginManager creates a new plugin manager
func newPluginManager(pluginPath string, zone string, config json.RawMessage) (*pluginManager, error) {
	logger := zap.L().Named("dns").With(zap.String("plugin_path", pluginPath))

	p := &pluginManager{
		pluginPath: pluginPath,
		logger:     logger,
		zone:       zone,
		config:     config,
	}

	if err := p.ensureRunning(true); err != nil {
		return nil, err
	}

	return p, nil
}

// ensureRunning makes sure the plugin is running
func (pm *pluginManager) ensureRunning(start bool) error {
	pm.lock.RLock()
	running := pm.client != nil && !pm.client.Exited()
	instance := pm.instance
	pm.lock.RUnlock()

	if running && instance != nil {
		return nil
	}

	// Need to restart
	pm.lock.Lock()
	defer pm.lock.Unlock()

	if !start {
		pm.logger.Info("Restarting DNS plugin")
	}

	if pm.client != nil {
		pm.client.Kill()
	}

	// Create a new client
	cmd := exec.Command(pm.pluginPath)
	pm.client = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: dnsplugin.Handshake,
		Plugins:         dnsplugin.PluginMap,
		Cmd:             cmd,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
			plugin.ProtocolGRPC,
		},
		Managed: true,
		Logger:  util.NewZapAdapter(pm.logger, "dns"),
	})

	// Connect via RPC
	rpcClient, err := pm.client.Client()
	if err != nil {
		return fmt.Errorf("error creating plugin client: %w", err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(dnsplugin.ProviderPluginName)
	if err != nil {
		return fmt.Errorf("error dispensing plugin: %w", err)
	}

	// Convert to the interface
	pm.instance = raw.(dnsplugin.Provider)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := pm.instance.Configure(ctx, pm.config); err != nil {
		return fmt.Errorf("error configuring plugin: %w", err)
	}

	pm.logger.Info("DNS plugin started")

	return nil
}

func (pm *pluginManager) SetRecord(ctx context.Context, recordType, recordName, value string) error {
	if err := pm.ensureRunning(false); err != nil {
		return err
	}

	_, err := pm.instance.SetRecords(ctx, pm.zone, []libdns.Record{{
		Type:  recordType,
		Name:  libdns.RelativeName(recordName, pm.zone),
		Value: value,
		TTL:   1 * time.Minute,
	}})

	return err
}
