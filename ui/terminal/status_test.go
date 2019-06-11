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

package terminal

import (
	"bytes"
	"fmt"
	"testing"

	"android/soong/ui/status"
)

func TestStatusOutput(t *testing.T) {
	tests := []struct {
		name  string
		calls func(stat status.StatusOutput)
		smart string
		dumb  string
	}{
		{
			name:  "two actions",
			calls: twoActions,
			smart: "\r\x1b[1m[  0% 0/2] action1\x1b[0m\x1b[K\r\x1b[1m[ 50% 1/2] action1\x1b[0m\x1b[K\r\x1b[1m[ 50% 1/2] action2\x1b[0m\x1b[K\r\x1b[1m[100% 2/2] action2\x1b[0m\x1b[K\n",
			dumb:  "[ 50% 1/2] action1\n[100% 2/2] action2\n",
		},
		{
			name:  "two parallel actions",
			calls: twoParallelActions,
			smart: "\r\x1b[1m[  0% 0/2] action1\x1b[0m\x1b[K\r\x1b[1m[  0% 0/2] action2\x1b[0m\x1b[K\r\x1b[1m[ 50% 1/2] action1\x1b[0m\x1b[K\r\x1b[1m[100% 2/2] action2\x1b[0m\x1b[K\n",
			dumb:  "[ 50% 1/2] action1\n[100% 2/2] action2\n",
		},
		{
			name:  "action with output",
			calls: actionsWithOutput,
			smart: "\r\x1b[1m[  0% 0/3] action1\x1b[0m\x1b[K\r\x1b[1m[ 33% 1/3] action1\x1b[0m\x1b[K\r\x1b[1m[ 33% 1/3] action2\x1b[0m\x1b[K\r\x1b[1m[ 66% 2/3] action2\x1b[0m\x1b[K\noutput1\noutput2\n\r\x1b[1m[ 66% 2/3] action3\x1b[0m\x1b[K\r\x1b[1m[100% 3/3] action3\x1b[0m\x1b[K\n",
			dumb:  "[ 33% 1/3] action1\n[ 66% 2/3] action2\noutput1\noutput2\n[100% 3/3] action3\n",
		},
		{
			name:  "action with output without newline",
			calls: actionsWithOutputWithoutNewline,
			smart: "\r\x1b[1m[  0% 0/3] action1\x1b[0m\x1b[K\r\x1b[1m[ 33% 1/3] action1\x1b[0m\x1b[K\r\x1b[1m[ 33% 1/3] action2\x1b[0m\x1b[K\r\x1b[1m[ 66% 2/3] action2\x1b[0m\x1b[K\noutput1\noutput2\n\r\x1b[1m[ 66% 2/3] action3\x1b[0m\x1b[K\r\x1b[1m[100% 3/3] action3\x1b[0m\x1b[K\n",
			dumb:  "[ 33% 1/3] action1\n[ 66% 2/3] action2\noutput1\noutput2\n[100% 3/3] action3\n",
		},
		{
			name:  "action with error",
			calls: actionsWithError,
			smart: "\r\x1b[1m[  0% 0/3] action1\x1b[0m\x1b[K\r\x1b[1m[ 33% 1/3] action1\x1b[0m\x1b[K\r\x1b[1m[ 33% 1/3] action2\x1b[0m\x1b[K\r\x1b[1m[ 66% 2/3] action2\x1b[0m\x1b[K\nFAILED: f1 f2\ntouch f1 f2\nerror1\nerror2\n\r\x1b[1m[ 66% 2/3] action3\x1b[0m\x1b[K\r\x1b[1m[100% 3/3] action3\x1b[0m\x1b[K\n",
			dumb:  "[ 33% 1/3] action1\n[ 66% 2/3] action2\nFAILED: f1 f2\ntouch f1 f2\nerror1\nerror2\n[100% 3/3] action3\n",
		},
		{
			name:  "action with empty description",
			calls: actionWithEmptyDescription,
			smart: "\r\x1b[1m[  0% 0/1] command1\x1b[0m\x1b[K\r\x1b[1m[100% 1/1] command1\x1b[0m\x1b[K\n",
			dumb:  "[100% 1/1] command1\n",
		},
		{
			name:  "messages",
			calls: actionsWithMessages,
			smart: "\r\x1b[1m[  0% 0/2] action1\x1b[0m\x1b[K\r\x1b[1m[ 50% 1/2] action1\x1b[0m\x1b[K\r\x1b[1mstatus\x1b[0m\x1b[K\r\x1b[Kprint\nFAILED: error\n\r\x1b[1m[ 50% 1/2] action2\x1b[0m\x1b[K\r\x1b[1m[100% 2/2] action2\x1b[0m\x1b[K\n",
			dumb:  "[ 50% 1/2] action1\nstatus\nprint\nFAILED: error\n[100% 2/2] action2\n",
		},
		{
			name:  "action with long description",
			calls: actionWithLongDescription,
			smart: "\r\x1b[1m[  0% 0/2] action with very long descrip\x1b[0m\x1b[K\r\x1b[1m[ 50% 1/2] action with very long descrip\x1b[0m\x1b[K\n",
			dumb:  "[ 50% 1/2] action with very long description to test eliding\n",
		},
		{
			name:  "action with output with ansi codes",
			calls: actionWithOuptutWithAnsiCodes,
			smart: "\r\x1b[1m[  0% 0/1] action1\x1b[0m\x1b[K\r\x1b[1m[100% 1/1] action1\x1b[0m\x1b[K\n\x1b[31mcolor\x1b[0m\n",
			dumb:  "[100% 1/1] action1\ncolor\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("smart", func(t *testing.T) {
				smart := &fakeSmartTerminal{termWidth: 40}
				stat := NewStatusOutput(smart, "", false)
				tt.calls(stat)
				stat.Flush()

				if g, w := smart.String(), tt.smart; g != w {
					t.Errorf("want:\n%q\ngot:\n%q", w, g)
				}
			})

			t.Run("dumb", func(t *testing.T) {
				dumb := &bytes.Buffer{}
				stat := NewStatusOutput(dumb, "", false)
				tt.calls(stat)
				stat.Flush()

				if g, w := dumb.String(), tt.dumb; g != w {
					t.Errorf("want:\n%q\ngot:\n%q", w, g)
				}
			})
		})
	}
}

