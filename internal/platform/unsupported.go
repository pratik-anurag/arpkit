//go:build !linux && !darwin

package platform

import "github.com/pratik-anurag/arpkit/internal/topology"

func collectPlatform(opts Options) (*topology.MachineProfile, error) {
	return nil, ErrUnsupported
}
