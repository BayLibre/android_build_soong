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
	rootDir  = flag.String("d", ".", "the value of // for load paths")
	suffix   = flag.String("s", ".star", "suffix for the generated Starlark files")
	dryRun   = flag.Bool("n", false, "dry run")
	recurse  = flag.Bool("r", false, "convert all dependent files")
	mode     = flag.String("mode", "", `"backup" to back up existing files, "write" to overwrite them`)
	warn     = flag.Bool("w", false, "warn about partially failed conversions")
	verbose  = flag.Bool("v", false, "print summary")
	errstat  = flag.Bool("e", false, "print error statistics")
	topLevel = flag.Bool("t", false,
		"emit print statement at the end of explicitly listed makefiles\n"+
			"(only when converting dependents, too)")
)

var backupSuffix string

func main() {
	flag.Parse()
	getConfigVariables()
	getSoongVariables()

	if (*suffix)[0] != '.' {
		*suffix = "." + *suffix
	}
	var errorCounts errorsByType
	mk2star.StarlarkSuffix = *suffix
	if *errstat {
		errorCounts = errorsByType{data: make(map[string]datum)}
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
		if !convertOne(mkFile, *topLevel && *recurse) {
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

type configRegistrar struct {
}

func (_ configRegistrar) NewVariable(name, flavor string) {
	mk2star.KnownVariables.NewVariable(name, "product", flavor)
}

func getConfigVariables() {
	path := filepath.Join(*rootDir, "build", "make", "core", "product.mk")
	if err := mk2star.FindConfigVariables(path, configRegistrar{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type fileNameScope struct{}

func (s fileNameScope) Get(name string) string {
	if name != "BUILD_SYSTEM" {
		return fmt.Sprintf("$(%s)", name)
	}
	return filepath.Join(*rootDir, "build", "make", "core")
}

func (s fileNameScope) Set(_, _ string) {
	panic("implement me")
}

func (s fileNameScope) Call(_ string, _ []string) []string {
	panic("implement me")
}

func (s fileNameScope) SetFunc(_ string, _ func([]string) []string) {
	panic("implement me")
}

type soongRegistrar struct{}

func (_ soongRegistrar) NewVariable(name string, flavor string) {
	mk2star.KnownVariables.NewVariable(name, "soong", flavor)
}

func getSoongVariables() {
	path := filepath.Join(*rootDir, "build", "make", "core", "soong_config.mk")
	err := mk2star.FindSoongVariables(path, fileNameScope{}, soongRegistrar{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

func convertOne(mkFile string, emitPrint bool) bool {
	if v, ok := converted[mkFile]; ok {
		return v != nil
	}
	converted[mkFile] = nil
	ss, err := mk2star.Convert(mkFile, nil, emitPrint)
	if err != nil {
		fmt.Fprintln(os.Stderr, mkFile, ": ", err)
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
			if !convertOne(sub, false) {
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
		fmt.Fprintf(os.Stderr, "Conversion was partially successful for %d files:\n", nPartial)
		for _, f := range sortedFiles {
			if ss := converted[f]; ss != nil && ss.HasErrors() {
				fmt.Fprintln(os.Stderr, "  ", f)
			}
		}
	}

	if *verbose {
		if nFailed > 0 {
			fmt.Fprintf(os.Stderr, "Conversion failed for %d files:\n", nFailed)
			for _, f := range sortedFiles {
				if converted[f] == nil {
					fmt.Fprintln(os.Stderr, "  ", f)
				}
			}
		}
		if nOk > 0 {
			fmt.Fprintf(os.Stderr, "Successfully converted %d files:\n", nOk)
			for _, f := range sortedFiles {
				if ss := converted[f]; ss != nil && !ss.HasErrors() {
					fmt.Fprintln(os.Stderr, "  ", f)
				}
			}
		}
	}
}

type datum struct {
	count int
	arg1  []string
}

type errorsByType struct {
	data map[string]datum
}

func (ebt errorsByType) NewError(message string, node parser.Node, args ...interface{}) {
	v, exists := ebt.data[message]
	if exists {
		v.count++
	} else {
		v = datum{1, nil}
	}
	if strings.Contains(message, "%s") {
		var newArg1 string
		newArg1 = fmt.Sprint(args[0])
		if message == "unsupported line" {
			newArg1 = node.Dump()
		} else if message == "unsupported directive %s" {
			if newArg1 == "include" || newArg1 == "-include" {
				newArg1 = node.Dump()
			}
		}
		v.arg1 = append(v.arg1, newArg1)
	}
	ebt.data[message] = v
}

func (ebt errorsByType) printStatistics() {
	if len(ebt.data) > 0 {
		fmt.Fprintln(os.Stderr, "Error counts:")
	}
	for message, data := range ebt.data {
		if len(data.arg1) == 0 {
			fmt.Fprintf(os.Stderr, "%4d %s\n", data.count, message)
			continue
		}
		itemsByFreq, count := stringsWithFreq(data.arg1)
		fmt.Fprintf(os.Stderr, "%4d %s [%d unique items]:\n", data.count, message, count)
		fmt.Fprintln(os.Stderr, "      ", itemsByFreq)
	}
}

func stringsWithFreq(items []string) (string, int) {
	freq := make(map[string]int)
	for _, item := range items {
		freq[item]++
	}
	var sorted []string
	for item := range freq {
		sorted = append(sorted, item)
	}
	sort.Slice(sorted, func(i int, j int) bool {
		return freq[sorted[i]] > freq[sorted[j]]
	})
	sep := ""
	res := ""
	for _, item := range sorted {
		count := freq[item]
		if count > 1 {
			res += fmt.Sprintf("%s%s(%d)", sep, item, count)
		} else {
			res += fmt.Sprintf("%s%s", sep, item)
		}
		sep = ", "
	}
	return res, len(sorted)
}
