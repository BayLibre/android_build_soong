package metrics

import (
	"time"

	"android/soong/ui/metrics/metrics_proto"
	"android/soong/ui/tracer"
)

type timeEvent struct {
	desc string
	name string

	atNanos uint64 // timestamp measured in nanoseconds since the reference date
}

type TimeTracer interface {
	Begin(name string, thread tracer.Thread)
	End(thread tracer.Thread)
}

type timeTracerImpl struct {
	activeEvents []timeEvent
}

var _ TimeTracer = &timeTracerImpl{}

func (t *timeTracerImpl) now() uint64 {
	return uint64(time.Now().UnixNano())
}

func (t *timeTracerImpl) Begin(name string, thread tracer.Thread) {
	t.beginAt(name, t.now())
}

func (t *timeTracerImpl) beginAt(name string, atNanos uint64) {
	t.activeEvents = append(t.activeEvents, timeEvent{name: name, atNanos: atNanos})
}

func (t *timeTracerImpl) End(thread tracer.Thread) {
	t.endAt(t.now())
}

func (t *timeTracerImpl) endAt(atNanos uint64) metrics_proto.PerfInfo {
	if len(t.activeEvents) < 1 {
		panic("Internal error: No pending events for endAt to end!")
	}
	lastEvent := t.activeEvents[len(t.activeEvents)-1]
	t.activeEvents = t.activeEvents[:len(t.activeEvents)-1]
	realTime := atNanos - lastEvent.atNanos

	return metrics_proto.PerfInfo{
		Desc:      &lastEvent.desc,
		Name:      &lastEvent.name,
		StartTime: &lastEvent.atNanos,
		RealTime:  &realTime}
}
