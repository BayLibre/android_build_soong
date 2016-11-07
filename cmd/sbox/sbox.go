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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func main() {
	var exitCode int
	var outFiles []string
	var args []string

	defer func(exitCode *int) {
		os.Exit(*exitCode)
	}(&exitCode)

	for i, arg := range os.Args[2:] {
		if arg == "--" {
			args = os.Args[i+3:]
			break
		} else {
			outFiles = append(outFiles, arg)
		}
	}

	// TODO: Use out/.sandbox as part of TempDir
	dir, err := ioutil.TempDir("", "sbox")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create temp dir:", err)
		exitCode = 1
		return
	}
	defer os.RemoveAll(dir)

	// TODO: Catch signals and cleanup too

	for i, arg := range args {
		os.MkdirAll(filepath.Dir(arg), 0777)
		if strings.Contains(arg, "__SBOX_GEN_PATH__") {
			args[i] = strings.Replace(arg, "__SBOX_GEN_PATH__", dir, -1)
		}
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if exit, ok := err.(*exec.ExitError); ok && !exit.Success() {
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok {
			exitCode = ws.ExitStatus()
			return
		} else {
			fmt.Fprintln(os.Stderr, exit)
			exitCode = 1
			return
		}
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		exitCode = 1
		return
	}

	for _, file := range outFiles {
		err = os.Rename(filepath.Join(dir, file), file)
		if err != nil {
			fmt.Println(os.Stderr, err)
			exitCode = 1
			return
		}
	}
}
