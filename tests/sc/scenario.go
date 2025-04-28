package sc

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	petname "github.com/dustinkirkland/golang-petname"
	ionscaleclt "github.com/jsiebens/ionscale/pkg/client/ionscale"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	ionscaleconnect "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1/ionscalev1connect"
	"github.com/jsiebens/ionscale/tests/tsn"
	"github.com/jsiebens/mockoidc"
	mockoidcv1 "github.com/jsiebens/mockoidc/pkg/gen/mockoidc/v1"
	"github.com/jsiebens/mockoidc/pkg/gen/mockoidc/v1/mockoidcv1connect"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const DefaultTargetVersion = "stable"

var (
	setupOnce     sync.Once
	targetVersion string
	pool          *dockertest.Pool
)

type Scenario struct {
	t              *testing.T
	pool           *dockertest.Pool
	network        *dockertest.Network
	mockoidc       *dockertest.Resource
	ionscale       *dockertest.Resource
	resources      []*dockertest.Resource
	ionscaleClient ionscaleconnect.IonscaleServiceClient
	mockoidcClient mockoidcv1connect.MockOIDCServiceClient
}

func (s *Scenario) CreateTailnet() *api.Tailnet {
	name := petname.Generate(3, "-")
	createTailnetResponse, err := s.ionscaleClient.CreateTailnet(context.Background(), connect.NewRequest(&api.CreateTailnetRequest{Name: name}))
	require.NoError(s.t, err)
	return createTailnetResponse.Msg.GetTailnet()
}

func (s *Scenario) CreateAuthKey(tailnetID uint64, ephemeral bool, tags ...string) string {
	if len(tags) == 0 {
		tags = []string{"tag:test"}
	}
	key, err := s.ionscaleClient.CreateAuthKey(context.Background(), connect.NewRequest(&api.CreateAuthKeyRequest{TailnetId: tailnetID, Ephemeral: ephemeral, Tags: tags, Expiry: durationpb.New(60 * time.Minute)}))
	require.NoError(s.t, err)
	return key.Msg.Value
}

func (s *Scenario) ListMachines(tailnetID uint64) []*api.Machine {
	machines, err := s.ionscaleClient.ListMachines(context.Background(), connect.NewRequest(&api.ListMachinesRequest{TailnetId: tailnetID}))
	require.NoError(s.t, err)
	return machines.Msg.Machines
}

func (s *Scenario) AuthorizeMachines(tailnetID uint64) {
	machines := s.ListMachines(tailnetID)
	for _, m := range machines {
		_, err := s.ionscaleClient.AuthorizeMachine(context.Background(), connect.NewRequest(&api.AuthorizeMachineRequest{MachineId: m.Id}))
		require.NoError(s.t, err)
	}
}

func (s *Scenario) ExpireMachines(tailnetID uint64) {
	machines := s.ListMachines(tailnetID)
	for _, m := range machines {
		_, err := s.ionscaleClient.ExpireMachine(context.Background(), connect.NewRequest(&api.ExpireMachineRequest{MachineId: m.Id}))
		require.NoError(s.t, err)
	}
}

func (s *Scenario) FindMachine(tailnetID uint64, name string) (uint64, error) {
	machines := s.ListMachines(tailnetID)

	for _, m := range machines {
		if m.Name == name {
			return m.Id, nil
		}
	}
	return 0, fmt.Errorf("machine %s not found", name)
}

func (s *Scenario) SetMachineName(machineID uint64, useOSHostname bool, name string) error {
	req := &api.SetMachineNameRequest{MachineId: machineID, UseOsHostname: useOSHostname, Name: name}
	_, err := s.ionscaleClient.SetMachineName(context.Background(), connect.NewRequest(req))
	return err
}

func (s *Scenario) SetACLPolicy(tailnetID uint64, policy *ionscaleclt.ACLPolicy) {
	_, err := s.ionscaleClient.SetACLPolicy(context.Background(), connect.NewRequest(&api.SetACLPolicyRequest{TailnetId: tailnetID, Policy: policy.Marshal()}))
	require.NoError(s.t, err)
}

