package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/pratik-anurag/arpkit/internal/platform"
	"github.com/pratik-anurag/arpkit/internal/topology"
)

func TestRunRedactHostnameJSON(t *testing.T) {
	t.Cleanup(func() {
		platformCollect = platform.Collect
	})
	platformCollect = func(opts platform.Options) (*topology.MachineProfile, error) {
		return &topology.MachineProfile{
			Metadata: topology.Metadata{
				Hostname: "secret-host",
			},
		}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--format=json", "--redact-hostname"}, &stdout, &stderr, BuildInfo{Version: "test"})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0 (stderr=%q)", code, stderr.String())
	}

	var profile topology.MachineProfile
	if err := json.Unmarshal(stdout.Bytes(), &profile); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if profile.Metadata.Hostname != "" {
		t.Fatalf("hostname = %q, want empty", profile.Metadata.Hostname)
	}
}

func TestRunKeepsHostnameByDefault(t *testing.T) {
	t.Cleanup(func() {
		platformCollect = platform.Collect
	})
	platformCollect = func(opts platform.Options) (*topology.MachineProfile, error) {
		return &topology.MachineProfile{
			Metadata: topology.Metadata{
				Hostname: "secret-host",
			},
		}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--format=json"}, &stdout, &stderr, BuildInfo{Version: "test"})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0 (stderr=%q)", code, stderr.String())
	}

	var profile topology.MachineProfile
	if err := json.Unmarshal(stdout.Bytes(), &profile); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if profile.Metadata.Hostname != "secret-host" {
		t.Fatalf("hostname = %q, want secret-host", profile.Metadata.Hostname)
	}
}
