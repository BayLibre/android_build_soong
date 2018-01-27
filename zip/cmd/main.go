// Copyright 2015 Google Inc. All rights reserved.
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
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"android/soong/zip"
)

type byteReaderCloser struct {
	*bytes.Reader
	io.Closer
}

type pathMapping struct {
	dest, src string
	zipMethod uint16
}

type uniqueSet map[string]bool

func (u *uniqueSet) String() string {
	return `""`
}

func (u *uniqueSet) Set(s string) error {
	if _, found := (*u)[s]; found {
		return fmt.Errorf("File %q was specified twice as a file to not deflate", s)
	} else {
		(*u)[s] = true
	}

	return nil
}

type file struct{}

type listFiles struct{}

type dir struct{}

func (f *file) String() string {
	return `""`
}

func (f *file) Set(s string) error {
	if *relativeRoot == "" {
		return fmt.Errorf("must pass -C before -f")
	}

	fArgs = append(fArgs, zip.FileArg{
		PathPrefixInZip:     filepath.Clean(*rootPrefix),
		SourcePrefixToStrip: filepath.Clean(*relativeRoot),
		SourceFiles:         []string{s},
	})

	return nil
}

func (l *listFiles) String() string {
	return `""`
}

func (l *listFiles) Set(s string) error {
	if *relativeRoot == "" {
		return fmt.Errorf("must pass -C before -l")
	}

	list, err := ioutil.ReadFile(s)
	if err != nil {
		return err
	}

	fArgs = append(fArgs, zip.FileArg{
		PathPrefixInZip:     filepath.Clean(*rootPrefix),
		SourcePrefixToStrip: filepath.Clean(*relativeRoot),
		SourceFiles:         strings.Split(string(list), "\n"),
	})

	return nil
}

func (d *dir) String() string {
	return `""`
}

func (d *dir) Set(s string) error {
	if *relativeRoot == "" {
		return fmt.Errorf("must pass -C before -D")
	}

	fArgs = append(fArgs, zip.FileArg{
		PathPrefixInZip:     filepath.Clean(*rootPrefix),
		SourcePrefixToStrip: filepath.Clean(*relativeRoot),
		GlobDir:             filepath.Clean(s),
	})

	return nil
}

var (
	out            = flag.String("o", "", "file to write zip file to")
	manifest       = flag.String("m", "", "input jar manifest file name")
	directories    = flag.Bool("d", false, "include directories in zip")
	rootPrefix     = flag.String("P", "", "path prefix within the zip at which to place files")
	relativeRoot   = flag.String("C", "", "path to use as relative root of files in following -f, -l, or -D arguments")
	parallelJobs   = flag.Int("j", runtime.NumCPU(), "number of parallel threads to use")
	compLevel      = flag.Int("L", 5, "deflate compression level (0-9)")
	emulateJar     = flag.Bool("jar", false, "modify the resultant .zip to emulate the output of 'jar'")
	writeIfChanged = flag.Bool("write_if_changed", false, "only update resultant .zip if it has changed")

	fArgs            zip.FileArgs
	nonDeflatedFiles = make(uniqueSet)

	cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
	traceFile  = flag.String("trace", "", "write trace to file")
)

func init() {
	flag.Var(&listFiles{}, "l", "file containing list of .class files")
	flag.Var(&dir{}, "D", "directory to include in zip")
	flag.Var(&file{}, "f", "file to include in zip")
	flag.Var(&nonDeflatedFiles, "s", "file path to be stored within the zip without compression")
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: zip -o zipfile [-m manifest] -C dir [-f|-l file]...\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func readArgFileIfExists() error {
	var argFile string
	if !(flag.NArg() > 0 && strings.HasPrefix(flag.Arg(0), "@")) {
		return nil
	} else {
		argFile = strings.TrimPrefix(flag.Arg(0), "@")
	}
	content, err := ioutil.ReadFile(argFile)
	if err != nil {
		return err
	}
	args := strings.Split(strings.TrimSpace(string(content)), " ")

	for {
		if len(args) == 0 {
			return nil
		}
		s := args[0]
		if len(s) < 2 || s[0] != '-' {
			return nil
		}
		numMinuses := 1
		if s[1] == '-' {
			numMinuses++
			if len(s) == 2 { // "--" terminates the flags
				return nil
			}
		}
		name := s[numMinuses:]
		if len(name) == 0 || name[0] == '-' || name[0] == '=' {
			return fmt.Errorf("bad flag syntax: %s", s)
		}

		// it's a flag. does it have an argument?
		args = args[1:]
		hasValue := false
		value := ""
		for i := 1; i < len(name); i++ { // equals cannot be first
			if name[i] == '=' {
				value = name[i+1:]
				hasValue = true
				name = name[0:i]
				break
			}
		}

		f := flag.Lookup(name)
		if f == nil {
			return fmt.Errorf("unrecognized flag: %s", s)
		}

		// This is a hack for how to check if f.Value is bool flag.
		// Since interface "boolFlag" is not exported from flag package, otherwise, we can
		// use f.Value.(boolFlag) to check.
		if f.DefValue == "false" { // special case: doesn't need an arg
			if hasValue {
				if err := f.Value.Set(value); err != nil {
					return fmt.Errorf("invalid boolean value %q for -%s: %v", value, name, err)
				}
			} else {
				if err := f.Value.Set("true"); err != nil {
					return fmt.Errorf("invalid boolean flag %s: %v", name, err)
				}
			}
		} else {
			// It must have a value, which might be the next argument.
			if !hasValue && len(args) > 0 {
				// value is the next arg
				hasValue = true
				value, args = args[0], args[1:]
			}
			if !hasValue {
				return fmt.Errorf("flag needs an argument: -%s", name)
			}
			if err := f.Value.Set(value); err != nil {
				return fmt.Errorf("invalid value %q for flag -%s: %v", value, name, err)
			}
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if err := readArgFileIfExists(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	err := zip.Run(zip.ZipArgs{
		FileArgs:                 fArgs,
		OutputFilePath:           *out,
		CpuProfileFilePath:       *cpuProfile,
		TraceFilePath:            *traceFile,
		EmulateJar:               *emulateJar,
		AddDirectoryEntriesToZip: *directories,
		CompressionLevel:         *compLevel,
		ManifestSourcePath:       *manifest,
		NumParallelJobs:          *parallelJobs,
		NonDeflatedFiles:         nonDeflatedFiles,
		WriteIfChanged:           *writeIfChanged,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
