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

package metrics

import (
	"io/ioutil"
	"os"

	"android/soong/ui/metrics/metrics_proto"

	"github.com/golang/protobuf/proto"
)

const (
	RunKati      = "kati"
	RunSoong     = "soong"
	PrimaryNinja = "ninja"
)

type Metrics struct {
	metrics    metrics_proto.MetricsBase
	TimeTracer TimeTracer
}

func New() (metrics *Metrics) {
	m := &Metrics{
		metrics:    metrics_proto.MetricsBase{},
		TimeTracer: &timeTracerImpl{},
	}
	return m
}

func (m *Metrics) SetTimeMetrics(name string, perf metrics_proto.PerfInfo) {
	switch perf.GetName() {
	case RunKati:
		m.metrics.KatiRuns = append(m.metrics.KatiRuns, &perf)
		break
	case RunSoong:
		m.metrics.SoongRuns = append(m.metrics.SoongRuns, &perf)
		break
	case PrimaryNinja:
		m.metrics.NinjaRuns = append(m.metrics.NinjaRuns, &perf)
		break
	default:
		// ignored
	}
}

func (m *Metrics) Serialize() (data []byte, err error) {
	return proto.Marshal(&m.metrics)
}

// exports the output to the file at outputPath
func (m *Metrics) Dump(outputPath string) (err error) {
	data, err := m.Serialize()
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
