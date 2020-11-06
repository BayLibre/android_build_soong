// Convert makefile containing device configuration to Starlark file
// The conversion can handle the following constructs in a makefile:
//   * comments
//   * simple variable assignments
//   * $(call init-product,<file>)
// All other constructs are carried over to the output starlark file as comments.
package main

import (
	"android/soong/androidmk/mk2star"
	"android/soong/androidmk/parser"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	rootDir = flag.String("d", ".", "the value of // for load paths")
	suffix  = flag.String("s", ".star", "suffix for the generated Starlark files")
	dryRun  = flag.Bool("n", false, "dry run")
	recurse = flag.Bool("r", false, "convert all dependent files")
	mode    = flag.String("mode", "", `"backup" to back up existing files, "write" to overwrite them`)
	warn    = flag.Bool("w", false, "warn about partially failed conversions")
	verbose = flag.Bool("v", false, "print summary")
	errstat = flag.Bool("e", false, "Print error statistics")
)

var backupSuffix string

func main() {
	flag.Parse()
	productMkPath := filepath.Join(*rootDir, "build", "make", "core", "product.mk")
	if err := mk2star.FindConfigVariables(productMkPath, &mk2star.ConfigVariables); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if (*suffix)[0] != '.' {
		*suffix = "." + *suffix
	}
	var errorCounts errorsByType
	mk2star.StarlarkSuffix = *suffix
	if *errstat {
		errorCounts = errorsByType{count: make(map[string]int)}
		mk2star.ErrorMonitor = errorCounts
	}

	if *mode == "backup" {
		t := time.Now()
		backupSuffix = fmt.Sprintf(".%04d%02d%02d%02d%02d%02d",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second())
	}

	ok := true
	for _, mkFile := range flag.Args() {
		if !convertOne(mkFile) {
			ok = false
		}
	}
	printStats()
	if *errstat {
		errorCounts.printStatistics()
	}
	if ok {
		os.Exit(0)
	}
	os.Exit(1)
}

var converted = make(map[string]*mk2star.StarScript)

//goland:noinspection RegExpRepeatedSpace
var cpNormalizer = regexp.MustCompile(
	"#  Copyright \\(C\\) 20.. The Android Open Source Project")

const cpNormalized = "#  Copyright (C) 20xx The Android Open Source Project"
const copyright = `#
#  Copyright (C) 20xx The Android Open Source Project
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
#
`

func convertOne(mkFile string) bool {
	if v, ok := converted[mkFile]; ok {
		return v != nil
	}
	converted[mkFile] = nil
	ss, err := mk2star.Convert(mkFile, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}
	outputFile := strings.TrimSuffix(mkFile, filepath.Ext(mkFile)) + *suffix

	if *dryRun {
		fmt.Printf("==== %s ====\n", outputFile)
		outText := cpNormalizer.ReplaceAllString(ss.String(), cpNormalized)
		fmt.Println(strings.TrimPrefix(outText, copyright))
	} else {
		if err := maybeBackup(outputFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return false
		}
		if err := ioutil.WriteFile(outputFile, []byte(ss.String()), 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return false
		}
	}
	ok := true
	if *recurse {
		for _, sub := range ss.SubConfigFiles() {
			// File may be absent if it is a conditional load
			if _, err := os.Stat(sub); os.IsNotExist(err) {
				continue
			}
			if !convertOne(sub) {
				ok = false
			}
		}
	}
	converted[mkFile] = ss
	return ok
}

func maybeBackup(filename string) error {
	stat, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil
	}
	if !stat.Mode().IsRegular() {
		return fmt.Errorf("%s exists and is not a regular file", filename)
	}
	switch *mode {
	case "backup":
		return os.Rename(filename, filename+backupSuffix)
	case "write":
		return os.Remove(filename)
	default:
		return fmt.Errorf("%s already exists, use --mode option", filename)
	}
}

func printStats() {
	var sortedFiles []string
	if *warn || *verbose {
		for p := range converted {
			sortedFiles = append(sortedFiles, p)
		}
		sort.Strings(sortedFiles)
	}

	nOk, nPartial, nFailed := 0, 0, 0
	for _, f := range sortedFiles {
		if converted[f] == nil {
			nFailed++
		} else if converted[f].HasErrors() {
			nPartial++
		} else {
			nOk++
		}
	}
	if (*warn || *verbose) && nPartial > 0 {
		fmt.Fprintln(os.Stderr, "Conversion was only partially successful for:")
		for _, f := range sortedFiles {
			if converted[f].HasErrors() {
				fmt.Fprintln(os.Stderr, "  ", f)
			}
		}
	}

	if *verbose {
		if nFailed > 0 {
			fmt.Fprintln(os.Stderr, "Conversion failed for:")
			for _, f := range sortedFiles {
				if converted[f] == nil {
					fmt.Fprintln(os.Stderr, "  ", f)
				}
			}
		}
		if nOk > 0 {
			fmt.Fprintln(os.Stderr, "Successfully converted:")
			for _, f := range sortedFiles {
				if ss := converted[f]; ss != nil && !ss.HasErrors() {
					fmt.Fprintln(os.Stderr, "  ", f)
				}
			}
		}
	}
}

type errorsByType struct {
	count map[string]int
}

func (ebt errorsByType) NewError(message string, _ parser.Node) {
	ebt.count[message] += 1
}

func (ebt errorsByType) printStatistics() {
	if len(ebt.count) > 0 {
		fmt.Fprintln(os.Stderr, "Error counts:")
	}
	for message, count := range ebt.count {
		fmt.Fprintf(os.Stderr, "%4d %s\n", count, message)
	}
}
