package sc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ory/dockertest/v3"
	"strings"
	"tailscale.com/ipn/ipnstate"
	"testing"
)

type TailscaleNode interface {
	Hostname() string
	Up(authkey string) ipnstate.Status
	IPv4() string
	IPv6() string
	WaitForPeers(expected int)
	Ping(target string)
}

type tailscaleNode struct {
	t           *testing.T
	loginServer string
	hostname    string
	resource    *dockertest.Resource
}

func (t *tailscaleNode) Hostname() string {
	return t.hostname
}

func (t *tailscaleNode) Up(authkey string) ipnstate.Status {
	t.mustExecTailscaleCmd("up", "--login-server", t.loginServer, "--authkey", authkey)
	return t.waitForReady()
}

func (t *tailscaleNode) IPv4() string {
	return t.mustExecTailscaleCmd("ip", "-4")
}

func (t *tailscaleNode) IPv6() string {
	return t.mustExecTailscaleCmd("ip", "-6")
}

func (t *tailscaleNode) waitForReady() ipnstate.Status {
	var status ipnstate.Status
	err := pool.Retry(func() error {
		out, err := t.execTailscaleCmd("status", "--json")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(out), &status); err != nil {
			return err
		}

		if status.CurrentTailnet != nil {
			return nil
		}

		return fmt.Errorf("not connected")
	})
	if err != nil {
		t.t.Fatal(err)
	}
	return status
}

func (t *tailscaleNode) WaitForPeers(expected int) {
	err := pool.Retry(func() error {
		out, err := t.execTailscaleCmd("status", "--json")
		if err != nil {
			return err
		}

		var status ipnstate.Status
		if err := json.Unmarshal([]byte(out), &status); err != nil {
			return err
		}

		if len(status.Peers()) != expected {
			return fmt.Errorf("incorrect peer count")
		}

		return nil
	})
	if err != nil {
		t.t.Fatal(err)
	}
}

func (t *tailscaleNode) Ping(target string) {
	result, err := t.execTailscaleCmd("ping", "--timeout=1s", "--c=10", "--until-direct=true", target)
	if err != nil {
		t.t.Fatal(err)
	}

	if !strings.Contains(result, "pong") && !strings.Contains(result, "is local") {
		t.t.Fatal("ping failed")
	}
}

func (t *tailscaleNode) execTailscaleCmd(cmd ...string) (string, error) {
	i := append([]string{"/app/tailscale", "--socket=/tmp/tailscaled.sock"}, cmd...)
	return execCmd(t.resource, i...)
}

func (t *tailscaleNode) mustExecTailscaleCmd(cmd ...string) string {
	i := append([]string{"/app/tailscale", "--socket=/tmp/tailscaled.sock"}, cmd...)
	s, err := execCmd(t.resource, i...)
	if err != nil {
		t.t.Fatal(err)
	}
	return s
}

func execCmd(resource *dockertest.Resource, cmd ...string) (string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode, err := resource.Exec(cmd, dockertest.ExecOptions{StdOut: &stdout, StdErr: &stderr})
	if err != nil {
		return "", err
	}

	if err != nil {
		return strings.TrimSpace(stdout.String()), err
	}

	if exitCode != 0 {
		return strings.TrimSpace(stdout.String()), fmt.Errorf("command failed with: %s", stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
