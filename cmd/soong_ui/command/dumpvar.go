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
	"os"
	"path/filepath"
	"strings"

	"android/soong/ui/build"
	"android/soong/ui/terminal"
)

// A dumpVarCommand represents the command --dumpvar-mode to print a value of
// the legacy make variable to stdout.
type dumpVarCommand struct {
	command
}

var _ Command = (*dumpVarCommand)(nil)

func init() {
	registerCommand(&dumpVarCommand{command: command{flagName: "dumpvar-mode"}})
}

// String returns a blank string since dump var command is a bool flag.
func (*dumpVarCommand) String() string {
	return ""
}

// Set is a no-op for dump var command.
func (d *dumpVarCommand) Set(string) error {
	return nil
}

// IsBoolFlag returns true as dump var command is a bool flag. This is to notify
// the FlagSet that the string passed in to Set must be "true".
func (d *dumpVarCommand) IsBoolFlag() bool {
	return true
}

// AddArgs stores the arguments that is owned by the dump var command.
func (d *dumpVarCommand) AddArgs(args []string) {
	d.args = args
}

// AddFlag adds the --dumpvar-mode flag in the flags.
func (d *dumpVarCommand) AddFlag(flags *flag.FlagSet) {
	flags.Var(d, d.flagName, "print the value of the legacy make variable VAR to stdout")
}

// Config returns the build configuration based on the build context.
func (d *dumpVarCommand) Config(ctx build.Context) build.Config {
	return build.NewConfig(ctx)
}

// Stdio returns a custom io implementation for the dump var command.
func (d *dumpVarCommand) Stdio() terminal.StdioInterface {
	return terminal.NewCustomStdio(os.Stdin, os.Stderr, os.Stderr)
}

// Execute executes the request based on the arguments passed in after dump var
// command. If "report_config" argument was specified, it will prints the list
// of build configuration environment variables in x=y format. Otherwise, the
// dump var command processes the VAR argument and print out the makefile
// variable value.
func (d *dumpVarCommand) Execute(ctx build.Context, config build.Config, _ string) {
	flags := flag.NewFlagSet("dumpvar", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s --dumpvar-mode [--abs] <VAR>\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "In dumpvar mode, print the value of the legacy make variable VAR to stdout")
		fmt.Fprintln(os.Stderr, "")

		fmt.Fprintln(os.Stderr, "'report_config' is a special case that prints the human-readable config banner")
		fmt.Fprintln(os.Stderr, "from the beginning of the build.")
		fmt.Fprintln(os.Stderr, "")
		flags.PrintDefaults()
	}
	abs := flags.Bool("abs", false, "Print the absolute path of the value")
	flags.Parse(d.args)

	if flags.NArg() != 1 {
		flags.Usage()
		os.Exit(1)
	}

	varName := flags.Arg(0)
	if varName == "report_config" {
		varData, err := build.DumpMakeVars(ctx, config, nil, build.BannerVars)
		if err != nil {
			ctx.Fatal(err)
		}

		fmt.Println(build.Banner(varData))
	} else {
		varData, err := build.DumpMakeVars(ctx, config, nil, []string{varName})
		if err != nil {
			ctx.Fatal(err)
		}

		if *abs {
			var res []string
			for _, path := range strings.Fields(varData[varName]) {
				if abs, err := filepath.Abs(path); err == nil {
					res = append(res, abs)
				} else {
					ctx.Fatalln("Failed to get absolute path of", path, err)
				}
			}
			fmt.Println(strings.Join(res, " "))
		} else {
			fmt.Println(varData[varName])
		}
	}
}
