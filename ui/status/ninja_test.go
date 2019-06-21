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

package status

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"android/soong/ui/logger"
	"android/soong/ui/status/ninja_frontend"
)

// Tests that closing the ninja reader when nothing has opened the other end of the fifo is fast.
func TestNinjaReader_Close(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "ninja_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	stat := &Status{}
	nr := NewNinjaReader(logger.New(ioutil.Discard), stat.StartTool(), filepath.Join(tempDir, "fifo"))

	start := time.Now()

	nr.Close()

	if g, w := time.Since(start), NINJA_READER_CLOSE_TIMEOUT; g >= w {
		t.Errorf("nr.Close timed out, %s > %s", g, w)
	}
}

type testNinjaReader struct {
	NinjaReader
}

func (t *testNinjaReader) start(id, startTime uint32, outputs, inputs []string) {
	t.handle(&ninja_frontend.Status{
		EdgeStarted: &ninja_frontend.Status_EdgeStarted{
			Id:        &id,
			StartTime: &startTime,
			Desc:      &outputs[0],
			Outputs:   outputs,
			Inputs:    inputs,
		},
	})
}

func (t *testNinjaReader) finish(id, endTime uint32) {
	status := int32(0)
	t.handle(&ninja_frontend.Status{
		EdgeFinished: &ninja_frontend.Status_EdgeFinished{
			Id:      &id,
			EndTime: &endTime,
			Status:  &status,
		},
	})
}

type fakeStatus struct{}

func (fakeStatus) SetTotalActions(total int)        {}
func (fakeStatus) StartAction(action *Action)       {}
func (fakeStatus) FinishAction(result ActionResult) {}
func (fakeStatus) Verbose(msg string)               {}
func (fakeStatus) Status(msg string)                {}
func (fakeStatus) Print(msg string)                 {}
func (fakeStatus) Error(msg string)                 {}
func (fakeStatus) Finish()                          {}

func TestNinjaReader_criticalPath(t *testing.T) {
	tests := []struct {
		name     string
		msgs     func(*testNinjaReader)
		want     []string
		wantTime int
	}{
		{
			name: "empty",
			msgs: func(n *testNinjaReader) {},
		},
		{
			name: "duplicate",
			msgs: func(n *testNinjaReader) {
				n.start(0, 0, []string{"a"}, nil)
				n.start(1, 0, []string{"a"}, nil)
				n.finish(0, 1)
				n.finish(0, 2)
			},
			want:     []string{"a"},
			wantTime: 1,
		},
		{
			name: "linear",
			//  a
			//  |
			//  b
			//  |
			//  c
			msgs: func(n *testNinjaReader) {
				n.start(0, 0, []string{"a"}, nil)
				n.finish(0, 1000)
				n.start(1, 1000, []string{"b"}, []string{"a"})
				n.finish(1, 2000)
				n.start(2, 3000, []string{"c"}, []string{"b"})
				n.finish(2, 4000)
			},
			want:     []string{"c", "b", "a"},
			wantTime: 3000,
		},
		{
			name: "diamond",
			//  a
			//  |\
			//  b c
			//  |/
			//  d
			msgs: func(n *testNinjaReader) {
				n.start(0, 0, []string{"a"}, nil)
				n.finish(0, 1000)
				n.start(1, 1000, []string{"b"}, []string{"a"})
				n.start(2, 1000, []string{"c"}, []string{"a"})
				n.finish(1, 2000)
				n.finish(2, 3000)
				n.start(3, 3000, []string{"d"}, []string{"b", "c"})
				n.finish(3, 4000)
			},
			want:     []string{"d", "c", "a"},
			wantTime: 4000,
		},
		{
			name: "multiple",
			//  a d
			//  | |
			//  b e
			//  |
			//  c
			msgs: func(n *testNinjaReader) {
				n.start(0, 0, []string{"a"}, nil)
				n.start(3, 0, []string{"d"}, nil)
				n.finish(0, 1000)
				n.finish(3, 1000)
				n.start(1, 1000, []string{"b"}, []string{"a"})
				n.start(4, 1000, []string{"e"}, []string{"d"})
				n.finish(1, 2000)
				n.start(2, 2000, []string{"c"}, []string{"b"})
				n.finish(2, 3000)
				n.finish(4, 4000)

			},
			want:     []string{"e", "d"},
			wantTime: 4000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &testNinjaReader{
				NinjaReader: NinjaReader{
					running:           make(map[uint32]startedAction),
					criticalPathNodes: make(map[string]*criticalPathNode),
					status:            fakeStatus{},
				},
			}

			tt.msgs(n)

			criticalPath := n.criticalPath()

			var descs []string
			for _, x := range criticalPath {
				descs = append(descs, x.action.Description)
			}

			if !reflect.DeepEqual(descs, tt.want) {
				t.Errorf("NinjaReader.criticalPath() = %v, want %v", descs, tt.want)
			}

			gotTime := 0
			if len(criticalPath) > 0 {
				gotTime = criticalPath[0].criticalPathMillis
			}
			if gotTime != tt.wantTime {
				t.Errorf("criticalPath[0].criticalPathMillis = %v, want %v", gotTime, tt.wantTime)
			}
		})
	}
}
