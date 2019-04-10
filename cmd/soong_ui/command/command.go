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
	"errors"
	"flag"

	"android/soong/ui/build"
	"android/soong/ui/terminal"
)

// To create a new command:
// 	 * Create a new go file with the command name as the filename.
// 	 * Define a new command struct that embed the command struct.
//   * Implement the Command and flag.Getter interface in the new command struct.
//   * Register the command in the init function by calling registerCommand.
//   * Add the new go file and it's dependencies to the Android.bp file.
//
// To delete a new command:
//   * Comment out the registerCommand function call in the init function.

// Command is an interface that executes the operation of the command.
// A command requires to use the flag Value interface in order for the
// command to be added in the FlagSet for arguments processing. Command
// is responsible to provide the flag and the flag description and the
// execution implementation.
type Command interface {

	// Returns true if the flag name belong to the Command.
	IsFlag(string) bool

	// Add the flag (name, description) to the FlagSet.
	AddFlag(*flag.FlagSet)

	// Store the arguments from the Command line.
	AddArgs([]string)

	// Returns the build configuration based on the stored arguments and
	// the build context.
	Config(ctx build.Context) build.Config

	// Execute the Command operation.
	Execute(ctx build.Context, config build.Config, logsDir string)

	// Returns what type of IO redirection this Command requires.
	Stdio() terminal.StdioInterface
}

// A command represents the list of common behaviour required
// for all commands to support.
type command struct {
	flagName string
	args     []string
}

func (c *command) IsFlag(flag string) bool {
	return "--"+c.flagName == flag
}

// list of registered commands
var commands []Command

// registerCommand registers a Command.
func registerCommand(cmd Command) {
	commands = append(commands, cmd)
}

// New returns an executable soong command based on the specified arguments.
// New assumes that the soong_ui options has been processed and stripped away
// from args. An error is returned if the command was not specified in the
// args list.
func New(args []string) (Command, error) {
	defaultRetErr := errors.New("no command specified")

	if len(args) < 1 {
		return nil, defaultRetErr
	}

	// If there is more than one command specified, abort it as there should
	// be only one command specified from the args.
	var retCmd Command
	for _, cmd := range commands {
		if cmd.IsFlag(args[0]) {
			if retCmd != nil {
				return nil, errors.New("only one command must to be specified")
			}
			retCmd = cmd
		}
	}

	if retCmd == nil {
		return nil, defaultRetErr
	}

	// Remove the first argument from the list since it is the command flag
	retCmd.AddArgs(args[1:])

	return retCmd, nil
}
