// The application to convert product configuration makefiles to Starlark.
// Converts either given list of files (and optionally the dependent files
// of the same kind), or all all product configuration makefiles in the
// given source tree.
// Previous version of a converted file can be backed up.
// Optionally prints detailed statistics at the end.
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
	"runtime/debug"
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
	traceVarFlag = flag.String("trace", "", "comma-separated list of variables to trace")
	allInSource  = flag.Bool("all", false, "convert all product config makefiles in the tree under //")
)

var backupSuffix string
var tracedVariables []string
var errorLogger = errorsByType{data: make(map[string]datum)}

var commandLineVariables = []string{"REMOVE_ATB_FROM_BCP"}

func main() {
	flag.Parse()
	getConfigVariables()
	getSoongVariables()

	// TODO(asmundak): setting CLI variables should be part of the config.
	for _, v := range commandLineVariables {
		mk2star.KnownVariables.NewVariable(v, "cli", "string")
	}

	if (*suffix)[0] != '.' {
		*suffix = "." + *suffix
	}

	if *mode == "backup" {
		t := time.Now()
		backupSuffix = fmt.Sprintf(".%04d%02d%02d%02d%02d%02d",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second())
	}
	if *traceVarFlag != "" {
		tracedVariables = strings.Split(*traceVarFlag, ",")
	}

	files := flag.Args()
	if *allInSource {
		if len(files) > 0 {
			fmt.Fprintf(os.Stderr, "file list cannot be specified when -all is present\n")
			os.Exit(1)
		}
		files = findProductMakefiles()
	}
	ok := true
	for _, mkFile := range files {
		if !convertOne(mkFile, *topLevel || *recurse) {
			ok = false
		}
	}
	printStats()
	if *errstat {
		errorLogger.printStatistics()
	}
	if ok {
		os.Exit(0)
	}
	os.Exit(1)
}

func findProductMakefiles() []string {
	const androidProductsMk = "AndroidProducts.mk"
	// Build the list of AndroidProducts.mk files: it's
	// build/make/target/product/AndroidProducts.mk plus
	// device/**/AndroidProducts.mk
	targetAndroidProductsFile := filepath.Join(*rootDir, "build", "make", "target", "product", androidProductsMk)
	if _, err := os.Stat(targetAndroidProductsFile); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n(hint: %s is not a source tree root)\n",
			targetAndroidProductsFile, err, *rootDir)
		return nil
	}
	androidProductsMkFiles := []string{targetAndroidProductsFile}
	err := filepath.Walk(filepath.Join(*rootDir, "device"),
		func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() && filepath.Base(path) == androidProductsMk {
				androidProductsMkFiles = append(androidProductsMkFiles, path)
			}
			return nil
		})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}

	var res []string
	for _, f := range androidProductsMkFiles {
		added, err := mk2star.FindProductMakefiles(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", f, err)
			continue
		}
		res = append(res, added...)
	}
	return res
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

func convertOne(mkFile string, emitPrint bool) (ok bool) {
	if v, ok := converted[mkFile]; ok {
		return v != nil
	}
	converted[mkFile] = nil
	defer func() {
		if r := recover(); r != nil {
			ok = false
			fmt.Fprintf(os.Stderr, "%s: panic while converting: %s\n%s\n", mkFile, r, debug.Stack())
		}
	}()

	mk2starRequest := mk2star.Request{
		MkFile:          mkFile,
		Reader:          nil,
		IsTopLevel:      emitPrint,
		RootDir:         *rootDir,
		OutputSuffix:    *suffix,
		TracedVariables: tracedVariables,
	}
	if *errstat {
		mk2starRequest.ErrorLogger = errorLogger
	}
	ss, err := mk2star.Convert(mk2starRequest)
	if err != nil {
		fmt.Fprintln(os.Stderr, mkFile, ": ", err)
		return false
	}
	script := ss.String()
	outputFile := strings.TrimSuffix(mkFile, filepath.Ext(mkFile)) + *suffix

	if *dryRun {
		fmt.Printf("==== %s ====\n", outputFile)
		outText := cpNormalizer.ReplaceAllString(script, cpNormalized)
		fmt.Println(strings.TrimPrefix(outText, copyright))
	} else {
		if err := maybeBackup(outputFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return false
		}
		if err := ioutil.WriteFile(outputFile, []byte(script), 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return false
		}
	}
	ok = true
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
	if *warn {
		if nPartial > 0 {
			fmt.Fprintf(os.Stderr, "Conversion was partially successful for:\n")
			for _, f := range sortedFiles {
				if ss := converted[f]; ss != nil && ss.HasErrors() {
					fmt.Fprintln(os.Stderr, "  ", f)
				}
			}
		}

		if nFailed > 0 {
			fmt.Fprintf(os.Stderr, "Conversion failed for files:\n")
			for _, f := range sortedFiles {
				if converted[f] == nil {
					fmt.Fprintln(os.Stderr, "  ", f)
				}
			}
		}
	}
	if *verbose {
		fmt.Fprintf(os.Stderr, "%-16s%5d\n", "Succeeded:", nOk)
		fmt.Fprintf(os.Stderr, "%-16s%5d\n", "Partial:", nPartial)
		fmt.Fprintf(os.Stderr, "%-16s%5d\n", "Failed:", nFailed)
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
		itemsByFreq, count := stringsWithFreq(data.arg1, 30)
		fmt.Fprintf(os.Stderr, "%4d %s [%d unique items]:\n", data.count, message, count)
		fmt.Fprintln(os.Stderr, "      ", itemsByFreq)
	}
}

func stringsWithFreq(items []string, topN int) (string, int) {
	freq := make(map[string]int)
	for _, item := range items {
		freq[strings.TrimPrefix(strings.TrimSuffix(item, "]"), "[")]++
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
	for i, item := range sorted {
		if i >= topN {
			res += " ..."
			break
		}
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
