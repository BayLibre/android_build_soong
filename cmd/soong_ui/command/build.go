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

	"android/soong/ui/build"
	"android/soong/ui/terminal"
)

// A buildCommand represents the command --build-mode to build a set of
// modules. This is based on the build action that is defined after the
// --build-mode flag (--m, --mma, ...). buildCommand is replacing the
// makeCommand where the build action finds the appropriate modules to
// build. Each build action requires the directory (or the list of
// directories) to be specified.
type buildCommand struct {
	command
}

var _ Command = (*buildCommand)(nil)

func init() {
	registerCommand(&buildCommand{command: command{flagName: "build-mode"}})
}

// String returns a blank string since the build command is a bool flag.
func (*buildCommand) String() string {
	return ""
}

// Set is a no-op for the build command.
func (b *buildCommand) Set(string) error {
	return nil
}

// IsBoolFlag returns true as build command is a bool flag. This is to notify
// the FlagSet that the string passed in to Set must be "true".
func (b *buildCommand) IsBoolFlag() bool {
	return true
}

// AddArgs stores the arguments that is owned by the build command.
func (b *buildCommand) AddArgs(args []string) {
	b.args = args
}

// AddFlag adds the --build-mode flag in the flags.
func (b *buildCommand) AddFlag(flags *flag.FlagSet) {
	usage := "build the modules based on the build action type: --m, --mm, --mmm, --mma, --mmma"
	flags.Var(b, b.flagName, usage)
}

// Config returns the build configuration based on the build context.
func (d *buildCommand) Config(ctx build.Context) build.Config {
	// TODO (patricearruda): process the build action and pass in the args to NewConfig. The processing
	// login will be under build/soong/ui/build/buildaction.go.
	return build.NewConfig(ctx)
}

// Stdio returns the standard io implementation for the build command.
func (b *buildCommand) Stdio() terminal.StdioInterface {
	return terminal.StdioImpl{}
}

// Execute is not yet implemented for the build command.
func (b *buildCommand) Execute(ctx build.Context, config build.Config, _ string) {
	// TODO (patricearruda): Add the implementation
}
