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

package build

import (
	"bufio"
	"path/filepath"
	"runtime"
	"strings"
)

// Checks for files in the out directory that have a rule that depends on them but no rule to
// create them. This catches a common set of build failures where a rule to generate a file is
// deleted (either by deleting a module in an Android.mk file, or by modifying the build system
// incorrectly).  These failures are often not caught by a local incremental build because the
// previously built files are still present in the output directory.
func testForDanglingRules(ctx Context, config Config) {
	// Many modules are disabled on mac.  Checking for dangling rules would cause lots of build
	// breakages, and presubmit wouldn't catch them, so just disable the check.
	if runtime.GOOS != "linux" {
		return
	}

	ctx.BeginTrace("test for dangling rules")
	defer ctx.EndTrace()

	// Get a list of leaf nodes in the dependency graph from ninja
	executable := config.PrebuiltBuildTool("ninja")

	args := []string{}
	args = append(args, config.NinjaArgs()...)
	args = append(args, "-f", config.CombinedNinjaFile())
	args = append(args, "-t", "targets", "rule")

	cmd := Command(ctx, config, "ninja", executable, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ctx.Fatal(err)
	}

	cmd.StartOrFatal()

	outDir := config.OutDir()
	bootstrapDir := filepath.Join(outDir, "soong", ".bootstrap")
	miniBootstrapDir := filepath.Join(outDir, "soong", ".minibootstrap")

	var danglingRules []string

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, outDir) {
			// Leaf node is not in the out directory.
			continue
		}
		if strings.HasPrefix(line, bootstrapDir) || strings.HasPrefix(line, miniBootstrapDir) {
			// Leaf node is in one of Soong's bootstrap directories, which do not have
			// full build rules in the primary build.ninja file.
			continue
		}
		danglingRules = append(danglingRules, line)
	}

	cmd.WaitOrFatal()

	if len(danglingRules) > 0 {
		ctx.Println("Dependencies in out found with no rule to create them:")
		for _, dep := range danglingRules {
			ctx.Println(dep)
		}
		ctx.Fatal("")
	}
}

// confirmCheckbuildUpToDate runs a dry-run of checkbuild and confirms that all rules are
// up-to-date.  It therefore only makes sense to call this function after a successful checkbuild.
// The usual cause of a violation is when a rule doesn't update its output file, or when a real
// rule depends on a phony rule.
func confirmCheckbuildUpToDate(ctx Context, config Config) {
	ctx.BeginTrace("test for unnecessary rebuilds")
	defer ctx.EndTrace()

	// Get a list of leaf nodes in the dependency graph from ninja
	executable := config.PrebuiltBuildTool("ninja")

	args := []string{}
	args = append(args, "-f", config.CombinedNinjaFile())
	args = append(args, "-n") // dry run
	args = append(args, "checkbuild")
	cmd := Command(ctx, config, "ninja", executable, args...)
	linePrefix := "Unnecessary rebuild: "
	cmd.Environment.Set("NINJA_STATUS", linePrefix)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ctx.Fatal(err)
	}

	cmd.StartOrFatal()

	var unnecessaryRebuilds []string

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, linePrefix) {
			unnecessaryRebuilds = append(unnecessaryRebuilds, strings.TrimPrefix(line, linePrefix))
		}
	}

	cmd.WaitOrFatal()

	if len(unnecessaryRebuilds) > 0 {
		ctx.Println("error: rules unnecessarily rebuilt after second checkbuild:")
		failuresToPrint := len(unnecessaryRebuilds)
		maxToPrint := 5
		if failuresToPrint > maxToPrint {
			unnecessaryRebuilds = unnecessaryRebuilds[0:maxToPrint]
		}
		for _, rule := range unnecessaryRebuilds {
			ctx.Println("   " + rule)
		}
		if failuresToPrint > maxToPrint {
			ctx.Printf("...and %d more\n", failuresToPrint-maxToPrint)
		}
		ctx.Println("This usually means a rule to build a file depends on a phony target.")
		ctx.Println("It can also happen if source files were modified during the build.")
		ctx.Fatal("")
	}
}
