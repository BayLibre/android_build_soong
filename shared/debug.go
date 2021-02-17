package shared

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

var (
	isDebugging bool
)

func ResolveDelveBinary() string {
	result := os.Getenv("SOONG_DELVE_PATH")
	if result == "" {
		result, _ = exec.LookPath("dlv")
	}

	return result
}

func removeVar(varsToRemove []string, v string) bool {
	for _, e := range varsToRemove {
		if e == v {
			return true
		}
	}

	return false
}

func IsDebugging() bool {
	return isDebugging
}

func ReexecWithDelveMaybe(delveListen, delvePath string, varsToRemove []string) {
	//if len(varsToRemove) == 0 {
	//	fmt.Printf("DL=%s, DP=%s\n", delveListen, delvePath)
	//	os.Exit(1)
	//}

	isDebugging = os.Getenv("SOONG_DELVE_REEXECUTED") == "true"
	if isDebugging || delveListen == "" {
		return
	}

	if delvePath == "" {
		fmt.Fprintln(os.Stderr, "Delve debugging requested but failed to find dlv")
		os.Exit(1)
	}

	soongDelveEnv := []string{}
	for _, env := range os.Environ() {
		idx := strings.IndexRune(env, '=')
		if idx != -1 {
			soongDelveEnv = append(soongDelveEnv, env)
		}
	}

	soongDelveEnv = append(soongDelveEnv, "SOONG_DELVE_REEXECUTED=true")

	dlvArgv := []string{
		delvePath,
		"--listen=:" + delveListen,
		"--headless=true",
		"--api-version=2",
		"exec",
		os.Args[0],
		"--",
	}

	dlvArgv = append(dlvArgv, os.Args[1:]...)
	syscall.Exec(delvePath, dlvArgv, soongDelveEnv)
	fmt.Fprintln(os.Stderr, "exec() failed while trying to reexec with Delve")
	os.Exit(1)
}
