// Copyright 2017 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"android/soong/ui/build/paths"
)

func main() {
	interposer, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to locate interposer executable:", err)
		os.Exit(1)
	}

	exitCode, err := Main(os.Stdout, os.Stderr, interposer, os.Args, paths.SendLog)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	os.Exit(exitCode)
}

var usage = fmt.Errorf(`To use the PATH interposer:
 * Write the original PATH variable to <interposer>_origpath
 * Set up a directory of symlinks to the PATH interposer, and use that in PATH

If a tool isn't in the allowed list, a log will be posted to the unix domain
socket at <interposer>_log.`)

type logFunc func(logSocket string, entry *paths.LogEntry, done chan interface{})

func Main(stdout, stderr io.Writer, interposer string, args []string, logger logFunc) (int, error) {
	base := filepath.Base(args[0])

	origPathFile := interposer + "_origpath"
	if base == filepath.Base(interposer) {
		return 1, usage
	}

	if allowed, ok := paths.AllowedPathEntries[base]; !ok || !allowed {
		if logger != nil {
			waitForLog := make(chan interface{})
			logger(interposer+"_log", &paths.LogEntry{
				Basename: base,
				Args:     args,
			}, waitForLog)
			defer func() { <-waitForLog }()
		}
		if ok && !allowed {
			return 1, fmt.Errorf("%q is not allowed to be used.", base)
		}
	}

	origPath, err := ioutil.ReadFile(origPathFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, usage
		} else {
			return 1, fmt.Errorf("Failed to read original PATH: %v", err)
		}
	}

	cmd := &exec.Cmd{
		Args: args,
		Env:  os.Environ(),

		Stdin:  os.Stdin,
		Stdout: stdout,
		Stderr: stderr,
	}

	if err := os.Setenv("PATH", string(origPath)); err != nil {
		return 1, fmt.Errorf("Failed to set PATH env: %v", err)
	}
	cmd.Path, err = exec.LookPath(base)
	if err != nil {
		return 1, err
	}

	if err = cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Exited() {
					return status.ExitStatus(), nil
				} else if status.Signaled() {
					exitCode := 128 + int(status.Signal())
					return exitCode, nil
				} else {
					return 1, exitErr
				}
			} else {
				return 1, nil
			}
		}
	}

	return 0, nil
}
