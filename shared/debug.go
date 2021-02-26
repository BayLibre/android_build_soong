package shared

import (
	"os"
	"os/exec"
)

var (
	isDebugging bool
)

// Finds the Delve binary to use. Either uses the SOONG_DELVE_PATH environment
// variable or if that is unset, looks at $PATH.
func ResolveDelveBinary() string {
	result := os.Getenv("SOONG_DELVE_PATH")
	if result == "" {
		result, _ = exec.LookPath("dlv")
	}

	return result
}

// Returns whether the current process is running under Delve due to
// ReexecWithDelveMaybe().
func IsDebugging() bool {
	return isDebugging
}
