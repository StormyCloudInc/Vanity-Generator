package main

import (
	"os"

	"gioui.org/app"
	"gioui.org/unit"

	"github.com/go-i2p/i2p-vanitygen/internal/config"
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
		// Restore the last window size; the OS clamps it to the screen if
		// needed, so small displays keep whatever fits.
		width, height := unit.Dp(520), unit.Dp(820)
		if cfg := config.Load(); cfg.WindowWidth >= 300 && cfg.WindowHeight >= 300 {
			width, height = unit.Dp(cfg.WindowWidth), unit.Dp(cfg.WindowHeight)
		}
		w.Option(app.Size(width, height))
		if err := ui.Run(w); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}
