// Copyright 2020 Google Inc. All rights reserved.
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

// This tool converts a tab-separated file of license dependencies into a
// text file.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"android/soong/notice/license"
)

const (
	sep = "========================================================================"
)

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `lic-to-txt, a tool to create NOTICE.txt files from .lic.tab files.

Usage: %s [-i <file>] [-o <file>] [--old <target-root>] [--new <target-root>]

  -i <file>
     read input from file, a tab-separated file where the 1st three fields are:
       target, path to license, and external name
  -o <file>
     write output to file
  -old <target-root>
     replace target-root from the start of target names
  -new <target-root>
     replace with target-root at the start of target names
`, os.Args[0])
	}

	var infile string
	var outfile string
	var oldRoot string
	var newRoot string

	flag.StringVar(&infile, "i", "", "input `filename`")
	flag.StringVar(&outfile, "o", "", "output filename")
	flag.StringVar(&oldRoot, "old", "", "old target `root-dir`")
	flag.StringVar(&newRoot, "new", "", "new target `root-dir`")
	flag.Parse()

	m, err := parseInput(infile)
	if err != nil {
		log.Fatalf("Error parsing %q: %v", *infile, err)
	}

	f, err := openOutput(outfile)
	if err != nil {
		log.Fatalf("Error opening output %q: %v", filename, err)
	}
	defer f.Close()

	for _, target := range m.Targets() {
		fmt.Fprintf(f, "%s\n", reRoot(target, oldRoot, newRoot))
		for _, ref := range m.TargetLicenses(target) {
			fmt.Fprintf(f, "  %s (content id: %s)\n", ref.Name, ref.ContentId)
		}
		fmt.Fprintf(f, "\n")
	}

	for _, contentId := range m.Licenses() {
		fmt.Fprintf(f, "%s\nContent id: %s\nName(s):\n", sep, contentId)
		for _, name := range m.LicenseNames(contentId) {
			fmt.Fprintf(f, "  %s\n", name)
		}
		fmt.Fprintf(f, "\nApplies to:\n")
		for _, target := range m.LicenseAppliesTo(contentId) {
			fmt.Fprintf(f, "  %s\n", target)
		}
		c := m.LicenseContent(contentId)
		if strings.HasSuffix(c, "\n") {
			fmt.Fprintf(f, "\n%s\n", c)
		} else {
			fmt.Fprintf(f, "\n%s\n\n", c)
		}
	}
}

func openOutput(filename string) (*os.File, error) {

	if len(filename) == 0 || filename == "-" {
		return os.Stdout, nil
	}
	return os.Create(filename)
}

func parseInput(filename string) (*license.Map, error) {

	var f *os.File
	if len(filename) == 0 || filename == "-" {
		f = os.Stdin
	} else {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatalf("Error opening input %q: %v", filename, err)
		}
		defer f.Close()
	}

	return license.Parse(f)
}

func reRoot(target, oldRoot, newRoot string) string {
	newTarget := target
	if len(oldRoot) > 0 && strings.HasPrefix(target, oldRoot) {
		newTarget = target[len(oldRoot):]
	} else if len(oldRoot) > 0 {
		log.Warnf("target %q not under expected root %q", target, oldRoot)
	}
	if len(newRoot) > 0 {
		newTarget = append(newRoot, newTarget)
	}
	return newTarget
}
