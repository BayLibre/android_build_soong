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
	"sync"
)

type Action struct {
	Description string
	Outputs     []string
	Command     string
}

type ActionResult struct {
	*Action

	Output string
	Error  error
}

type Counts struct {
	TotalActions    int
	RunningActions  int
	StartedActions  int
	FinishedActions int
}

type ToolStatus interface {
	SetTotalActions(total int)
	StartAction(action *Action)
	FinishAction(result ActionResult)

	// Verbose takes a non-important message that is never printed to the
	// screen, but is in the verbose build log, etc
	Verbose(msg string)
	// Status takes a less important message that may be printed to the
	// screen, but overwritten by another status message. The full message
	// will still appear in the verbose build log.
	Status(msg string)
	// Print takes an message and displays it to the screen and other
	// output logs, etc.
	Print(msg string)
	// Error is similar to Print, but treats it similarly to a failed
	// action, showing it in the error logs, etc.
	Error(msg string)

	Finish()
}

type MsgLevel int

const (
	VerboseLvl MsgLevel = iota
	StatusLvl
	PrintLvl
	ErrorLvl
)

func (l MsgLevel) Prefix() string {
	switch l {
	case VerboseLvl:
		return "verbose: "
	case StatusLvl:
		return "status: "
	case PrintLvl:
		return ""
	case ErrorLvl:
		return "error: "
	default:
		panic("Unknown message level")
	}
}

type StatusOutput interface {
	StartAction(action *Action, counts Counts)
	FinishAction(result ActionResult, counts Counts)
	Message(level MsgLevel, msg string)
	Flush()
}

type Status struct {
	counts  Counts
	outputs []StatusOutput
	lock    sync.Mutex
}

func (s *Status) AddOutput(output StatusOutput) {
	if output == nil {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.outputs = append(s.outputs, output)
}

func (s *Status) StartTool() ToolStatus {
	return &toolStatus{
		status: s,
	}
}

func (s *Status) Finish() {
	for _, o := range s.outputs {
		o.Flush()
	}
}

func (s *Status) updateTotalActions(diff int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.counts.TotalActions += diff
}

func (s *Status) startAction(action *Action) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.counts.RunningActions += 1
	s.counts.StartedActions += 1

	for _, o := range s.outputs {
		o.StartAction(action, s.counts)
	}
}

func (s *Status) finishAction(result ActionResult) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.counts.RunningActions -= 1
	s.counts.FinishedActions += 1

	for _, o := range s.outputs {
		o.FinishAction(result, s.counts)
	}
}

func (s *Status) message(level MsgLevel, msg string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, o := range s.outputs {
		o.Message(level, msg)
	}
}

type toolStatus struct {
	status *Status

	counts Counts
	lock   sync.Mutex
}

var _ ToolStatus = (*toolStatus)(nil)

func (d *toolStatus) SetTotalActions(total int) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if total >= d.counts.StartedActions && total != d.counts.TotalActions {
		d.status.updateTotalActions(total - d.counts.TotalActions)
		d.counts.TotalActions = total
	}
}

func (d *toolStatus) StartAction(action *Action) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.counts.RunningActions += 1
	d.counts.StartedActions += 1

	if d.counts.StartedActions > d.counts.TotalActions {
		d.status.updateTotalActions(d.counts.StartedActions - d.counts.TotalActions)
		d.counts.TotalActions = d.counts.StartedActions
	}

	d.status.startAction(action)
}

func (d *toolStatus) FinishAction(result ActionResult) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.counts.RunningActions -= 1
	d.counts.FinishedActions += 1

	d.status.finishAction(result)
}

func (d *toolStatus) Verbose(msg string) {
	d.status.message(VerboseLvl, msg)
}
func (d *toolStatus) Status(msg string) {
	d.status.message(StatusLvl, msg)
}
func (d *toolStatus) Print(msg string) {
	d.status.message(PrintLvl, msg)
}
func (d *toolStatus) Error(msg string) {
	d.status.message(ErrorLvl, msg)
}

func (d *toolStatus) Finish() {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.counts.TotalActions != d.counts.StartedActions {
		d.status.updateTotalActions(d.counts.StartedActions - d.counts.TotalActions)
	}

	// TODO: update status to correct running/finished edges?
	d.counts.RunningActions = 0
	d.counts.TotalActions = d.counts.StartedActions
}
