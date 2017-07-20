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
	"android/soong/fs"
	"errors"
	"io"
	"io/ioutil"
)

var (
	cpuprofile     string
	filenameToFind string
	verbose        bool
	dbPath         string
)

func init() {
	flag.StringVar(&cpuprofile, "cpuprofile", "",
		"filepath of profile file to write (optional)")
	flag.StringVar(&filenameToFind, "name", "", "name of file to find")
	flag.BoolVar(&verbose, "v", false, "log addition information")
	flag.StringVar(&dbPath, "db", "", "filepath of cache db")
}

var usage = func() {
	fmt.Printf("usage: finder -name <fileName> --db <dbPath> <searchDirectory> [<searchDirectory>...]\n")
	flag.PrintDefaults()
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
func run() error {
	startTime := time.Now()
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

	var writer io.Writer
	if verbose {
		writer = os.Stderr
	} else {
		writer = ioutil.Discard
	}

	// TODO: replace Lshortfile with Llongfile when bug 63821638 is done
	logger := log.New(writer, "", log.Ldate|log.Lmicroseconds|log.Lshortfile)

	logger.Printf("finder starting at %v\n", startTime)

	rootPaths := flag.Args()
	if len(rootPaths) < 1 {
		usage()
		return fmt.Errorf(
			"must give at least one <searchDirectory>")
	}

	params := finder.CacheParams{
		RootDirs:     rootPaths,
		ExcludeDirs:  []string{".git", ".repo"},
		PruneFiles:   []string{".android-out-dir"},
		IncludeFiles: []string{filenameToFind},
	}
	if dbPath == "" {
		usage()
		return errors.New("param 'db' must be nonempty")
	}
	service := finder.New(params, fs.OsFs, *logger, dbPath)
	defer service.Shutdown()
	matches := service.FindNamed(filenameToFind)
	findDuration := time.Since(startTime)
	logger.Printf("ran find in %v\n", findDuration)
	logger.Printf("found %v inodes\n", len(matches))
	sort.Strings(matches)
	for _, match := range matches {
		fmt.Println(match)
	}
	logger.Printf("end of %v inodes\n", len(matches))

	logger.Printf("finder completed in %v\n", time.Now().Sub(startTime))
	return nil
}
