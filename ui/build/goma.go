package build

import (
	"errors"
	"path/filepath"
)

const gomaCtlScript = "goma_ctl.py"

var gomaCtlNotFound = errors.New("goma_ctl.py not found")

func startGoma(ctx Context, config Config) error {
	ctx.BeginTrace("goma_ctl")
	defer ctx.EndTrace()

	var gomaCtl string
	if gomaDir, ok := config.Environment().Get("GOMA_DIR"); ok {
		gomaCtl = filepath.Join(gomaDir, gomaCtlScript)
	} else if home, ok := config.Environment().Get("HOME"); ok {
		gomaCtl = filepath.Join(home, "goma", gomaCtlScript)
	} else {
		return gomaCtlNotFound
	}

	cmd := Command(ctx, config, "goma_ctl.py ensure_start", gomaCtl, "ensure_start")

	if err := cmd.Run(); err != nil {
		ctx.Fatalf("goma_ctl.py ensure_start failed with: %v", err)
	}

	return nil
}
