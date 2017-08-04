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
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"android/soong/finder"
	"android/soong/fs"
	"android/soong/ui/build"
	"android/soong/ui/logger"
	"android/soong/ui/tracer"
)

func indexList(s string, list []string) int {
	for i, l := range list {
		if l == s {
			return i
		}
	}

	return -1
}

func inList(s string, list []string) bool {
	return indexList(s, list) != -1
}

func dumpListToFile(list []string, filePath string) {
	desiredText := strings.Join(list, "\n")
	desiredBytes := []byte(desiredText)
	actualBytes, err := ioutil.ReadFile(filePath)
	if err != nil || !bytes.Equal(desiredBytes, actualBytes) {
		ioutil.WriteFile(filePath, desiredBytes, 0777)
	}
}

func runFind(ctx build.Context, config build.Config) *finder.Finder {
	startTime := uint64(time.Now().UnixNano())
	dir, err := os.Getwd()
	if err != nil {
		ctx.Fatal(err.Error())
	}
	cacheParams := finder.CacheParams{
		WorkingDirectory: dir,
		RootDirs:         []string{"."},
		ExcludeDirs:      []string{".git", ".repo"},
		PruneFiles:       []string{".android-out-dir", ".find-ignore"},
		IncludeFiles:     []string{"Android.mk", "Android.bp", "CleanSpec.mk"},
	}
	f := finder.New(cacheParams, fs.OsFs, logger.New(ioutil.Discard), filepath.Join(config.SoongOutDir(), "files.db"))
	dumpDir := filepath.Join(config.FileListDir())
	os.MkdirAll(dumpDir, 0777)

	androidMks := f.FindFirstNamedAt(".", "Android.mk")
	dumpListToFile(androidMks, dumpDir+"/Android.mk.list")

	androidBps := f.FindNamedAt(".", "Android.bp")
	dumpListToFile(androidBps, dumpDir+"/Android.bp.list")

	cleanSpecs := f.FindFirstNamedAt(".", "CleanSpec.mk")
	dumpListToFile(cleanSpecs, dumpDir+"/CleanSpec.mk.list")

	ctx.CompleteTrace("find modules", startTime, uint64(time.Now().UnixNano()))

	return f
}

func main() {
	logWriter := os.Stderr
	log := logger.New(logWriter)
	defer log.Cleanup()

	if len(os.Args) < 2 || !inList("--make-mode", os.Args) {
		log.Fatalln("The `soong` native UI is not yet available.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trace := tracer.New(log)
	defer trace.Close()

	build.SetupSignals(log, cancel, func() {
		trace.Close()
		log.Cleanup()
	})

	buildCtx := build.Context{&build.ContextImpl{
		Context:        ctx,
		Logger:         log,
		Tracer:         trace,
		StdioInterface: build.StdioImpl{},
	}}
	config := build.NewConfig(buildCtx, os.Args[1:]...)

	log.SetVerbose(config.IsVerbose())
	build.SetupOutDir(buildCtx, config)

	if config.Dist() {
		logsDir := filepath.Join(config.DistDir(), "logs")
		os.MkdirAll(logsDir, 0777)
		log.SetOutput(filepath.Join(logsDir, "soong.log"))
		trace.SetOutput(filepath.Join(logsDir, "build.trace"))
	} else {
		log.SetOutput(filepath.Join(config.OutDir(), "soong.log"))
		trace.SetOutput(filepath.Join(config.OutDir(), "build.trace"))
	}

	if start, ok := os.LookupEnv("TRACE_BEGIN_SOONG"); ok {
		if !strings.HasSuffix(start, "N") {
			if start_time, err := strconv.ParseUint(start, 10, 64); err == nil {
				log.Verbosef("Took %dms to start up.",
					time.Since(time.Unix(0, int64(start_time))).Nanoseconds()/time.Millisecond.Nanoseconds())
				buildCtx.CompleteTrace("startup", start_time, uint64(time.Now().UnixNano()))
			}
		}

		if executable, err := os.Executable(); err == nil {
			trace.ImportMicrofactoryLog(filepath.Join(filepath.Dir(executable), "."+filepath.Base(executable)+".trace"))
		}
	}
	f := runFind(buildCtx, config)

	// tell the finder to shut down, but don't wait for it to finish before starting the build
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		f.Shutdown()
		wg.Done()
	}()

	build.Build(buildCtx, config, build.BuildAll)

	// give the Finder a chance to finish saving its database before exiting
	wg.Wait()
}
