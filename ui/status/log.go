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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"

	"android/soong/ui/logger"
	"android/soong/ui/status/build_error_proto"
	"android/soong/ui/status/completion_proto"
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

func (v *verboseLog) Write(p []byte) (int, error) {
	fmt.Fprint(v.w, string(p))
	return len(p), nil
}

type errorLog struct {
	w     io.WriteCloser
	empty bool
}

func NewErrorLog(log logger.Logger, filename string) StatusOutput {
	f, err := logger.CreateFileWithRotation(filename, 5)
	if err != nil {
		log.Println("Failed to create error log file:", err)
		return nil
	}

	return &errorLog{
		w:     f,
		empty: true,
	}
}

func (e *errorLog) StartAction(action *Action, counts Counts) {}

func (e *errorLog) FinishAction(result ActionResult, counts Counts) {
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
}

func (e *errorLog) Write(p []byte) (int, error) {
	fmt.Fprint(e.w, string(p))
	return len(p), nil
}

type errorProtoLog struct {
	errorProto soong_build_error_proto.BuildError
	filename   string
	log        logger.Logger
}

func NewProtoErrorLog(log logger.Logger, filename string) StatusOutput {
	return &errorProtoLog{
		errorProto: soong_build_error_proto.BuildError{},
		filename:   filename,
		log:        log,
	}
}

func (e *errorProtoLog) StartAction(action *Action, counts Counts) {}

func (e *errorProtoLog) FinishAction(result ActionResult, counts Counts) {
	if result.Error == nil {
		return
	}

	e.errorProto.ActionErrors = append(e.errorProto.ActionErrors, &soong_build_error_proto.BuildActionError{
		Description: proto.String(result.Description),
		Command:     proto.String(result.Command),
		Output:      proto.String(result.Output),
		Artifacts:   result.Outputs,
		Error:       proto.String(result.Error.Error()),
	})
}

func (e *errorProtoLog) Flush() {
	data, err := proto.Marshal(&e.errorProto)
	if err != nil {
		e.log.Printf("Failed to marshal build status proto: %v\n", err)
		return
	}
	err = ioutil.WriteFile(e.filename, []byte(data), 0644)
	if err != nil {
		e.log.Printf("Failed to write file %s: %v\n", e.filename, err)
	}
}

func (e *errorProtoLog) Message(level MsgLevel, message string) {
	if level > ErrorLvl {
		e.errorProto.ErrorMessages = append(e.errorProto.ErrorMessages, message)
	}
}

func (e *errorProtoLog) Write(p []byte) (int, error) {
	return 0, errors.New("not supported")
}

type buildCompletionLog struct {
	filename string
	log      logger.Logger
}

func NewBuildCompletionLog(log logger.Logger, filename string) StatusOutput {
	return &buildCompletionLog{
		filename: filename,
		log:      log,
	}
}

func (b *buildCompletionLog) StartAction(action *Action, counts Counts) {
	b.updateCounters(counts)
}

func (b *buildCompletionLog) FinishAction(result ActionResult, counts Counts) {
	b.updateCounters(counts)
}

func (b *buildCompletionLog) Flush() {
	//Not required.
}

func (b *buildCompletionLog) Message(level MsgLevel, message string) {
	// Not required.
}

func (b *buildCompletionLog) Write(p []byte) (int, error) {
	return 0, errors.New("not supported")
}

func (b *buildCompletionLog) updateCounters(counts Counts) {
	err := writeToFile(
		&soong_build_completion_status_proto.BuildCompletionStatus{
			CurrentActions:  proto.Uint64(uint64(counts.RunningActions)),
			FinishedActions: proto.Uint64(uint64(counts.FinishedActions)),
			TotalActions:    proto.Uint64(uint64(counts.TotalActions)),
		},
		b.filename,
	)
	if err != nil {
		b.log.Printf("Failed to write file %s: %v\n", b.filename, err)
	}
}

func writeToFile(pb proto.Message, outputPath string) (err error) {
	data, err := proto.Marshal(pb)
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
