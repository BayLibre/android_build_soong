// Copyright 2025 Google Inc. All rights reserved.
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

package incremental_javac_input_lib

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	fid_lib "android/soong/cmd/find_input_delta/find_input_delta_lib"
	dependency_proto "com/android/build/make/tools/dependency_mapper/protolib_go"
	"github.com/google/blueprint/pathtools"
	"google.golang.org/protobuf/proto"
)

var fileSepRegex = regexp.MustCompile("[^[:space:]]+")

type UsageMap struct {
	FilePath string

	Usages []string

	IsDependencyToAll bool

	GeneratedClasses []string
}

func GenerateIncrementalInput(srcs, deps, javacTarget, srcDeps, localHeaderJars string) (err error) {
	incInputPath := javacTarget + ".inc.rsp"
	removedClassesPath := javacTarget + ".rem.rsp"
	inputPcState := javacTarget + ".input.pc_state"
	depsPcState := javacTarget + ".deps.pc_state"
	headersPcState := javacTarget + ".headers.pc_state"

	var classesForRemoval []string
	var incAllSources bool

	// Read the srcRspFile contents
	srcList := readRspFile(srcs)
	// run find_input_delta, save [add + ch] as a []string,  and [del] as another []string
	addF, delF, chF := findInputDelta(srcList, inputPcState, javacTarget, true)
	var incInputList []string
	incInputList = append(incInputList, addF...)
	incInputList = append(incInputList, chF...)

	// check if deps have changed
	depsList := readRspFile(deps)
	if addD, delD, chD := findInputDelta(depsList, depsPcState, javacTarget, false); len(addD)+len(delD)+len(chD) > 0 {
		incAllSources = true
	}

	// If doing fullCompile, we can just return the full list.
	// We can do this earlier as well, but allowing findInputDelta to run outputs
	// the changed sources to build metrics as well as maintains the state of
	// inputs we require if the partialCompile is switched on again.
	if fullCompileEnabled() {
		return writeOutput(incInputPath, removedClassesPath, srcList, classesForRemoval)
	}

	// if javacTarget does not exist, we can include all sources
	if incAllSources || !fileExists(javacTarget) {
		incAllSources = true
	}

	// if incInputList has the same size as srcList, we can just include all
	// sources
	if incAllSources || len(incInputList) == len(srcList) {
		incAllSources = true
	}

	// if dependencyMap does not exist, we can include all sources
	if incAllSources || !fileExists(srcDeps) {
		incAllSources = true
	}

	headersChanged := false
	// if headers do not change, we can just keep the incInputList as is.
	if addH, delH, chH := findInputDelta(strings.Split(localHeaderJars, " "), headersPcState, javacTarget, false); len(addH)+len(delH)+len(chH) > 0 {
		headersChanged = true
	}

	// use revDepsMap to find all usages, add them to output, alongside [add + ch] files
	if headersChanged && fileExists(srcDeps) {
		usageMap, _ := generateUsageMap(srcDeps)
		// if including all sources, no need to check the usageMap
		if !incAllSources {
			incInputList, incAllSources = getUsages(usageMap, append(append(addF, chF...)), delF)
		}
		// use usageMap to add all classes that were generated from removed files.
		classesForRemoval = generateRemovalList(usageMap, delF)
	}

	if incAllSources {
		incInputList = srcList
	}

	// write the output to output path(s)
	return writeOutput(incInputPath, removedClassesPath, incInputList, classesForRemoval)
}

// Checks if Full Compile is enabled or not
func fullCompileEnabled() bool {
	usePartialCompile := os.Getenv("SOONG_USE_PARTIAL_COMPILE")
	if usePartialCompile == "true" || usePartialCompile == "1" || usePartialCompile == "y" || usePartialCompile == "yes" {
		return false
	}
	return true
}

// Returns the list of files that use added, modified or deleted files.
// Returns whether to include all src Files in incremental src set
func getUsages(usageMap map[string]UsageMap, modifiedFiles, deletedFiles []string) ([]string, bool) {
	usagesSet := make(map[string]bool)

	// First add all the modified files in the output
	for _, incInput := range modifiedFiles {
		usagesSet[incInput] = true
	}
	// Add all the usages of modified + deleted files
	for _, modFile := range append(modifiedFiles, deletedFiles...) {
		if um, exists := usageMap[modFile]; exists {
			if um.IsDependencyToAll {
				return []string{}, true
			}
			for _, usage := range um.Usages {
				usagesSet[usage] = true
			}
		}
	}

	var usages []string
	for usage := range usagesSet {
		usages = append(usages, usage)
	}
	return usages, false
}

