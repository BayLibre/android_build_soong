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

// A dumpVarsCommand represents the command --dumpvars-mode to print the list
// the value of one or more makefile variables to stdout.
type dumpVarsCommand struct {
	command
}

var _ Command = (*dumpVarsCommand)(nil)

func init() {
	registerCommand(&dumpVarsCommand{command: command{flagName: "dumpvars-mode"}})
}

// String returns a blank string since dump var command is a bool flag.
func (*dumpVarsCommand) String() string {
	return ""
}

// Set is a no-op for dump vars command.
func (d *dumpVarsCommand) Set(str string) error {
	return nil
}

// IsBoolFlag returns true as dump vars command is a bool flag. This is to notify
// the FlagSet that the string passed in to Set must be "true".
func (d *dumpVarsCommand) IsBoolFlag() bool {
	return true
}

// AddArgs stores the arguments that is owned by the dump vars command.
func (d *dumpVarsCommand) AddArgs(args []string) {
	d.args = args
}

// AddFlag adds the --dumpvars-mode flag in the flags.
func (d *dumpVarsCommand) AddFlag(flags *flag.FlagSet) {
	flags.Var(d, d.flagName, "dump the values of one or more legacy make variables, in shell syntax")
}

// Config returns the build configuration based on the build context.
func (d *dumpVarsCommand) Config(ctx build.Context) build.Config {
	return build.NewConfig(ctx)
}

// Stdio returns a custom io implementation for the dump vars command.
func (d *dumpVarsCommand) Stdio() terminal.StdioInterface {
	return terminal.NewCustomStdio(os.Stdin, os.Stderr, os.Stderr)
}

// Execute executes the request based on the arguments passed in after dump
// vars command. "report_config" is a special variable passed in to --vars
// argument which prints the list of build configuration environment variables
// in x=y format. --vars accepts a list of makefile variables and prints the
// value of each one line by line.
func (d *dumpVarsCommand) Execute(ctx build.Context, config build.Config, _ string) {
	flags := flag.NewFlagSet("dumpvars", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s --dumpvars-mode [--vars=\"VAR VAR ...\"]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "In dumpvars mode, dump the values of one or more legacy make variables, in")
		fmt.Fprintln(os.Stderr, "shell syntax. The resulting output may be sourced directly into a shell to")
		fmt.Fprintln(os.Stderr, "set corresponding shell variables.")
		fmt.Fprintln(os.Stderr, "")

		fmt.Fprintln(os.Stderr, "'report_config' is a special case that dumps a variable containing the")
		fmt.Fprintln(os.Stderr, "human-readable config banner from the beginning of the build.")
		fmt.Fprintln(os.Stderr, "")
		flags.PrintDefaults()
	}

	varsStr := flags.String("vars", "", "Space-separated list of variables to dump")
	absVarsStr := flags.String("abs-vars", "", "Space-separated list of variables to dump (using absolute paths)")

	varPrefix := flags.String("var-prefix", "", "String to prepend to all variable names when dumping")
	absVarPrefix := flags.String("abs-var-prefix", "", "String to prepent to all absolute path variable names when dumping")

	flags.Parse(d.args)

	if flags.NArg() != 0 {
		flags.Usage()
		os.Exit(1)
	}

	vars := strings.Fields(*varsStr)
	absVars := strings.Fields(*absVarsStr)

	allVars := append([]string{}, vars...)
	allVars = append(allVars, absVars...)

	if i := indexList("report_config", allVars); i != -1 {
		allVars = append(allVars[:i], allVars[i+1:]...)
		allVars = append(allVars, build.BannerVars...)
	}

	if len(allVars) == 0 {
		return
	}

	varData, err := build.DumpMakeVars(ctx, config, nil, allVars)
	if err != nil {
		ctx.Fatal(err)
	}

	for _, name := range vars {
		if name == "report_config" {
			fmt.Printf("%sreport_config='%s'\n", *varPrefix, build.Banner(varData))
		} else {
			fmt.Printf("%s%s='%s'\n", *varPrefix, name, varData[name])
		}
	}
	for _, name := range absVars {
		var res []string
		for _, path := range strings.Fields(varData[name]) {
			abs, err := filepath.Abs(path)
			if err != nil {
				ctx.Fatalln("Failed to get absolute path of", path, err)
			}
			res = append(res, abs)
		}
		fmt.Printf("%s%s='%s'\n", *absVarPrefix, name, strings.Join(res, " "))
	}
}
