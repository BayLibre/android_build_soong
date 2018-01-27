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
	out, manifest, rootPrefix, relativeRoot, cpuProfile, traceFile *string
	directories, emulateJar, writeIfChanged                        *bool
	parallelJobs, compLevel                                        *int

	//out            = flag.String("o", "", "file to write zip file to")
	//manifest       = flag.String("m", "", "input jar manifest file name")
	//directories    = flag.Bool("d", false, "include directories in zip")
	//rootPrefix     = flag.String("P", "", "path prefix within the zip at which to place files")
	//relativeRoot   = flag.String("C", "", "path to use as relative root of files in following -f, -l, or -D arguments")
	//parallelJobs   = flag.Int("j", runtime.NumCPU(), "number of parallel threads to use")
	//compLevel      = flag.Int("L", 5, "deflate compression level (0-9)")
	//emulateJar     = flag.Bool("jar", false, "modify the resultant .zip to emulate the output of 'jar'")
	//writeIfChanged = flag.Bool("write_if_changed", false, "only update resultant .zip if it has changed")

	fArgs            zip.FileArgs
	nonDeflatedFiles = make(uniqueSet)

	//cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
	//traceFile  = flag.String("trace", "", "write trace to file")
)

func init() {
	//flag.Var(&listFiles{}, "l", "file containing list of .class files")
	//flag.Var(&dir{}, "D", "directory to include in zip")
	//flag.Var(&file{}, "f", "file to include in zip")
	//flag.Var(&nonDeflatedFiles, "s", "file path to be stored within the zip without compression")
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: zip -o zipfile [-m manifest] -C dir [-f|-l file]...\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func readRespFile(argFile string) ([]string, error) {
	bytes, err := ioutil.ReadFile(argFile)
	if err != nil {
		return nil, err
	}
	content := []rune(string(bytes))
	// Ninja only dumps single quote for escaping. Enclosing characters in quotes
	// preserves the literal value of all characters within the quotes
	// with the exception of quote itself.
	isQuote := func(r rune) bool {
		switch {
		case r == '\'',
			r == '"':
			return true
		default:
			return false
		}
	}

	isWhitespace := func(r rune) bool {
		switch {
		case r == ' ',
			r == '\t',
			r == '\n',
			r == '\r',
			r == '\f',
			r == '\v':
			return true
		default:
			return false
		}
	}

	var args []string
	var arg []rune
	for i := 0; i < len(content); i++ {
		if len(arg) == 0 {
			for i < len(content) && isWhitespace(content[i]) {
				i++
			}
		}
		if i == len(content) {
			break
		}

		// Backslash escapes the next char.
		if i+1 < len(content) && content[i] == '\\' {
			i++ // Skip the escape.
			arg = append(arg, content[i])
			continue
		}

		// Consume a quoted string.
		if isQuote(content[i]) {
			quote := content[i]
			i++
			for ; i < len(content) && content[i] != quote; i++ {
			}
			if i == len(content) {
				break
			}
			continue
		}

		// End the argument if this is whitespace.
		if isWhitespace(content[i]) {
			if len(arg) != 0 {
				args = append(args, string(arg))
			}
			arg = arg[:0]
			continue
		}
		// This is a normal char.  Append it.
		arg = append(arg, content[i])
	}

	if len(arg) != 0 {
		args = append(args, string(arg))
	}

	return args, nil
}

func main() {
	var expandedArgs []string
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "@") {
			respArgs, err := readRespFile(strings.TrimPrefix(arg, "@"))
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			} else {
				expandedArgs = append(expandedArgs, respArgs...)
			}
		} else {
			expandedArgs = append(expandedArgs, arg)
		}
	}
	fmt.Println(expandedArgs)

	flags := flag.NewFlagSet("flags", flag.ExitOnError)

	out = flags.String("o", "", "file to write zip file to")
	manifest = flags.String("m", "", "input jar manifest file name")
	directories = flags.Bool("d", false, "include directories in zip")
	rootPrefix = flags.String("P", "", "path prefix within the zip at which to place files")
	relativeRoot = flags.String("C", "", "path to use as relative root of files in following -f, -l, or -D arguments")
	parallelJobs = flags.Int("j", runtime.NumCPU(), "number of parallel threads to use")
	compLevel = flags.Int("L", 5, "deflate compression level (0-9)")
	emulateJar = flags.Bool("jar", false, "modify the resultant .zip to emulate the output of 'jar'")
	writeIfChanged = flags.Bool("write_if_changed", false, "only update resultant .zip if it has changed")

	cpuProfile = flags.String("cpuprofile", "", "write cpu profile to file")
	traceFile = flags.String("trace", "", "write trace to file")

	flags.Var(&listFiles{}, "l", "file containing list of .class files")
	flags.Var(&dir{}, "D", "directory to include in zip")
	flags.Var(&file{}, "f", "file to include in zip")
	flags.Var(&nonDeflatedFiles, "s", "file path to be stored within the zip without compression")

	flags.Parse(expandedArgs[1:])

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
