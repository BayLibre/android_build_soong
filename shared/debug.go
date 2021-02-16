package shared

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

func ReexecWithDelveMaybe() {
	var listenVarName string
	binary := path.Base(os.Args[0])

	if binary == "soong_ui" {
		listenVarName = "SOONG_UI_DELVE"
	} else if binary == "soong_build" {
		listenVarName = "SOONG_DELVE"
	}

	if listenVarName == "" {
		return
	}

	soongDelveListen := os.Getenv(listenVarName)
	if soongDelveListen == "" {
		return
	}

	soongDelvePath := os.Getenv("SOONG_DELVE_PATH")

	if soongDelvePath == "" {
		soongDelvePath, _ = exec.LookPath("dlv")
	}

	soongDelveEnv := []string{}
	for _, env := range os.Environ() {
		idx := strings.IndexRune(env, '=')
		if idx != -1 {
			if env[:idx] != listenVarName && env[:idx] != "SOONG_DELVE_PATH" {
				soongDelveEnv = append(soongDelveEnv, env)
			}
		}
	}

	if soongDelvePath == "" {
		fmt.Fprintln(os.Stderr, "SOONG_DELVE is set but failed to find dlv")
		os.Exit(1)
	}
	dlvArgv := []string{
		soongDelvePath,
		"--listen=:" + soongDelveListen,
		"--headless=true",
		"--api-version=2",
		"exec",
		os.Args[0],
		"--",
	}
	dlvArgv = append(dlvArgv, os.Args[1:]...)
	syscall.Exec(soongDelvePath, dlvArgv, soongDelveEnv)
	fmt.Fprintln(os.Stderr, "exec() failed while trying to reexec with Delve")
	os.Exit(1)
}
