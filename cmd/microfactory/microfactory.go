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

// Microfactory is intended to bootstrap a go program from source with minimal
// overhead. The initial run will likely take longer than something like 'go
// run', but it will cache the build results for subsequent runs. It also
// handles multiple packages without a GOPATH (like in an Android tree).
//
// You'll generally wrap this in a script (like soong_ui.bash) that will set up
// the Go environment and find the src and output directories.
//
// When nothing has been run before, the shell script will use 'go run' to
// execute this source file, which will then build and cache itself for future
// use, then restart itself in order to build the main executable.
//
// On future runs, the shell script just calls the cached microfactory version,
// which rebuilds/relaunches itself as necessary, and rebuilds the main
// executable as necessary. Rebuilds are only required if the contents of any
// referenced file changes -- we store and compare the hash of every file.
//
// The main executable is defined by a .fact file that contains lines of the
// form:
//
//   (file|pkg|dep) "value"
//
// `pkg` lines define a new package, any lines up to the next `pkg` line will
// refer to this package.
//
// `dep` lines define a dependency to another package.
//
// `file` lines add a specific file to the current package.
//
// Any `file` lines at the beginning of the file are assigned to the `main`
// package, which implicitly depends on every other package defined.
package main

import (
	"bytes"
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/scanner"
)

var (
	output = flag.String("o", "", "Output file")
	mysrc  = flag.String("s", "", "Microfactory source in case rebuild is necessary")
	mybin  = flag.String("b", "", "Microfactory binary location")

	goToolDir = filepath.Join(runtime.GOROOT(), "pkg", "tool", runtime.GOOS+"_"+runtime.GOARCH)
)

type GoPackage struct {
	Name  string
	Deps  []*GoPackage
	Files []string

	pkgDir     string
	output     string
	hashResult []byte

	mutex    sync.Mutex
	compiled bool
	failed   error
	rebuilt  bool
}

func (p *GoPackage) Compile(srcDir, outDir string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.compiled {
		return p.failed
	}
	p.compiled = true
	hash := sha1.New()
	fmt.Fprintln(hash, runtime.GOOS, runtime.GOARCH, runtime.Version())

	var wg sync.WaitGroup
	for _, dep := range p.Deps {
		wg.Add(1)
		go func(dep *GoPackage) {
			defer wg.Done()
			dep.Compile(srcDir, outDir)
		}(dep)
	}
	wg.Wait()
	for _, dep := range p.Deps {
		if dep.failed != nil {
			p.failed = dep.failed
			return p.failed
		}
		hash.Write(dep.hashResult)
	}

	p.pkgDir = filepath.Join(outDir, p.Name)
	p.output = filepath.Join(p.pkgDir, p.Name) + ".a"
	shaFile := p.output + ".hash"

	cmd := exec.Command(filepath.Join(goToolDir, "compile"), "-o", p.output, "-p", p.Name, "-complete", "-pack", "-trimpath", srcDir)
	for _, dep := range p.Deps {
		cmd.Args = append(cmd.Args, "-I", dep.pkgDir)
	}
	for _, file := range p.Files {
		filename := filepath.Join(srcDir, file)
		cmd.Args = append(cmd.Args, filename)
		f, err := os.Open(filename)
		if err != nil {
			f.Close()
			err = fmt.Errorf("%s: %v", filename, err)
			p.failed = err
			return err
		}
		_, err = io.Copy(hash, f)
		if err != nil {
			f.Close()
			err = fmt.Errorf("%s: %v", filename, err)
			p.failed = err
			return err
		}
		f.Close()
	}
	p.hashResult = hash.Sum(nil)

	var rebuild bool
	if _, err := os.Stat(p.output); err != nil {
		rebuild = true
	}
	if !rebuild {
		if oldSha, err := ioutil.ReadFile(shaFile); err == nil {
			rebuild = !bytes.Equal(oldSha, p.hashResult)
		} else {
			rebuild = true
		}
	}

	if !rebuild {
		return nil
	}

	err := os.RemoveAll(p.pkgDir)
	if err != nil {
		err = fmt.Errorf("%s: %v", p.Name, err)
		p.failed = err
		return err
	}

	err = os.MkdirAll(filepath.Dir(p.output), 0777)
	if err != nil {
		err = fmt.Errorf("%s: %v", p.Name, err)
		p.failed = err
		return err
	}

	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("%s: %v", p.Name, err)
		p.failed = err
		return err
	}

	err = ioutil.WriteFile(shaFile, p.hashResult, 0666)
	if err != nil {
		err = fmt.Errorf("%s: %v", p.Name, err)
		p.failed = err
		return err
	}

	p.rebuilt = true

	return nil
}

