// Copyright 2018 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

// This file contains the functionality to represent a build event in respect
// to the metric system. A build event corresponds to a block of scoped code
// that contains a begin and a defer end trace. When defined, the duration of
// the scoped code is measure along with other performance measurements such
// as memory.
//
// As explained in the metrics package, the metrics system is a stacked based
// system since the collected metrics is considered to be topline metrics.
// The steps of the build system in the UI layer is sequential. Hence, the
// functionality defined below follows the stack data structure operations.

import (
	"os"
	"syscall"
	"time"

	"android/soong/ui/metrics/metrics_proto"
	"android/soong/ui/tracer"

	"github.com/golang/protobuf/proto"
)

// _now wraps the time.Now() function. _now is declared for unit testing purpose.
var _now = func() time.Time {
	return time.Now()
}

// EventTracer is the interface that provides functionality to trace a
// block of code on time and performance. The End call expects the
// Begin call is invoked, otherwise panic is raised.
type EventTracer interface {
	// Begin starts tracing the event.
	Begin(name, desc string, thread tracer.Thread)

	// End performs post calculations such as duration of the event, aggregates
	// the collected performance information into PerfInfo protobuf message.
	End(thread tracer.Thread) soong_metrics_proto.PerfInfo

	// AddProcResInfo adds information on an executed process such as max resident
	// set memory and the number of voluntary context switches.
	AddProcResInfo(string, *os.ProcessState)
}

// event holds the performance of the metrics data of a single build event.
type event struct {
	// The event name (mostly used for grouping a set of events)
	name string

	// The description of the event (used to uniquely identify an event
	// for metrics analysis).
	desc string

	// The time that the event started to occur.
	start time.Time

	// The list of process resource information that was executed.
	procResInfo []*soong_metrics_proto.ProcessResourceInfo
}

type eventTracerImpl struct {
	activeEvents []event
}

var _ EventTracer = &eventTracerImpl{}

func (t *eventTracerImpl) AddProcResInfo(name string, state *os.ProcessState) {
	if len(t.activeEvents) < 1 {
		return
	}

	rusage := state.SysUsage().(*syscall.Rusage)
	curEvent := &t.activeEvents[len(t.activeEvents)-1]
	curEvent.procResInfo = append(curEvent.procResInfo, &soong_metrics_proto.ProcessResourceInfo{
		Name:             proto.String(name),
		UserTimeMicros:   proto.Uint64(uint64(rusage.Utime.Usec)),
		SystemTimeMicros: proto.Uint64(uint64(rusage.Stime.Usec)),
		MinorPageFaults:  proto.Uint64(uint64(rusage.Minflt)),
		MajorPageFaults:  proto.Uint64(uint64(rusage.Majflt)),
		// ru_inblock and ru_oublock are measured in blocks of 512 bytes.
		IoInputKb:                  proto.Uint64(uint64(rusage.Inblock / 2)),
		IoOutputKb:                 proto.Uint64(uint64(rusage.Oublock / 2)),
		VoluntaryContextSwitches:   proto.Uint64(uint64(rusage.Nvcsw)),
		InvoluntaryContextSwitches: proto.Uint64(uint64(rusage.Nivcsw)),
	})
}

func (t *eventTracerImpl) Begin(name, desc string, _ tracer.Thread) {
	t.activeEvents = append(t.activeEvents,
		event{
			name:  name,
			desc:  desc,
			start: _now(),
		},
	)
}

func (t *eventTracerImpl) End(tracer.Thread) soong_metrics_proto.PerfInfo {
	if len(t.activeEvents) < 1 {
		panic("Internal error: No pending events for endAt to end!")
	}
	lastEvent := t.activeEvents[len(t.activeEvents)-1]
	t.activeEvents = t.activeEvents[:len(t.activeEvents)-1]
	realTime := uint64(_now().Sub(lastEvent.start).Nanoseconds())

	return soong_metrics_proto.PerfInfo{
		Desc:                  proto.String(lastEvent.desc),
		Name:                  proto.String(lastEvent.name),
		StartTime:             proto.Uint64(uint64(lastEvent.start.UnixNano())),
		RealTime:              proto.Uint64(realTime),
		ProcessesResourceInfo: lastEvent.procResInfo,
	}
}
