// Copyright 2020 Google Inc. All Rights Reserved.
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

import (
	"testing"
	"time"
)

func TestPanicOnMultipleSetBaseTime(t *testing.T) {
	met := New()
	met.SetBaseTime(time.Now())

	defer func() { recover() }()

	met.SetBaseTime(time.Now())

	t.Errorf("expected panic")
}

func TestLogBuildStepEnforcesName(t *testing.T) {
	met := New()
	met.SetBaseTime(time.Now())

	defer func() { recover() }()

	met.LogBuildStep("this_is_a_name_that_wont_ever_be_used")

	t.Errorf("expected panic")
}

func assertTimeRange(t *testing.T, name string, val *uint64, base time.Time, lowerBound time.Time, upperBound time.Time) {
	if val == nil {
		t.Fatalf("%s not set", name)
	}
	adjusted := base.Add(time.Duration(*val))
	if !lowerBound.Before(adjusted) {
		t.Fatalf("%s %s is lower than range (%s, %s)", name, adjusted, lowerBound, upperBound)
	}
	if !upperBound.After(adjusted) {
		t.Fatalf("%s %s is higher than range (%s, %s)", name, adjusted, lowerBound, upperBound)
	}
}

func TestLogBuildStep(t *testing.T) {

	met := New()
	base := time.Now()
	met.SetBaseTime(base)

	met.LogBuildStep("StartingMainNinjaNs")
	time2 := time.Now()
	met.LogBuildStep("StartingMainNinjaNs") // deliberately the same
	time3 := time.Now()
	met.LogBuildStep("FinishedNs")
	time4 := time.Now()

	if met.metrics.BuildSteps == nil {
		t.Fatalf("build steps is nil")
	}

	assertTimeRange(t, "StartingMainNinjaNs", met.metrics.BuildSteps.StartingMainNinjaNs, base, time2, time3)
	assertTimeRange(t, "FinishedNs", met.metrics.BuildSteps.FinishedNs, base, time3, time4)
}
