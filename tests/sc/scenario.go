package sc

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	ionscaleclt "github.com/jsiebens/ionscale/pkg/client/ionscale"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	ionscaleconnect "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1/ionscalev1connect"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
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

const DefaultTargetVersion = "1.56.0"

var (
	setupOnce     sync.Once
	targetVersion string
	pool          *dockertest.Pool
)

type Scenario interface {
	NewTailscaleNode(hostname string) TailscaleNode

	ListMachines(tailnetID uint64) []*api.Machine
	CreateAuthKey(tailnetID uint64, ephemeral bool) string
	CreateTailnet(name string) *api.Tailnet
}

type scenario struct {
	t         *testing.T
	pool      *dockertest.Pool
	network   *dockertest.Network
	ionscale  *dockertest.Resource
	resources []*dockertest.Resource
	client    ionscaleconnect.IonscaleServiceClient
}

func (s *scenario) CreateTailnet(name string) *api.Tailnet {
	createTailnetResponse, err := s.client.CreateTailnet(context.Background(), connect.NewRequest(&api.CreateTailnetRequest{Name: name}))
	if err != nil {
		s.t.Fatal(err)
	}
	return createTailnetResponse.Msg.GetTailnet()
}

func (s *scenario) CreateAuthKey(tailnetID uint64, ephemeral bool) string {
	key, err := s.client.CreateAuthKey(context.Background(), connect.NewRequest(&api.CreateAuthKeyRequest{TailnetId: tailnetID, Ephemeral: ephemeral, Tags: []string{"tag:test"}, Expiry: durationpb.New(60 * time.Minute)}))
	if err != nil {
		s.t.Fatal(err)
	}
	return key.Msg.Value
}

func (s *scenario) ListMachines(tailnetID uint64) []*api.Machine {
	machines, err := s.client.ListMachines(context.Background(), connect.NewRequest(&api.ListMachinesRequest{TailnetId: tailnetID}))
	if err != nil {
		s.t.Fatal(err)
	}
	return machines.Msg.Machines
}

func (s *scenario) NewTailscaleNode(hostname string) TailscaleNode {
	tailscaleOptions := &dockertest.RunOptions{
		Repository:   fmt.Sprintf("ts-%s", strings.Replace(targetVersion, ".", "-", -1)),
		Hostname:     hostname,
		Networks:     []*dockertest.Network{s.network},
		ExposedPorts: []string{"1055"},
		Cmd: []string{
			"/app/tailscaled", "--tun", "userspace-networking", "--socks5-server", "0.0.0.0:1055", "--socket", "/tmp/tailscaled.sock",
		},
	}

	resource, err := s.pool.RunWithOptions(
		tailscaleOptions,
		restartPolicy,
	)
	if err != nil {
		s.t.Fatal(err)
	}

	err = s.pool.Retry(portCheck(resource.GetPort("1055/tcp")))
	if err != nil {
		s.t.Fatal(err)
	}

	s.resources = append(s.resources, resource)

	return &tailscaleNode{
		t:           s.t,
		loginServer: "http://ionscale:8080",
		hostname:    hostname,
		resource:    resource,
	}
}

func Run(t *testing.T, f func(s Scenario)) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipped due to -short flag")
	}

	setupOnce.Do(prepareDockerPoolAndImages)

	if pool == nil {
		t.FailNow()
	}

	var err error
	s := &scenario{t: t}

	defer func() {
		for _, r := range s.resources {
			_ = pool.Purge(r)
		}

		if s.ionscale != nil {
			_ = pool.Purge(s.ionscale)
		}

		if s.network != nil {
			_ = s.network.Close()
		}

		s.resources = nil
		s.network = nil
	}()

	if s.pool, err = dockertest.NewPool(""); err != nil {
		t.Fatal(err)
	}

	s.network, err = pool.CreateNetwork("ionscale-test")
	if err != nil {
		t.Fatal(err)
	}

	currentPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	ionscale := &dockertest.RunOptions{
		Hostname:   "ionscale",
		Repository: "ionscale-test",
		Mounts: []string{
			fmt.Sprintf("%s/config:/etc/ionscale", currentPath),
		},
		Networks:     []*dockertest.Network{s.network},
		ExposedPorts: []string{"8080"},
		Cmd:          []string{"server", "--config", "/etc/ionscale/config.yaml"},
	}

	s.ionscale, err = pool.RunWithOptions(ionscale, restartPolicy)
	if err != nil {
		t.Fatal(err)
	}

	port := s.ionscale.GetPort("8080/tcp")

	err = pool.Retry(httpCheck(port, "/key"))
	if err != nil {
		t.Fatal(err)
	}

	auth, err := ionscaleclt.LoadClientAuth("804ecd57365342254ce6647da5c249e85c10a0e51e74856bfdf292a2136b4249")
	if err != nil {
		t.Fatal(err)
	}

	s.client, err = ionscaleclt.NewClient(auth, fmt.Sprintf("http://localhost:%s", port), true)
	if err != nil {
		t.Fatal(err)
	}

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
		ContextDir: "./docker/tailscale",
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
