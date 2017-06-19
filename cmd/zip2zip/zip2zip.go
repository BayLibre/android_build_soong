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
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"android/soong/third_party/zip"
)

var (
	input     = flag.String("i", "", "zip file to read from")
	output    = flag.String("o", "", "output file")
	sortGlobs = flag.Bool("s", false, "sort matches from globs (defaults to the order from the input zip file)")
)

type matchEntry struct {
	file     *zip.File
	fileName string
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: zip2zip -i zipfile -o zipfile [-s] [filespec]...")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "  filespec:")
		fmt.Fprintln(os.Stderr, "    <name>")
		fmt.Fprintln(os.Stderr, "    <in_name>:<out_name>")
		fmt.Fprintln(os.Stderr, "    <glob>:<out_dir>/")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "<glob> uses the rules at https://golang.org/pkg/path/filepath/#Match")
		fmt.Fprintln(os.Stderr, "As a special exception, '**' is supported to specify all files in the input zip")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Files will be copied with their existing compression from the input zipfile to")
		fmt.Fprintln(os.Stderr, "the output zipfile, in the order of filespec arguments")
	}

	flag.Parse()

	if flag.NArg() == 0 || *input == "" || *output == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.SetFlags(log.Lshortfile)

	reader, err := zip.OpenReader(*input)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	output, err := os.Create(*output)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	writer := zip.NewWriter(output)
	defer func() {
		err := writer.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	if err := zip2zip(&reader.Reader, writer, *sortGlobs, flag.Args()); err != nil {
		log.Fatal(err)
	}
}

func zip2zip(reader *zip.Reader, writer *zip.Writer, sortGlobs bool, args []string) error {
	for _, arg := range args {
		var input string
		var output string

		// Reserve escaping for future implementation, so make sure no
		// one is using \ and expecting a certain behavior.
		if strings.Contains(arg, "\\") {
			return fmt.Errorf("\\ characters are not currently supported")
		}

		args := strings.SplitN(arg, ":", 2)
		input = args[0]
		if len(args) == 2 {
			output = args[1]
		}

		matchAll := input == "**"
		if !matchAll && strings.Contains(input, "**") {
			return fmt.Errorf("** is only supported on its own, not with other characters")
		}

		if strings.IndexAny(input, "*?[") >= 0 {
			matches := []matchEntry{}

			for _, file := range reader.File {
				match := matchAll

				if !match {
					var err error
					match, err = filepath.Match(input, file.Name)
					if err != nil {
						return err
					}
				}

				if match {
					var newFileName string
					if output == "" {
						newFileName = file.Name
					} else {
						_, name := filepath.Split(file.Name)
						newFileName = filepath.Join(output, name)
					}
					matches = append(matches, matchEntry{file, newFileName})
				}
			}

			if sortGlobs {
				sort.SliceStable(matches, func(i, j int) bool {
					return matches[i].fileName < matches[j].fileName
				})
			}

			for _, match := range matches {
				if err := writer.CopyFrom(match.file, match.fileName); err != nil {
					return err
				}
			}
		} else {
			if output == "" {
				output = input
			}
			for _, file := range reader.File {
				if input == file.Name {
					err := writer.CopyFrom(file, output)
					if err != nil {
						return err
					}
					break
				}
			}
		}
	}

	return nil
}
