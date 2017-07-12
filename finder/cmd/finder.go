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

package main

import (
	lib "android/soong/finder"
	"fmt"
	"os"
	"sort"
	"time"
)

func main() {
	startDate := time.Now()
	//f, err := os.Create("profile")
	//if err != nil {
	//	panic(err)
	//}
	//pprof.StartCPUProfile(f)
	//defer pprof.StopCPUProfile()
	fmt.Printf("finder starting at %v\n", startDate)

	rootPath := os.Args[1]
	fileName := os.Args[2]

	filterDir := func(items *lib.ChildList) {
		writeIndex := 0
		for _, fileName := range items.Files {
			// include only these files
			if fileName == "Android.mk" || fileName == "Android.bp" || fileName == "CleanSpec.mk" {
				items.Files[writeIndex] = fileName
				writeIndex++
			}
			// ignore output dirs
			if fileName == ".android-out-dir" {
				items.Files = []string{}
				items.SubDirs = []string{}
				return
			}
		}
		// resize
		items.Files = items.Files[:writeIndex]

		writeIndex = 0
		for _, dirName := range items.SubDirs {
			items.SubDirs[writeIndex] = dirName
			// ignore other dirs that are known to not be inputs to the build process
			if dirName == ".git" || dirName == ".repo" {
				// don't include
			} else {
				writeIndex++
			}
		}
		// resize
		items.SubDirs = items.SubDirs[:writeIndex]
	}

	service := lib.New(filterDir, "/tmp/jeffdb")
	matches := service.FindNamed(rootPath, fileName)
	service.WaitUntilIdle()
	findDate := time.Now()
	findDuration := findDate.Sub(startDate)
	fmt.Printf("ran find in %v\n", findDuration)
	fmt.Printf("found %v inodes\n", len(matches))
	summary := []string{}
	maxCount := 2000
	sort.Strings(matches)
	if len(matches) < maxCount {
		summary = matches
	} else {
		summary = append(matches[:maxCount], "more...")
	}
	for _, match := range summary {
		fmt.Println(match)
	}
	fmt.Printf("end of %v inodes\n", len(matches))
	service.Shutdown()

	fmt.Printf("finder completed in %v\n", time.Now().Sub(startDate))
}
