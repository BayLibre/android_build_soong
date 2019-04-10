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
	"fmt"
	"time"

	"android/soong/ui/build"
	"android/soong/ui/terminal"
)

// A makeCommand represents the command --make-mode to build a set of modules
// where each module name is formatted as MODULES-IN-<dirname>. This is the
// legacy building mode (--build-mode is the current version)
type makeCommand struct {
	command
}

var _ Command = (*makeCommand)(nil)

func init() {
	registerCommand(&makeCommand{command: command{flagName: "make-mode"}})
}

// String returns a blank string since the make command is a bool flag.
func (*makeCommand) String() string {
	return ""
}

// Set marks the make command as the execution operation. The string parameter
// is ignored since makeCommand is a bool flag.
func (m *makeCommand) Set(string) error {
	return nil
}

// IsBoolFlag returns true as make command is a bool flag. This is to notify
// the FlagSet that the string passed in to Set must be "true".
func (m *makeCommand) IsBoolFlag() bool {
	return true
}

// AddArgs stores the arguments that is owned by the make command.
func (m *makeCommand) AddArgs(args []string) {
	m.args = args
}

// AddFlag adds the --make-mode flag in the flags.
func (m *makeCommand) AddFlag(flags *flag.FlagSet) {
	usage := "build the modules by the target name (i.e. soong_docs)"
	flags.Var(m, m.flagName, usage)
}

// // Config returns the build configuration based on the build context. The
// args are passed in to the build config as additional processing is done.
func (m *makeCommand) Config(ctx build.Context) build.Config {
	return build.NewConfig(ctx, m.args...)
}

// Stdio returns the standard io implementation for make-mode build.
func (*makeCommand) Stdio() terminal.StdioInterface {
	return terminal.StdioImpl{}
}

// Execute builds the modules based on the build configuration. The logsDir
// is used for informational purpose only since showcommands has been
// deprecated.
func (*makeCommand) Execute(ctx build.Context, config build.Config, logsDir string) {
	if config.IsVerbose() {
		writer := ctx.Writer
		writer.Print("! The argument `showcommands` is no longer supported.")
		writer.Print("! Instead, the verbose log is always written to a compressed file in the output dir:")
		writer.Print("!")
		writer.Print(fmt.Sprintf("!   gzip -cd %s/verbose.log.gz | less -R", logsDir))
		writer.Print("!")
		writer.Print("! Older versions are saved in verbose.log.#.gz files")
		writer.Print("")
		time.Sleep(5 * time.Second)
	}

	toBuild := build.BuildAll
	if config.Checkbuild() {
		toBuild |= build.RunBuildTests
	}
	build.Build(ctx, config, toBuild)
}
