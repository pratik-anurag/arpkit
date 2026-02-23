package platform

import (
	"errors"

	"github.com/pratik-anurag/arpkit/internal/topology"
)

var ErrUnsupported = errors.New("unsupported platform")

type Options struct {
	Debug bool
	Root  string
}

func Collect(opts Options) (*topology.MachineProfile, error) {
	return collectPlatform(opts)
}