func (s *Scenario) SetIAMPolicy(tailnetID uint64, policy *ionscaleclt.IAMPolicy) {
	_, err := s.ionscaleClient.SetIAMPolicy(context.Background(), connect.NewRequest(&api.SetIAMPolicyRequest{TailnetId: tailnetID, Policy: policy.Marshal()}))
	require.NoError(s.t, err)
}

func (s *Scenario) EnableMachineAutorization(tailnetID uint64) {
	_, err := s.ionscaleClient.EnableMachineAuthorization(context.Background(), connect.NewRequest(&api.EnableMachineAuthorizationRequest{TailnetId: tailnetID}))
	require.NoError(s.t, err)
}

func (s *Scenario) GetMachineRoutes(machineID uint64) *api.MachineRoutes {
	routes, err := s.ionscaleClient.GetMachineRoutes(context.Background(), connect.NewRequest(&api.GetMachineRoutesRequest{MachineId: machineID}))
	require.NoError(s.t, err)
	return routes.Msg.Routes
}

func (s *Scenario) PushOIDCUser(sub, email, preferredUsername string) {
	_, err := s.mockoidcClient.PushUser(context.Background(), connect.NewRequest(&mockoidcv1.PushUserRequest{Subject: sub, Email: email, PreferredUsername: preferredUsername}))
	require.NoError(s.t, err)
}

func (s *Scenario) printIonscaleLogs() error {
	var stdout bytes.Buffer

	err := s.pool.Client.Logs(docker.LogsOptions{
		Context:      context.TODO(),
		Container:    s.ionscale.Container.ID,
		OutputStream: &stdout,
		ErrorStream:  io.Discard,
		Tail:         "all",
		RawTerminal:  false,
		Stdout:       true,
		Stderr:       true,
		Follow:       false,
		Timestamps:   false,
	})

	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(stdout.Bytes())

	return err
}

type TailscaleNodeConfig struct {
	Hostname string
}

type TailscaleNodeOpt = func(*TailscaleNodeConfig)

func RandomName() string {
	return petname.Generate(3, "-")
}

func WithName(name string) TailscaleNodeOpt {
	return func(config *TailscaleNodeConfig) {
		config.Hostname = name
	}
}

func (s *Scenario) NewTailscaleNode(opts ...TailscaleNodeOpt) *tsn.TailscaleNode {
	config := &TailscaleNodeConfig{Hostname: petname.Generate(3, "-")}
	for _, o := range opts {
		o(config)
	}

	runOpts := &dockertest.RunOptions{
		Repository:   fmt.Sprintf("ts-%s", strings.Replace(targetVersion, ".", "-", -1)),
		Hostname:     config.Hostname,
		Networks:     []*dockertest.Network{s.network},
		ExposedPorts: []string{"1055"},
		Cmd: []string{
			"/app/tailscaled", "--tun", "userspace-networking", "--socks5-server", "0.0.0.0:1055", "--socket", "/tmp/tailscaled.sock",
		},
	}

	resource, err := s.pool.RunWithOptions(
		runOpts,
		restartPolicy,
	)
	require.NoError(s.t, err)

	err = s.pool.Retry(portCheck(resource.GetPort("1055/tcp")))
	require.NoError(s.t, err)

	s.resources = append(s.resources, resource)

	return tsn.New(s.t, config.Hostname, "https://ionscale", resource, s.pool.Retry)
}