func twoActions(stat status.StatusOutput) {
	action1 := &status.Action{Description: "action1"}
	result1 := status.ActionResult{Action: action1}
	action2 := &status.Action{Description: "action2"}
	result2 := status.ActionResult{Action: action2}

	counts := status.Counts{
		TotalActions: 2,
	}

	counts.StartedActions++
	counts.RunningActions++
	stat.StartAction(action1, counts)

	counts.FinishedActions++
	stat.FinishAction(result1, counts)

	counts.StartedActions++
	stat.StartAction(action2, counts)

	counts.FinishedActions++
	counts.RunningActions--
	stat.FinishAction(result2, counts)
}

func twoParallelActions(stat status.StatusOutput) {
	action1 := &status.Action{Description: "action1"}
	result1 := status.ActionResult{Action: action1}
	action2 := &status.Action{Description: "action2"}
	result2 := status.ActionResult{Action: action2}

	counts := status.Counts{
		TotalActions: 2,
	}

	counts.StartedActions++
	counts.RunningActions++
	stat.StartAction(action1, counts)

	counts.StartedActions++
	counts.RunningActions++
	stat.StartAction(action2, counts)

	counts.FinishedActions++
	counts.RunningActions--
	stat.FinishAction(result1, counts)

	counts.FinishedActions++
	counts.RunningActions--
	stat.FinishAction(result2, counts)
}

func actionsWithOutput(stat status.StatusOutput) {
	action1 := &status.Action{Description: "action1"}
	result1 := status.ActionResult{Action: action1}
	action2 := &status.Action{Description: "action2"}
	result2 := status.ActionResult{Action: action2, Output: "output1\noutput2\n"}
	action3 := &status.Action{Description: "action3"}
	result3 := status.ActionResult{Action: action3}

	counts := status.Counts{
		TotalActions: 3,
	}

	counts.StartedActions++
	counts.RunningActions++
	stat.StartAction(action1, counts)

	counts.FinishedActions++
	stat.FinishAction(result1, counts)

	counts.StartedActions++
	stat.StartAction(action2, counts)

	counts.FinishedActions++
	stat.FinishAction(result2, counts)

	counts.StartedActions++
	stat.StartAction(action3, counts)

	counts.FinishedActions++
	counts.RunningActions--
	stat.FinishAction(result3, counts)
}

