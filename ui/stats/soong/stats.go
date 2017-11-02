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

// This package receives and exports summary metrics about the build process:
// for example, timing information and module counts
package stats

import (
	"io/ioutil"
	"os"
	"time"

	"android/soong/ui/stats/soong/gen"

	"github.com/golang/protobuf/proto"
)

const (
	RunKati      = "kati"
	RunSoong     = "soong"
	PrimaryNinja = "ninja"
)

type eventStart struct {
	name    string
	atNanos uint64 // timestamp measured in nanoseconds since the reference date
}

type Collector struct {
	gen.BuildStats

	activeEvents []eventStart
}

func New() (listener *Collector) {
	c := &Collector{
		BuildStats: gen.BuildStats{},
	}
	return c
}

func (c *Collector) now() uint64 {
	return uint64(time.Now().UnixNano())
}

func (c *Collector) BeginEvent(name string) {
	c.BeginEventAt(name, c.now())
}

func (c *Collector) BeginEventAt(name string, atNanos uint64) {
	c.activeEvents = append(c.activeEvents, eventStart{name: name, atNanos: atNanos})
}

func (c *Collector) EndEvent() {
	c.EndEventAt(c.now())
}

func (c *Collector) EndEventAt(atNanos uint64) {
	if len(c.activeEvents) < 1 {
		panic("Internal error: No pending events for EndEventAt to end!")
	}
	lastEvent := c.activeEvents[len(c.activeEvents)-1]
	c.activeEvents = c.activeEvents[:len(c.activeEvents)-1]

	c.RecordEvent(lastEvent.name, atNanos-lastEvent.atNanos)
}

func (c *Collector) RecordEvent(name string, durationNanos uint64) {
	switch name {
	case RunKati:
		c.BuildStats.KatiDuration = durationNanos
		break
	case RunSoong:
		c.BuildStats.SoongDuration = durationNanos
		break
	case PrimaryNinja:
		c.BuildStats.NinjaDuration = durationNanos
		break
	default:
		// ignored
	}
}

func (c *Collector) Serialize() (data []byte, err error) {
	return proto.Marshal(&c.BuildStats)
}

// exports the output to the file at outputPath
func (c *Collector) Dump(outputPath string) (err error) {
	data, err := c.Serialize()
	if err != nil {
		return err
	}
	tempPath := outputPath + ".tmp"
	err = ioutil.WriteFile(tempPath, data, 0777)
	if err != nil {
		return err
	}
	err = os.Rename(tempPath, outputPath)
	if err != nil {
		return err
	}

	return nil
}
