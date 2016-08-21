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

package tracer

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"android/soong/ui/logger"
)

type Thread uint64

const (
	MainThread     = Thread(iota)
	MaxInitThreads = Thread(iota)
)

type Tracer interface {
	Begin(name string, thread Thread)
	End(name string, thread Thread)
	Complete(name string, thread Thread, begin, end uint64)

	ImportNinjaLog(thread Thread, filename string, startOffset time.Time)
}

type tracerImpl struct {
	lock sync.Mutex
	log  logger.Logger

	buf  bytes.Buffer
	file *os.File
	w    io.WriteCloser

	firstEvent bool
	nextTid    uint64
}

var _ Tracer = &tracerImpl{}

type viewerEvent struct {
	Name  string      `json:"name,omitempty"`
	Phase string      `json:"ph"`
	Scope string      `json:"s,omitempty"`
	Time  uint64      `json:"ts"`
	Dur   uint64      `json:"dur,omitempty"`
	Pid   uint64      `json:"pid"`
	Tid   uint64      `json:"tid"`
	ID    uint64      `json:"id,omitempty"`
	Arg   interface{} `json:"args,omitempty"`
}

type nameArg struct {
	Name string `json:"name"`
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func New(log logger.Logger) *tracerImpl {
	ret := &tracerImpl{
		log: log,

		firstEvent: true,
		nextTid:    uint64(MaxInitThreads),
	}

	ret.w = nopCloser{&ret.buf}
	fmt.Fprintln(ret.w, "[")

	ret.defineThread(MainThread, "main")

	return ret
}

func (t *tracerImpl) SetOutput(filename string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	// chrome://tracing requires that compressed trace files end in .gz
	if !strings.HasSuffix(filename, ".gz") {
		filename += ".gz"
	}

	f, err := logger.CreateFileWithRotation(filename, 5)
	if err != nil {
		t.log.Println("Failed to create trace file:", err)
		return
	}
	t.file = f
	t.w = gzip.NewWriter(f)

	if _, err := io.Copy(t.w, &t.buf); err != nil {
		t.log.Println("Failed to write trace buffer to file:", err)
	}
	t.buf = bytes.Buffer{}
}

func (t *tracerImpl) Close() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.file != nil {
		fmt.Fprintln(t.w, "]")

		if err := t.w.Close(); err != nil {
			t.log.Println("Error closing trace writer:", err)
		}

		if err := t.file.Close(); err != nil {
			t.log.Println("Error closing trace file:", err)
		}
		t.file = nil
		t.w = nopCloser{&t.buf}
	}
}

func (t *tracerImpl) writeEvent(event *viewerEvent) {
	t.lock.Lock()
	defer t.lock.Unlock()

	bytes, err := json.Marshal(event)
	if err != nil {
		t.log.Println("Failed to marshal event:", err)
		t.log.Verbosef("Event: %#v", event)
		return
	}

	if !t.firstEvent {
		fmt.Fprintln(t.w, ",")
	} else {
		t.firstEvent = false
	}

	if _, err = t.w.Write(bytes); err != nil {
		t.log.Println("Trace write error:", err)
	}
}

func (t *tracerImpl) NewThread(name string) Thread {
	t.lock.Lock()
	ret := Thread(t.nextTid)
	t.nextTid += 1
	t.lock.Unlock()

	t.defineThread(ret, name)
	return ret
}

func (t *tracerImpl) defineThread(thread Thread, name string) {
	t.writeEvent(&viewerEvent{
		Name:  "thread_name",
		Phase: "M",
		Pid:   0,
		Tid:   uint64(thread),
		Arg: &nameArg{
			Name: name,
		},
	})
}

func (t *tracerImpl) Begin(name string, thread Thread) {
	t.writeEvent(&viewerEvent{
		Name:  name,
		Phase: "B",
		Time:  uint64(time.Now().UnixNano()) / 1000,
		Pid:   0,
		Tid:   uint64(thread),
	})
}

func (t *tracerImpl) End(name string, thread Thread) {
	t.writeEvent(&viewerEvent{
		Name:  name,
		Phase: "E",
		Time:  uint64(time.Now().UnixNano()) / 1000,
		Pid:   0,
		Tid:   uint64(thread),
	})
}

func (t *tracerImpl) Complete(name string, thread Thread, begin, end uint64) {
	t.writeEvent(&viewerEvent{
		Name:  name,
		Phase: "X",
		Time:  begin / 1000,
		Dur:   (end - begin) / 1000,
		Pid:   0,
		Tid:   uint64(thread),
	})
}
