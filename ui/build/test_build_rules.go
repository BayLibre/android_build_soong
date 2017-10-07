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
	"strings"
)

// Checks for files in the out directory that have a rule that depends on them but no rule to
// create them.
func runTestBuildRules(ctx Context, config Config) {

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
			continue
		}
		if strings.HasPrefix(line, bootstrapDir) || strings.HasPrefix(line, miniBootstrapDir) {
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
