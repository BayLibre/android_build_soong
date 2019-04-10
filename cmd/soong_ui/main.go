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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"android/soong/cmd/soong_ui/command"
	"android/soong/ui/build"
	"android/soong/ui/logger"
	"android/soong/ui/metrics"
	"android/soong/ui/status"
	"android/soong/ui/terminal"
	"android/soong/ui/tracer"
)

// Main execution of soong_ui. The command format is as follows:
//
//    soong_ui [<opt 1> <opt 2> ... <opt n>] <command> [arg 1> <arg 2> ... <arg n>]
//
// The options are global flags. Command is the type of soong_ui execution.
// Only one type of execution is specified. The args are specific to the
// command.
func main() {
	// determine which type of build execution soong is going to do.
	c, err := command.New(os.Args[1:])
	if err != nil {
		// TODO (patricearruda) Replace string with err when the command format is fully implemented.
		fmt.Printf("The `soong` native UI is not yet available.\n")
		os.Exit(1)
	}

	writer := terminal.NewWriter(c.Stdio())
	defer writer.Finish()

	log := logger.New(writer)
	defer log.Cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trace := tracer.New(log)
	defer trace.Close()

	met := metrics.New()

	stat := &status.Status{}
	defer stat.Finish()
	stat.AddOutput(terminal.NewStatusOutput(writer, os.Getenv("NINJA_STATUS"),
		build.OsEnvironment().IsEnvTrue("ANDROID_QUIET_BUILD")))
	stat.AddOutput(trace.StatusTracer())

	build.SetupSignals(log, cancel, func() {
		trace.Close()
		log.Cleanup()
		stat.Finish()
	})

	buildCtx := build.Context{ContextImpl: &build.ContextImpl{
		Context: ctx,
		Logger:  log,
		Metrics: met,
		Tracer:  trace,
		Writer:  writer,
		Status:  stat,
	}}

	config := c.Config(buildCtx)

	build.SetupOutDir(buildCtx, config)

	logsDir := config.OutDir()
	if config.Dist() {
		logsDir = filepath.Join(config.DistDir(), "logs")
	}

	os.MkdirAll(logsDir, 0777)
	log.SetOutput(filepath.Join(logsDir, "soong.log"))
	trace.SetOutput(filepath.Join(logsDir, "build.trace"))
	stat.AddOutput(status.NewVerboseLog(log, filepath.Join(logsDir, "verbose.log")))
	stat.AddOutput(status.NewErrorLog(log, filepath.Join(logsDir, "error.log")))

	defer met.Dump(filepath.Join(logsDir, "build_metrics"))

	if start, ok := os.LookupEnv("TRACE_BEGIN_SOONG"); ok {
		if !strings.HasSuffix(start, "N") {
			if start_time, err := strconv.ParseUint(start, 10, 64); err == nil {
				log.Verbosef("Took %dms to start up.",
					time.Since(time.Unix(0, int64(start_time))).Nanoseconds()/time.Millisecond.Nanoseconds())
				buildCtx.CompleteTrace(metrics.RunSetupTool, "startup", start_time, uint64(time.Now().UnixNano()))
			}
		}

		if executable, err := os.Executable(); err == nil {
			trace.ImportMicrofactoryLog(filepath.Join(filepath.Dir(executable), "."+filepath.Base(executable)+".trace"))
		}
	}

	f := build.NewSourceFinder(buildCtx, config)
	defer f.Shutdown()
	build.FindSources(buildCtx, config, f)

	c.Execute(buildCtx, config, logsDir)
}
