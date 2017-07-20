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
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"time"

	lib "android/soong/finder"

	"github.com/google/blueprint/pathtools"
)

func main() {
	startDate := time.Now()
	f, err := os.Create("profile")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	writer, err := os.Create("/tmp/finder-log")
	if err != nil {
		panic(err)
	}
	// TODO: replace Lshortfile with Llongfile when bug 63821638 is done
	logger := log.New(writer, "", log.Ldate|log.Lmicroseconds|log.Lshortfile)

	//logger := logger.New(ioutil.Discard).SetVerbose(false).SetOutput("/tmp/finder-log")

	logger.Printf("finder starting at %v\n", startDate)

	rootPath := os.Args[1]
	fileName := os.Args[2]

	// with all filters filtering, it takes 0.24s and the db is 26M
	// with no filters filtering, it takes 0.7s and the db is 86M
	// with just the file filter, it takes 0.33s and the db is 44M
	// with the file filter and the .git/.repo filter, it goes back to 0.24s and 26M

	params := lib.CacheParams{
		RootDirs:     []string{rootPath},
		ExcludeDirs:  []string{".git", ".repo"},
		PruneFiles:   []string{".android-out-dir"},
		IncludeFiles: []string{"Android.mk", "Android.bp", "CleanSpec.mk"}}
	service, err := lib.New(params, pathtools.OsFs, *logger, "/tmp/jeffdb")
	if err != nil {
		panic(err)
	}
	matches := service.FindNamed(rootPath, fileName)
	service.WaitUntilIdle()
	findDate := time.Now()
	findDuration := findDate.Sub(startDate)
	logger.Printf("ran find in %v\n", findDuration)
	logger.Printf("found %v inodes\n", len(matches))
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
	logger.Printf("end of %v inodes\n", len(matches))
	service.Shutdown()

	logger.Printf("finder completed in %v\n", time.Now().Sub(startDate))
}