func (p *GoPackage) Link(out string) error {
	if p.Name != "main" {
		return fmt.Errorf("Can only link main package")
	}

	shaFile := filepath.Join(filepath.Dir(out), "."+filepath.Base(out)+"_hash")

	if !p.rebuilt {
		if _, err := os.Stat(out); err != nil {
			p.rebuilt = true
		} else if oldSha, err := ioutil.ReadFile(shaFile); err != nil {
			p.rebuilt = true
		} else {
			p.rebuilt = !bytes.Equal(oldSha, p.hashResult)
		}
	}
	if !p.rebuilt {
		return nil
	}

	err := os.Remove(shaFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	err = os.Remove(out)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	cmd := exec.Command(filepath.Join(goToolDir, "link"), "-o", out)
	for _, dep := range p.Deps {
		cmd.Args = append(cmd.Args, "-L", dep.pkgDir)
	}
	cmd.Args = append(cmd.Args, p.output)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(shaFile, p.hashResult, 0666)
}

// checkForRebuild checks to see if microfactory itself needs to be rebuilt.
//
// It will launch a new copy of microfactory if it needed to be rebuilt.
func checkForRebuild() {
	intermediates := filepath.Join(filepath.Dir(*mybin), "."+filepath.Base(*mybin)+"_intermediates")

	err := os.MkdirAll(intermediates, 0777)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create intermediates directory: %q", err)
		os.Exit(1)
	}

	pkg := &GoPackage{
		Name:  "main",
		Files: []string{filepath.Base(*mysrc)},
	}
	err = pkg.Compile(filepath.Dir(*mysrc), intermediates)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = pkg.Link(*mybin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if !pkg.rebuilt {
		return
	}

	cmd := exec.Command(*mybin, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err == nil {
		os.Exit(0)
	} else if e, ok := err.(*exec.ExitError); ok {
		os.Exit(e.ProcessState.Sys().(syscall.WaitStatus).ExitStatus())
	}
	os.Exit(1)
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 || *output == "" {
		fmt.Fprintln(os.Stderr, "Usage: go run microfactory.go -o out/binary <file.fact>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *mybin != "" && *mysrc != "" {
		checkForRebuild()
	}

	mainPackage, err := readFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	srcDir := filepath.Dir(flag.Arg(0))
	intermediates := filepath.Join(filepath.Dir(*output), "."+filepath.Base(*output)+"_intermediates")

	err = os.MkdirAll(intermediates, 0777)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create intermediates directory: %q", err)
		os.Exit(1)
	}

	err = mainPackage.Compile(srcDir, intermediates)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to compile:", err)
		os.Exit(1)
	}

	err = mainPackage.Link(*output)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to link:", err)
		os.Exit(1)
	}
}

// parseError creates an `error` prefixed with the file and line number where it happened.
func parseError(pos scanner.Position, format string, rest ...interface{}) error {
	errorText := fmt.Sprintf(format, rest...)
	return fmt.Errorf("%s: %s", pos.String(), errorText)
}

// readFile reads a *.fact file and returns a *GoPackage of the main package.
func readFile(name string) (*GoPackage, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	packages := make(map[string]*GoPackage)
	curPackage := &GoPackage{
		Name: "main",
	}
	mainPackage := curPackage

	scan := scanner.Scanner{}
	scan.Init(f)
	scan.Position.Filename = name
	scan.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanComments | scanner.SkipComments
	for {
		r := scan.Scan()
		if r == scanner.EOF {
			break
		}
		if r != scanner.Ident {
			return nil, parseError(scan.Pos(), "Expected identifier, found %q", r)
		}
		mode := scan.TokenText()
		modePos := scan.Pos()

		r = scan.Scan()
		if r != scanner.String {
			return nil, parseError(scan.Pos(), "Expected string, found %q", r)
		}
		value, err := strconv.Unquote(scan.TokenText())
		if err != nil {
			return nil, parseError(scan.Pos(), "Failed to parse %q: %s", scan.TokenText(), value)
		}

		switch mode {
		case "package":
			if len(curPackage.Files) == 0 {
				return nil, parseError(modePos, "No files defined for %q", curPackage.Name)
			}
			if _, ok := packages[value]; ok {
				return nil, parseError(scan.Pos(), "Package %q already defined", value)
			}
			curPackage = &GoPackage{
				Name: value,
			}
			packages[value] = curPackage
			mainPackage.Deps = append(mainPackage.Deps, curPackage)
		case "dep":
			if curPackage == mainPackage {
				return nil, parseError(scan.Pos(), "main package automatically depends on all packages")
			}
			if dep, ok := packages[value]; ok {
				curPackage.Deps = append(curPackage.Deps, dep)
			} else {
				return nil, parseError(scan.Pos(), "dependency %q not defined", value)
			}
		case "file":
			if filepath.Clean(value) != value || strings.HasPrefix(value, "../") {
				return nil, parseError(scan.Pos(), "use clean file path, no '..'")
			}
			curPackage.Files = append(curPackage.Files, value)
		default:
			return nil, parseError(scan.Pos(), "Unknown command %q ('package', 'dep', and 'file' are supported)", mode)
		}
	}

	return mainPackage, nil
}