func Run(t *testing.T, f func(s *Scenario)) {
	if testing.Short() {
		t.Skip("skipped due to -short flag")
	}

	setupOnce.Do(prepareDockerPoolAndImages)

	if pool == nil {
		t.FailNow()
	}

	var err error
	s := &Scenario{t: t}

	defer func() {
		for _, r := range s.resources {
			_ = pool.Purge(r)
		}

		if verbose() {
			_ = s.printIonscaleLogs()
		}

		if s.ionscale != nil {
			_ = pool.Purge(s.ionscale)
		}

		if s.mockoidc != nil {
			_ = pool.Purge(s.mockoidc)
		}

		if s.network != nil {
			_ = s.network.Close()
		}

		s.resources = nil
		s.network = nil
	}()

	s.pool, err = dockertest.NewPool("")
	require.NoError(t, err)

	s.network, err = pool.CreateNetwork("ionscale-test")
	require.NoError(s.t, err)

	currentPath, err := os.Getwd()
	require.NoError(s.t, err)

	// run mockoidc container
	{
		mockoidcOpts := &dockertest.RunOptions{
			Hostname:     "mockoidc",
			Repository:   "ghcr.io/jsiebens/mockoidc",
			Networks:     []*dockertest.Network{s.network},
			ExposedPorts: []string{"80"},
			Cmd:          []string{"--listen-addr", ":80", "--server-url", "http://mockoidc"},
		}

		s.mockoidc, err = pool.RunWithOptions(mockoidcOpts, restartPolicy)
		require.NoError(s.t, err)

		port := s.mockoidc.GetPort("80/tcp")
		err = pool.Retry(httpCheck(port, "/oidc/.well-known/openid-configuration"))
		require.NoError(s.t, err)

		s.mockoidcClient = mockoidc.NewClient(fmt.Sprintf("http://localhost:%s", port), true)
	}

	ionscale := &dockertest.RunOptions{
		Hostname:   "ionscale",
		Repository: "ionscale-test",
		Mounts: []string{
			fmt.Sprintf("%s/config:/etc/ionscale", currentPath),
		},
		Networks:     []*dockertest.Network{s.network},
		ExposedPorts: []string{"443"},
		Cmd:          []string{"server", "--config", "/etc/ionscale/config.yaml"},
	}

	s.ionscale, err = pool.RunWithOptions(ionscale, restartPolicy)
	require.NoError(s.t, err)

	port := s.ionscale.GetPort("443/tcp")

	addr := fmt.Sprintf("https://localhost:%s", port)
	auth, err := ionscaleclt.LoadClientAuth(addr, "804ecd57365342254ce6647da5c249e85c10a0e51e74856bfdf292a2136b4249")
	require.NoError(s.t, err)

	s.ionscaleClient, err = ionscaleclt.NewClient(auth, addr, true)
	require.NoError(s.t, err)

	err = pool.Retry(func() error {
		_, err := s.ionscaleClient.GetVersion(context.Background(), connect.NewRequest(&api.GetVersionRequest{}))
		return err
	})
	require.NoError(s.t, err)

	f(s)
}

func restartPolicy(config *docker.HostConfig) {
	config.AutoRemove = true
	config.RestartPolicy = docker.RestartPolicy{
		Name: "no",
	}
}

func portCheck(port string) func() error {
	return func() error {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%s", port))
		if err != nil {
			return err
		}
		defer conn.Close()
		return nil
	}
}

func httpCheck(port string, path string) func() error {
	return func() error {
		url := fmt.Sprintf("http://localhost:%s%s", port, path)

		resp, err := http.Get(url)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code not OK")
		}

		return nil
	}
}

func prepareDockerPoolAndImages() {
	targetVersion = os.Getenv("IONSCALE_TESTS_TS_TARGET_VERSION")
	if targetVersion == "" {
		targetVersion = DefaultTargetVersion
	}

	pool, _ = dockertest.NewPool("")

	buildOpts := &dockertest.BuildOptions{
		ContextDir: "../",
		Dockerfile: "tests/docker/tailscale/Dockerfile",
		BuildArgs: []docker.BuildArg{
			{
				Name:  "TAILSCALE_VERSION",
				Value: targetVersion,
			},
		},
	}

	err := pool.Client.BuildImage(docker.BuildImageOptions{
		Name:         fmt.Sprintf("ts-%s", strings.Replace(targetVersion, ".", "-", -1)),
		Dockerfile:   buildOpts.Dockerfile,
		OutputStream: io.Discard,
		ContextDir:   buildOpts.ContextDir,
		BuildArgs:    buildOpts.BuildArgs,
		Platform:     buildOpts.Platform,
	})
	if err != nil {
		log.Fatal(err)
	}

	buildOpts = &dockertest.BuildOptions{
		ContextDir: "../",
		Dockerfile: "tests/docker/ionscale/Dockerfile",
	}

	err = pool.Client.BuildImage(docker.BuildImageOptions{
		Name:         "ionscale-test",
		Dockerfile:   buildOpts.Dockerfile,
		OutputStream: io.Discard,
		ContextDir:   buildOpts.ContextDir,
		BuildArgs:    buildOpts.BuildArgs,
		Platform:     buildOpts.Platform,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func verbose() bool {
	return os.Getenv("IONSCALE_TESTS_VERBOSE") == "true"
}
