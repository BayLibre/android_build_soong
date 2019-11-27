// Copyright 2019 Google Inc. All rights reserved.
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
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"android/soong/ui/build"
	"android/soong/ui/logger"
	"android/soong/ui/metrics"
	soong_metrics_proto "android/soong/ui/metrics/metrics_proto"
	"android/soong/ui/status"
	"android/soong/ui/terminal"
	"android/soong/ui/tracer"

	"github.com/golang/protobuf/proto"
)

type Test struct {
	name string
	args []string

	results *TestResults
}

type TestResults struct {
	metrics *metrics.Metrics
}

func (t *Test) Run(logsSubDir string) {
	output := terminal.NewStatusOutput(os.Stdout, "", false, false)

	log := logger.New(output)
	defer log.Cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trace := tracer.New(log)
	defer trace.Close()

	met := metrics.New()

	stat := &status.Status{}
	defer stat.Finish()
	stat.AddOutput(output)
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
		Writer:  output,
		Status:  stat,
	}}

	config := build.NewConfig(buildCtx, t.args...)
	build.SetupOutDir(buildCtx, config)

	logsDir := filepath.Join(config.OutDir(), "cuj", logsSubDir)

	os.MkdirAll(logsDir, 0777)
	log.SetOutput(filepath.Join(logsDir, "soong.log"))
	trace.SetOutput(filepath.Join(logsDir, "build.trace"))
	stat.AddOutput(status.NewVerboseLog(log, filepath.Join(logsDir, "verbose.log")))
	stat.AddOutput(status.NewErrorLog(log, filepath.Join(logsDir, "error.log")))
	stat.AddOutput(status.NewProtoErrorLog(log, filepath.Join(logsDir, "build_error")))
	stat.AddOutput(status.NewCriticalPath(log))

	defer func() {
		os.MkdirAll(logsDir, 0777)
		met.Dump(filepath.Join(logsDir, "soong_metrics"))
	}()

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

	build.Build(buildCtx, config, build.BuildAll)

	t.results = &TestResults{
		metrics: met,
	}
}

func main() {
	outDir := os.Getenv("OUT_DIR")
	if outDir == "" {
		outDir = "out"
	}

	tests := []Test{
		{
			name: "clean",
			args: []string{"clean"},
		},
		{
			name: "nothing",
			args: []string{"nothing"},
		},
		{
			name: "nothing2",
			args: []string{"nothing"},
		},
		{
			name: "nothing3",
			args: []string{"nothing"},
		},
		{
			name: "framework",
			args: []string{"framework"},
		},
		{
			name: "framework2",
			args: []string{"framework"},
		},
		{
			name: "framework3",
			args: []string{"framework"},
		},
	}

	cujMetrics := metrics.NewCUJMetrics()
	for i, t := range tests {
		logsSubDir := fmt.Sprintf("%02d_%s", i, t.name)
		t.Run(logsSubDir)
		cujMetrics.Add(t.name, t.results.metrics)
	}

	cujMetrics.Dump(filepath.Join(outDir, "cuj", "cuj_metrics.pb"))
}

func writeMetrics(met *soong_metrics_proto.CUJs, outputPath string) error {
	data, err := proto.Marshal(met)
	if err != nil {
		return err
	}
	tempPath := outputPath + ".tmp"
	err = ioutil.WriteFile(tempPath, []byte(data), 0644)
	if err != nil {
		return err
	}
	err = os.Rename(tempPath, outputPath)
	if err != nil {
		return err
	}

	return nil
}
