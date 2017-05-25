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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func runNinja(ctx Context, config Config) {
	ctx.BeginTrace("ninja")
	defer ctx.EndTrace()

	executable := config.PrebuiltBuildTool("ninja")
	args := []string{
		"-d", "keepdepfile",
	}

	args = append(args, config.NinjaArgs()...)

	var parallel int
	if config.UseGoma() {
		parallel = config.RemoteParallel()
	} else {
		parallel = config.Parallel()
	}
	args = append(args, "-j", strconv.Itoa(parallel))
	if config.keepGoing != 1 {
		args = append(args, "-k", strconv.Itoa(config.keepGoing))
	}

	args = append(args, "-f", config.CombinedNinjaFile())

	if config.IsVerbose() {
		args = append(args, "-v")
	}
	args = append(args, "-w", "dupbuild=err")

	cmd := Command(ctx, config, "ninja", executable, args...)
	cmd.Environment.AppendFromKati(config.KatiEnvFile())

	// Allow both NINJA_ARGS and NINJA_EXTRA_ARGS, since both have been
	// used in the past to specify extra ninja arguments.
	if extra, ok := cmd.Environment.Get("NINJA_ARGS"); ok {
		cmd.Args = append(cmd.Args, strings.Fields(extra)...)
	}
	if extra, ok := cmd.Environment.Get("NINJA_EXTRA_ARGS"); ok {
		cmd.Args = append(cmd.Args, strings.Fields(extra)...)
	}

	if _, ok := cmd.Environment.Get("NINJA_STATUS"); !ok {
		cmd.Environment.Set("NINJA_STATUS", "[%p %f/%t] ")
	}

	cmd.Stdin = ctx.Stdin()
	cmd.Stdout = ctx.Stdout()
	cmd.Stderr = ctx.Stderr()
	logPath := filepath.Join(config.OutDir(), ".ninja_log")
	ninjaHeartbeatDuration := time.Minute * 5
	if overrideText, ok := cmd.Environment.Get("NINJA_HEARTBEAT_INTERVAL"); ok {
		// For example, "1m"
		overrideDuration, err := time.ParseDuration(overrideText)
		if err == nil && overrideDuration.Seconds() > 0 {
			ninjaHeartbeatDuration = overrideDuration
		}
	}
	// if the ninja log isn't updated often enough, then we want to show some diagnostics
	PollUntil(ninjaHeartbeatDuration, cmd.Done, ninjaStatusChecker(ctx, config, logPath))

	startTime := time.Now()
	defer ctx.ImportNinjaLog(logPath, startTime)

	cmd.RunOrFatal()
}

// ninjaStatusChecker returns a function that checks whether Ninja appears to be stuck (looping forever).
// Note that even if Ninja is not stuck, it's still possible for several individual tasks to be stuck.
//   For example, if Ninja is running 50 tasks at once of which 49 are stuck, then the remaining 1 worker can slowly
//   complete the majority of the build. However, in this situation the caller isn't going to want to wait for Ninja to
//   finish most of the build in a single-threaded manner, and will probably kill the build before it completes.
// So, this function is a sufficient but not necessary condition for suspecting that Ninja or one of its tasks are
// likely to be stuck
func ninjaStatusChecker(ctx Context, config Config, filepath string) func() {
	var prevTime time.Time
	checker := func() {
		info, err := os.Stat(filepath)
		var newTime time.Time
		if err == nil {
			newTime = info.ModTime()
		}
		if newTime == prevTime {
			// ninja may be stuck
			dumpStucknessDiagnostics(ctx, config, filepath, newTime)
		}
		prevTime = newTime

	}
	return checker
}

// dumpStucknessDiagnostics gets called when it is suspected that Ninja is stuck and we want to output some diagnostics
func dumpStucknessDiagnostics(ctx Context, config Config, statusPath string, lastUpdated time.Time) {

	ctx.Verbosef("ninja may be stuck; last update to %v was %v. dumping process tree...", statusPath, lastUpdated)

	// The "pstree" command doesn't exist on Mac, but "pstree" on Linux gives more convenient output than "ps"
	// So, we try pstree first, and ps second
	pstreeCommandText := fmt.Sprintf("pstree -pal %v", os.Getpid())
	psCommandText := "ps -ef"
	commandText := pstreeCommandText + " || " + psCommandText

	cmd := Command(ctx, config, "dump process tree", "bash", "-c", commandText)
	output := cmd.CombinedOutputOrFatal()
	ctx.Verbose(string(output))

	ctx.Printf("done\n")
}