// Returns the list of class files to be removed, as a result of deleting a source file.
func generateRemovalList(usageMap map[string]UsageMap, delFiles []string) []string {
	var classesForRemoval []string
	for _, delFile := range delFiles {
		if _, exists := usageMap[delFile]; exists {
			classesForRemoval = append(classesForRemoval, usageMap[delFile].GeneratedClasses...)
		}
	}
	return classesForRemoval
}

// Generates the usage map, by reading the supplied dependency map as a proto
// Throws error if the map is unparsable.
func generateUsageMap(srcDeps string) (map[string]UsageMap, error) {
	var message = &dependency_proto.FileDependencyList{}

	usageMapSet := make(map[string]UsageMap)

	data, err := os.ReadFile(srcDeps)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Println("err: ", err)
		panic(err)
	}
	err = proto.Unmarshal(data, message)
	if err != nil {
		fmt.Println("err: ", err)
		panic(err)
	}
	for _, dep := range message.FileDependency {
		addUsageMapIfNotPresent(usageMapSet, *dep.FilePath)
		for _, depV := range dep.FileDependencies {
			addUsageMapIfNotPresent(usageMapSet, depV)
			updatedUsageMap := usageMapSet[depV]
			updatedUsageMap.Usages = append(updatedUsageMap.Usages, *dep.FilePath)
			usageMapSet[depV] = updatedUsageMap
		}
		updatedUsageMap := usageMapSet[*dep.FilePath]
		updatedUsageMap.IsDependencyToAll = *dep.IsDependencyToAll
		updatedUsageMap.GeneratedClasses = dep.GeneratedClasses
		usageMapSet[*dep.FilePath] = updatedUsageMap
	}
	return usageMapSet, nil
}

func addUsageMapIfNotPresent(usageMapSet map[string]UsageMap, key string) {
	if _, exists := usageMapSet[key]; !exists {
		usageMap := UsageMap{
			FilePath:          key,
			Usages:            []string{},
			IsDependencyToAll: false,
			GeneratedClasses:  []string{},
		}
		usageMapSet[key] = usageMap
	}
}

func fileExists(filePath string) bool {
	fileFound := true
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fileFound = false
	}
	return fileFound
}

func readRspFile(rspFile string) (list []string) {
	data, err := os.ReadFile(rspFile)
	if err != nil {
		panic(err)
	}
	list = append(list, fileSepRegex.FindAllString(string(data), -1)...)
	return list
}

// Writes incInput and classesForRemoval to output paths
func writeOutput(incInputPath, removedClassesPath string, incInputList, classesForRemoval []string) (err error) {
	err = pathtools.WriteFileIfChanged(incInputPath, []byte(strings.Join(incInputList, "\n")), 0644)
	if err != nil {
		return err
	}
	return pathtools.WriteFileIfChanged(removedClassesPath, []byte(strings.Join(classesForRemoval, "\n")), 0644)
}

// Runs find_input_delta on the inputs provided, saving the temp state in the
// priorStateFile provided. Also outputs to build metrics if desired.
func findInputDelta(inputs []string, priorStateFile, target string, outputMetrics bool) ([]string, []string, []string) {
	newStateFile := priorStateFile + ".new"

	priorState, err := fid_lib.LoadState(priorStateFile, fid_lib.OsFs)
	if err != nil {
		panic(err)
	}
	// Create the new state
	newState, err := fid_lib.CreateState(inputs, false, fid_lib.OsFs)
	if err != nil {
		panic(err)
	}
	if err = fid_lib.WriteState(newState, newStateFile); err != nil {
		panic(err)
	}

	fileList := fid_lib.CompareInternalState(priorState, newState, target)

	if outputMetrics {
		metrics_dir := os.Getenv("SOONG_METRICS_AGGREGATION_DIR")
		out_dir := os.Getenv("OUT_DIR")
		if metrics_dir != "" {
			if err = fileList.WriteMetrics(metrics_dir, out_dir); err != nil {
				panic(err)
			}
		}
	}

	return flattenChanges(fileList)
}

// Flattens the output of find_input_delta for javac's consumption.
func flattenChanges(root *fid_lib.FileList) ([]string, []string, []string) {
	var allAdditions []string
	var allDeletions []string
	var allChangedFiles []string

	for _, addition := range root.Additions {
		allAdditions = append(allAdditions, addition)
	}

	for _, del := range root.Deletions {
		allDeletions = append(allDeletions, del)
	}

	for _, ch := range root.Changes {
		allChangedFiles = append(allChangedFiles, ch.Name)
	}

	return allAdditions, allDeletions, allChangedFiles
}
