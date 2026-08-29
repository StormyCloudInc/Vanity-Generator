//go:build !darwin && !cgo

package gpu

import "fmt"

// Available returns false when GPU support is not compiled in.
func Available() bool { return false }

// ListDevices returns nil when GPU support is not compiled in.
func ListDevices() ([]Device, error) { return nil, nil }

// ProbeMain is a no-op when GPU support is not compiled in.
func ProbeMain(string) int { return 0 }

// NewWorker returns an error when GPU support is not compiled in.
func NewWorker(cfg WorkerConfig) (*Worker, error) {
	return nil, fmt.Errorf("GPU support not available (built without CGo)")
}

// NewTorV3Worker returns an error when GPU support is not compiled in.
func NewTorV3Worker(cfg TorV3WorkerConfig) (*TorV3Worker, error) {
	return nil, fmt.Errorf("GPU support not available (built without CGo)")
}