func actionsWithOutputWithoutNewline(stat status.StatusOutput) {
	action1 := &status.Action{Description: "action1"}
	result1 := status.ActionResult{Action: action1}
	action2 := &status.Action{Description: "action2"}
	result2 := status.ActionResult{Action: action2, Output: "output1\noutput2"}
	action3 := &status.Action{Description: "action3"}
	result3 := status.ActionResult{Action: action3}

	counts := status.Counts{
		TotalActions: 3,
	}

	counts.StartedActions++
	counts.RunningActions++
	stat.StartAction(action1, counts)

	counts.FinishedActions++
	stat.FinishAction(result1, counts)

	counts.StartedActions++
	stat.StartAction(action2, counts)

	counts.FinishedActions++
	stat.FinishAction(result2, counts)

	counts.StartedActions++
	stat.StartAction(action3, counts)

	counts.FinishedActions++
	counts.RunningActions--
	stat.FinishAction(result3, counts)
}

func actionsWithError(stat status.StatusOutput) {
	action1 := &status.Action{Description: "action1"}
	result1 := status.ActionResult{Action: action1}
	action2 := &status.Action{Description: "action2", Outputs: []string{"f1", "f2"}, Command: "touch f1 f2"}
	result2 := status.ActionResult{Action: action2, Output: "error1\nerror2\n", Error: fmt.Errorf("error1")}
	action3 := &status.Action{Description: "action3"}
	result3 := status.ActionResult{Action: action3}

	counts := status.Counts{
		TotalActions: 3,
	}

	counts.StartedActions++
	counts.RunningActions++
	stat.StartAction(action1, counts)

	counts.FinishedActions++
	stat.FinishAction(result1, counts)

	counts.StartedActions++
	stat.StartAction(action2, counts)

	counts.FinishedActions++
	stat.FinishAction(result2, counts)

	counts.StartedActions++
	stat.StartAction(action3, counts)

	counts.FinishedActions++
	counts.RunningActions--
	stat.FinishAction(result3, counts)
}

func actionWithEmptyDescription(stat status.StatusOutput) {
	action1 := &status.Action{Command: "command1"}
	result1 := status.ActionResult{Action: action1}

	counts := status.Counts{
		TotalActions: 1,
	}

	counts.StartedActions++
	counts.RunningActions++
	stat.StartAction(action1, counts)

	counts.FinishedActions++
	counts.RunningActions--
	stat.FinishAction(result1, counts)
}

func actionsWithMessages(stat status.StatusOutput) {
	action1 := &status.Action{Description: "action1"}
	result1 := status.ActionResult{Action: action1}
	action2 := &status.Action{Description: "action2"}
	result2 := status.ActionResult{Action: action2}

	counts := status.Counts{
		TotalActions: 2,
	}

	counts.StartedActions++
	counts.RunningActions++
	stat.StartAction(action1, counts)

	counts.FinishedActions++
	stat.FinishAction(result1, counts)

	stat.Message(status.VerboseLvl, "verbose")
	stat.Message(status.StatusLvl, "status")
	stat.Message(status.PrintLvl, "print")
	stat.Message(status.ErrorLvl, "error")

	counts.StartedActions++
	stat.StartAction(action2, counts)

	counts.FinishedActions++
	counts.RunningActions--
	stat.FinishAction(result2, counts)
}

func actionWithLongDescription(stat status.StatusOutput) {
	action1 := &status.Action{Description: "action with very long description to test eliding"}
	result1 := status.ActionResult{Action: action1}

	counts := status.Counts{
		TotalActions: 2,
	}

	counts.StartedActions++
	counts.RunningActions++
	stat.StartAction(action1, counts)

	counts.FinishedActions++
	counts.RunningActions--
	stat.FinishAction(result1, counts)
}

func actionWithOuptutWithAnsiCodes(stat status.StatusOutput) {
	action1 := &status.Action{Description: "action1"}
	result1 := status.ActionResult{Action: action1, Output: "\x1b[31mcolor\x1b[0m"}

	counts := status.Counts{
		TotalActions: 1,
	}

	counts.StartedActions++
	counts.RunningActions++
	stat.StartAction(action1, counts)

	counts.FinishedActions++
	counts.RunningActions--
	stat.FinishAction(result1, counts)
}
