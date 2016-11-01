// Copyright 2016 Google Inc. All rights reserved.
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
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func filter(pipe io.Reader, output io.Writer, pathToRemove string) error {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "1 warning generated." ||
			line == "Use -header-filter=.* to display errors from all non-system headers. Use -system-headers to display errors from system headers as well." ||
			strings.HasSuffix(line, " warnings generated.") {

			continue
		}
		if pathToRemove != "" && strings.HasPrefix(line, pathToRemove) {
			line = line[len(pathToRemove):]
		}
		fmt.Fprintln(output, line)
	}

	return scanner.Err()
}

func main() {
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "creating pipe:", err)
		os.Exit(1)
	}
	cmd.Stdout = cmd.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "start:", err)
		os.Exit(1)
	}

	absPath, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error calculating current directory:", err)
		os.Exit(1)
	}

	if err := filter(stderr, os.Stdout, absPath+"/"); err != nil {
		fmt.Fprintln(os.Stderr, "reading output:", err)
		defer os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			if w, ok := err.ProcessState.Sys().(syscall.WaitStatus); ok {
				os.Exit(w.ExitStatus())
			}
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
