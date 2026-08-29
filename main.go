package main

import (
	"os"

	"gioui.org/app"

	"github.com/go-i2p/i2p-vanitygen/internal/gpu"
	"github.com/go-i2p/i2p-vanitygen/internal/ui"
	"github.com/go-i2p/i2p-vanitygen/internal/updater"
)

func main() {
	// GPU probe subprocess mode: compile/run a test kernel on one device and
	// exit. Isolates driver compiler crashes from the main app.
	if idx, ok := os.LookupEnv(gpu.ProbeEnv); ok {
		os.Exit(gpu.ProbeMain(idx))
	}

	updater.Cleanup()

	go func() {
		w := new(app.Window)
		w.Option(app.Title("Vanity Domain Generator"))
		w.Option(app.Size(520, 820))
		if err := ui.Run(w); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}
