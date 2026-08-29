package gpu

// Device represents a detected GPU compute device.
type Device struct {
	Name          string
	Vendor        string
	MaxWorkGroups int
	Backend       string // "Metal" or "OpenCL"
	Index         int    // backend device index, pass as WorkerConfig.DeviceIndex
}

// ProbeEnv is the environment variable that switches the binary into GPU
// probe mode: the process compiles and runs a tiny kernel on the given device
// index, then exits. Run as a subprocess so that a driver whose compiler
// crashes (e.g. Mesa r600 aborts on global atomics) cannot take down the app.
const ProbeEnv = "VANITYGEN_GPU_PROBE"

// WorkerConfig configures a GPU vanity search worker.
type WorkerConfig struct {
	DeviceIndex  int
	DestTemplate [391]byte // the 391-byte I2P destination to mutate
	Prefix       string    // target base32 prefix
	BatchSize    uint64    // hashes per kernel dispatch (e.g. 1<<22)
}

// BatchResult holds the outcome of one GPU batch.
type BatchResult struct {
	Found        bool
	MatchCounter uint64 // counter value that produced the matching prefix
	Checked      uint64 // number of hashes computed in this batch
}

// Worker represents an active GPU compute session.
// Created by NewWorker, must be closed with Close.
type Worker struct {
	impl workerImpl
}

// RunBatch dispatches one batch of work to the GPU starting at counterStart.
// Blocks until the GPU finishes.
func (w *Worker) RunBatch(counterStart uint64) (BatchResult, error) {
	return w.impl.runBatch(counterStart)
}

// Close releases all GPU resources.
func (w *Worker) Close() {
	w.impl.close()
}

// workerImpl is the platform-specific backend interface.
type workerImpl interface {
	runBatch(counterStart uint64) (BatchResult, error)
	close()
}

// TorV3WorkerConfig configures a GPU worker for Tor v3 vanity search.
type TorV3WorkerConfig struct {
	DeviceIndex int
	Prefix      string
	BatchSize   uint64 // number of pubkeys per GPU dispatch
}

// TorV3Worker represents an active GPU session for Tor v3 vanity checking.
// CPU precomputes Ed25519 public keys, GPU checks SHA3-256 + base32 prefix.
type TorV3Worker struct {
	impl torV3WorkerImpl
}

// RunBatch dispatches a batch of precomputed pubkeys to the GPU for checking.
// pubkeys must contain keyCount*32 bytes. Returns BatchResult where MatchCounter
// is the index of the matching key in the pubkeys array.
func (w *TorV3Worker) RunBatch(pubkeys []byte, keyCount uint64) (BatchResult, error) {
	return w.impl.runBatch(pubkeys, keyCount)
}

// Close releases all GPU resources.
func (w *TorV3Worker) Close() {
	w.impl.close()
}

// torV3WorkerImpl is the platform-specific backend for Tor v3 GPU checking.
type torV3WorkerImpl interface {
	runBatch(pubkeys []byte, keyCount uint64) (BatchResult, error)
	close()
}
