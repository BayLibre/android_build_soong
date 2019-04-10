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

package command

import (
	"flag"
	"reflect"
	"testing"

	"android/soong/ui/build"
	"android/soong/ui/terminal"
)

type fakeCmdOne struct {
	command
}

var _ Command = (*fakeCmdOne)(nil)

func (*fakeCmdOne) String() string {
	return ""
}

func (f *fakeCmdOne) Set(str string) error {
	return nil
}

func (f *fakeCmdOne) IsBoolFlag() bool {
	return true
}

func (f *fakeCmdOne) AddArgs(args []string) {
	f.args = args
}

func (f *fakeCmdOne) AddFlag(flags *flag.FlagSet) {
	flags.Var(f, f.flagName, "fake command for testing")
}

func (*fakeCmdOne) Config(ctx build.Context) build.Config {
	return build.NewConfig(ctx)
}

func (*fakeCmdOne) Stdio() terminal.StdioInterface {
	return terminal.StdioImpl{}
}

func (f *fakeCmdOne) Execute(ctx build.Context, config build.Config, _ string) {
}

type fakeCmdTwo struct {
	command
}

var _ Command = (*fakeCmdTwo)(nil)

func (*fakeCmdTwo) String() string {
	return ""
}

func (f *fakeCmdTwo) Set(str string) error {
	return nil
}

func (f *fakeCmdTwo) IsBoolFlag() bool {
	return true
}

func (f *fakeCmdTwo) AddArgs(args []string) {
	f.args = args
}

func (f *fakeCmdTwo) AddFlag(flags *flag.FlagSet) {
	flags.Var(f, f.flagName, "fake command for testing")
}

func (*fakeCmdTwo) Config(ctx build.Context) build.Config {
	return build.NewConfig(ctx)
}

func (*fakeCmdTwo) Stdio() terminal.StdioInterface {
	return terminal.StdioImpl{}
}

func (f *fakeCmdTwo) Execute(ctx build.Context, config build.Config, _ string) {
}

func TestNew(t *testing.T) {
	defaultErrStr := "no command specified"
	tests := []struct {
		description  string
		args         []string
		errStr       string
		expectedMode reflect.Type
	}{{
		description:  "valid command - fakeCmdOne",
		args:         []string{"--fake-one", "arg1", "arg2", "arg3"},
		expectedMode: reflect.TypeOf(&fakeCmdOne{}),
	}, {
		description:  "valid command - fakeCmdTwo",
		args:         []string{"--fake-two", "arg1", "arg2", "arg3"},
		expectedMode: reflect.TypeOf(&fakeCmdTwo{}),
	}, {
		description: "no command found",
		args:        []string{"--some-fake", "arg1", "arg2", "arg3"},
		errStr:      defaultErrStr,
	}, {
		description: "no command specified",
		args:        []string{"arg1", "arg2", "arg3"},
		errStr:      defaultErrStr,
	}, {
		description:  "valid --fake with double dashed arguments",
		args:         []string{"--fake-one", "--var1", "--var2", "arg1"},
		expectedMode: reflect.TypeOf(&fakeCmdOne{}),
	}, {
		// fails as the options should have been processed before
		// processing the command.
		description: "soong_ui options with --fake command specified",
		args:        []string{"--metrics", "--dump-stack", "--fake", "arg1", "arg2", "arg3"},
		errStr:      defaultErrStr,
	}}

	orgCommands := commands
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			commands = append(orgCommands,
				&fakeCmdOne{command: command{flagName: "fake-one"}},
				&fakeCmdTwo{command: command{flagName: "fake-two"}})
			defer func() { commands = orgCommands }()
			m, err := New(tt.args)
			if tt.errStr == "" {
				if err != nil {
					t.Fatalf("not expecting error: %v", err)
				}
				if m == nil {
					t.Fatalf("returned nil for mode")
				}

				modeType := reflect.TypeOf(m)
				if modeType != tt.expectedMode {
					t.Errorf("expected %s, got %s for more type", tt.expectedMode, modeType)
				}
			} else {
				if err == nil {
					t.Fatalf("expecting error: %s", tt.errStr)
				}

				if err.Error() != tt.errStr {
					t.Errorf("expected %s, got %s for error string", tt.errStr, err.Error())
				}
			}
		})
	}
}
