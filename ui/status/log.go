// Copyright 2018 Google Inc. All rights reserved.
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

package status

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"android/soong/ui/logger"
	"android/soong/ui/status/build_status_proto"

	"github.com/golang/protobuf/proto"
)

type verboseLog struct {
	w io.WriteCloser
}

func NewVerboseLog(log logger.Logger, filename string) StatusOutput {
	if !strings.HasSuffix(filename, ".gz") {
		filename += ".gz"
	}

	f, err := logger.CreateFileWithRotation(filename, 5)
	if err != nil {
		log.Println("Failed to create verbose log file:", err)
		return nil
	}

	w := gzip.NewWriter(f)

	return &verboseLog{
		w: w,
	}
}

func (v *verboseLog) StartAction(action *Action, counts Counts) {}

func (v *verboseLog) FinishAction(result ActionResult, counts Counts) {
	cmd := result.Command
	if cmd == "" {
		cmd = result.Description
	}

	fmt.Fprintf(v.w, "[%d/%d] %s\n", counts.FinishedActions, counts.TotalActions, cmd)

	if result.Error != nil {
		fmt.Fprintf(v.w, "FAILED: %s\n", strings.Join(result.Outputs, " "))
	}

	if result.Output != "" {
		fmt.Fprintln(v.w, result.Output)
	}
}

func (v *verboseLog) Flush() {
	v.w.Close()
}

func (v *verboseLog) Message(level MsgLevel, message string) {
	fmt.Fprintf(v.w, "%s%s\n", level.Prefix(), message)
}

type buildStatus struct {
	status build_status_proto.BuildStatus

	filename string
}

type errorLog struct {
	w              io.WriteCloser
	buildRunStatus buildStatus
	log            logger.Logger
	empty          bool
}

func NewErrorLog(log logger.Logger, filename string, protoFilename string) StatusOutput {
	f, err := logger.CreateFileWithRotation(filename, 5)
	if err != nil {
		log.Println("Failed to create error log file:", err)
		return nil
	}

	return &errorLog{
		w:     f,
		log:   log,
		empty: true,
		buildRunStatus: buildStatus{
			filename: protoFilename,
		},
	}
}

func (e *errorLog) StartAction(action *Action, counts Counts) {}

func (e *errorLog) FinishAction(result ActionResult, counts Counts) {
	e.buildRunStatus.add(result)
	if result.Error == nil {
		return
	}

	if !e.empty {
		fmt.Fprintf(e.w, "\n\n")
	}
	e.empty = false

	fmt.Fprintf(e.w, "FAILED: %s\n", result.Description)

	if len(result.Outputs) > 0 {
		fmt.Fprintf(e.w, "Outputs: %s\n", strings.Join(result.Outputs, " "))
	}

	fmt.Fprintf(e.w, "Error: %s\n", result.Error)
	if result.Command != "" {
		fmt.Fprintf(e.w, "Command: %s\n", result.Command)
	}
	fmt.Fprintf(e.w, "Output:\n%s\n", result.Output)
}

func (e *errorLog) Flush() {
	e.buildRunStatus.setBuildResult()
	e.buildRunStatus.dump(e.log)
	e.w.Close()
}

func (e *errorLog) Message(level MsgLevel, message string) {
	if level < ErrorLvl {
		return
	}

	if !e.empty {
		fmt.Fprintf(e.w, "\n\n")
	}
	e.empty = false

	fmt.Fprintf(e.w, "error: %s\n", message)
	e.buildRunStatus.status.ErrorMessages = append(e.buildRunStatus.status.ErrorMessages, message)
}

func (b *buildStatus) add(result ActionResult) {
	if result.Error != nil {
		output := result.Output
		if len(result.Outputs) > 0 {
			output += " " + strings.Join(result.Outputs, " ")
		}
		b.status.ActionErrors = append(b.status.ActionErrors, &build_status_proto.BuildActionError{
			Error:       proto.String(result.Error.Error()),
			Description: proto.String(result.Description),
			Command:     proto.String(result.Command),
			Output:      proto.String(output),
		})
	}
}

func (b *buildStatus) setBuildResult() {
	if len(b.status.ActionErrors) == 0 {
		b.status.Result = build_status_proto.BuildStatus_PASSED.Enum()
	} else {
		b.status.Result = build_status_proto.BuildStatus_FAILED.Enum()
	}
}

func (b *buildStatus) dump(log logger.Logger) {
	data, err := proto.Marshal(&b.status)
	if err != nil {
		log.Println("Failed to marshal build status proto: %v", err)
		return
	}
	err = ioutil.WriteFile(b.filename, []byte(data), 0644)
	if err != nil {
		log.Println("Failed to write file %s: %v", b.filename, err)
	}
}
