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
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"time"

	"android/soong/finder"

	"github.com/google/blueprint/pathtools"
)

var (
	cpuprofile     string
	filenameToFind string
)

func init() {
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to file")
	flag.StringVar(&filenameToFind, "name", "", "name of file to find")
}

var usage = func() {
	fmt.Printf("usage: finder -name <fileName> <searchDirectory>\n")
	flag.PrintDefaults()
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}
func run() error {
	startDate := time.Now()
	flag.Parse()

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			return fmt.Errorf("error opening cpuprofile: %s", err)
		}
		pprof.StartCPUProfile(f)
		defer f.Close()
		defer pprof.StopCPUProfile()
	}

	writer, err := os.Create("/tmp/finder-log")
	if err != nil {
		return err
	}
	// TODO: replace Lshortfile with Llongfile when bug 63821638 is done
	logger := log.New(writer, "", log.Ldate|log.Lmicroseconds|log.Lshortfile)

	logger.Printf("finder starting at %v\n", startDate)

	rootPaths := flag.Args()
	if len(rootPaths) != 1 {
		usage()
		return fmt.Errorf(
			"The number of root paths provided to search must be equal to 1. Got %v: %q\n",
			len(rootPaths), rootPaths)
	}
	rootPath := rootPaths[0]

	params := finder.CacheParams{
		RootDirs:     []string{rootPath},
		ExcludeDirs:  []string{".git", ".repo"},
		PruneFiles:   []string{".android-out-dir"},
		IncludeFiles: []string{filenameToFind},
	}
	service := finder.New(params, pathtools.OsFs, *logger, "/tmp/finder-db")
	matches := service.FindNamed(rootPath, filenameToFind)
	findDate := time.Now()
	findDuration := findDate.Sub(startDate)
	logger.Printf("ran find in %v\n", findDuration)
	logger.Printf("found %v inodes\n", len(matches))
	sort.Strings(matches)
	for _, match := range matches {
		fmt.Println(match)
	}
	logger.Printf("end of %v inodes\n", len(matches))
	service.Shutdown()

	logger.Printf("finder completed in %v\n", time.Now().Sub(startDate))
	return nil
}
