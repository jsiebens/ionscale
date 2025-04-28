package tsn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/net/netcheck"
	"testing"
	"time"
)

func New(t *testing.T, name, loginServer string, resource *dockertest.Resource, retry func(func() error) error) *TailscaleNode {
	return &TailscaleNode{
		t:           t,
		loginServer: loginServer,
		hostname:    name,
		resource:    resource,
		retry:       retry,
	}
}

type TailscaleNode struct {
	t           *testing.T
	loginServer string
	hostname    string
	resource    *dockertest.Resource
	retry       func(func() error) error
}

func (t *TailscaleNode) Hostname() string {
	return t.hostname
}

func (t *TailscaleNode) Up(authkey string, flags ...UpFlag) error {
	cmd := []string{"up", "--login-server", t.loginServer, "--authkey", authkey}
	for _, f := range flags {
		cmd = append(cmd, f...)
	}

	t.mustExecTailscaleCmd(cmd...)
	return t.WaitFor(Connected())
}

func (t *TailscaleNode) Set(flags ...UpFlag) string {
	cmd := []string{"set"}
	for _, f := range flags {
		cmd = append(cmd, f...)
	}

	return t.mustExecTailscaleCmd(cmd...)
}

func (t *TailscaleNode) LoginWithOidc(flags ...UpFlag) (int, error) {
	check := func(stdout, stderr string) bool {
		return strings.Contains(stderr, "To authenticate, visit:")
	}

	cmd := []string{"login", "--login-server", t.loginServer}
	for _, f := range flags {
		cmd = append(cmd, f...)
	}

	_, stderr, err := t.execTailscaleCmdWithCheck(check, cmd...)
	require.NoError(t.t, err)

	urlStr := strings.ReplaceAll(stderr, "To authenticate, visit:\n\n\t", "")
	urlStr = strings.TrimSpace(urlStr)

	u, err := url.Parse(urlStr)
	require.NoError(t.t, err)

	q := u.Query()
	q.Set("oidc", "true")
	u.RawQuery = q.Encode()

	responseCode, err := t.curl(u)
	require.NoError(t.t, err)

	if responseCode == http.StatusOK {
		return responseCode, t.WaitFor(Connected())
	}

	return responseCode, nil
}

func (t *TailscaleNode) IPv4() string {
	return t.mustExecTailscaleCmd("ip", "-4")
}

func (t *TailscaleNode) IPv6() string {
	return t.mustExecTailscaleCmd("ip", "-6")
}

func (t *TailscaleNode) WaitFor(c Condition, additional ...Condition) error {
	err := t.retry(func() error {
		out, _, err := t.execTailscaleCmd("status", "--json")
		if err != nil {
			return err
		}

		var status ipnstate.Status
		if err := json.Unmarshal([]byte(out), &status); err != nil {
			return err
		}

		if !c(&status) {
			return fmt.Errorf("condition not met")
		}

		for _, a := range additional {
			if !a(&status) {
				return fmt.Errorf("condition not met")
			}
		}

		return nil
	})
	return err
}

func (t *TailscaleNode) Check(c Condition, additional ...Condition) error {
	out, _, err := t.execTailscaleCmd("status", "--json")
	if err != nil {
		return err
	}

	var status ipnstate.Status
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		return err
	}

	if !c(&status) {
		return fmt.Errorf("condition not met")
	}

	for _, a := range additional {
		if !a(&status) {
			return fmt.Errorf("condition not met")
		}
	}

	return nil
}

func (t *TailscaleNode) Ping(target string) error {
	result, _, err := t.execTailscaleCmd("ping", "--timeout=1s", "--c=10", "--until-direct=true", target)
	if err != nil {
		return err
	}

	if !strings.Contains(result, "pong") && !strings.Contains(result, "is local") {
		return fmt.Errorf("ping failed")
	}

	return nil
}

func (t *TailscaleNode) SetHostname(hostname string) error {
	_, _, err := t.execTailscaleCmd("set", "--hostname", hostname)
	if err != nil {
		return err
	}
	return nil
}

func (t *TailscaleNode) NetCheck() (*netcheck.Report, error) {
	result, _, err := t.execTailscaleCmd("netcheck", "--format=json")
	if err != nil {
		return nil, err
	}

	var nm netcheck.Report
	err = json.Unmarshal([]byte(result), &nm)
	if err != nil {
		return nil, err
	}

	return &nm, err
}

func (t *TailscaleNode) curl(url *url.URL) (int, error) {
	cmd := []string{"curl", "-L", "-s", "-o", "/dev/null", "-w", "%{http_code}", url.String()}
	stdout, _, err := execCmd(t.resource, cmd...)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(stdout)
}

func (t *TailscaleNode) execTailscaleCmd(cmd ...string) (string, string, error) {
	i := append([]string{"/app/tailscale", "--socket=/tmp/tailscaled.sock"}, cmd...)
	return execCmd(t.resource, i...)
}

func (t *TailscaleNode) execTailscaleCmdWithCheck(check func(string, string) bool, cmd ...string) (string, string, error) {
	i := append([]string{"/app/tailscale", "--socket=/tmp/tailscaled.sock"}, cmd...)
	return execCmdWithCheck(t.resource, check, i...)
}

func (t *TailscaleNode) mustExecTailscaleCmd(cmd ...string) string {
	i := append([]string{"/app/tailscale", "--socket=/tmp/tailscaled.sock"}, cmd...)
	s, _, err := execCmd(t.resource, i...)
	require.NoError(t.t, err)
	return s
}

func execCmd(resource *dockertest.Resource, cmd ...string) (string, string, error) {
	return execCmdWithCheck(resource, nil, cmd...)
}

func execCmdWithCheck(resource *dockertest.Resource, check func(string, string) bool, cmd ...string) (string, string, error) {
	tr := strings.TrimSpace
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	type result struct {
		exitCode int
		err      error
	}

	resultChan := make(chan result, 1)
	checkChan := make(chan bool, 1)

	go func() {
		exitCode, err := resource.Exec(cmd, dockertest.ExecOptions{StdOut: &stdout, StdErr: &stderr})
		resultChan <- result{exitCode, err}
	}()

	if check != nil {
		done := make(chan bool)
		ticker := time.NewTicker(2 * time.Second)
		defer func() {
			ticker.Stop()
			done <- true
			close(done)
		}()
		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					if check(tr(stdout.String()), tr(stderr.String())) {
						checkChan <- true
					}
				}
			}
		}()
	}

	select {
	case <-checkChan:
		return tr(stdout.String()), tr(stderr.String()), nil
	case res := <-resultChan:
		if res.err != nil {
			return stdout.String(), stderr.String(), res.err
		}

		if res.exitCode != 0 {
			return tr(stdout.String()), tr(stderr.String()), fmt.Errorf("invalid exit code %d", res.exitCode)
		}

		return tr(stdout.String()), tr(stderr.String()), nil
	case <-time.After(60 * time.Second):
		return tr(stdout.String()), tr(stderr.String()), fmt.Errorf("command timed out")
	}
}
