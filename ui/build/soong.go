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
	"os"
	"path/filepath"
)

func runSoongBootstrap(ctx Context, config Config) {
	ctx.BeginTrace("bootstrap soong")
	defer ctx.EndTrace()

	cmd := Command(ctx, config, "soong bootstrap", "build/blueprint/bootstrap.bash", "-t")
	cmd.Environment.Set("BLUEPRINTDIR", "./build/blueprint")
	cmd.Environment.Set("BOOTSTRAP", "./build/blueprint/bootstrap.bash")
	cmd.Environment.Set("BUILDDIR", config.SoongOutDir())
	cmd.Environment.Set("GOROOT", filepath.Join("./prebuilts/go", config.HostPrebuiltTag()))
	cmd.Environment.Set("NINJA_BUILDDIR", config.OutDir())
	cmd.Environment.Set("SRCDIR", ".")
	cmd.Environment.Set("TOPNAME", "Android.bp")
	cmd.Sandbox = soongSandbox
	cmd.Stdout = ctx.Stdout()
	cmd.Stderr = ctx.Stderr()
	cmd.RunOrFatal()
}

func runSoong(ctx Context, config Config) {
	ctx.BeginTrace("soong")
	defer ctx.EndTrace()

	envFile := filepath.Join(config.SoongOutDir(), ".soong.environment")
	envTool := filepath.Join(config.SoongOutDir(), ".bootstrap/bin/soong_env")
	if _, err := os.Stat(envFile); err == nil {
		if _, err := os.Stat(envTool); err == nil {
			cmd := Command(ctx, config, "soong_env", envTool, envFile)
			cmd.Sandbox = soongSandbox
			cmd.Stdout = ctx.Stdout()
			cmd.Stderr = ctx.Stderr()
			if err := cmd.Run(); err != nil {
				ctx.Verboseln("soong_env failed, forcing manifest regeneration")
				os.Remove(envFile)
			}
		} else {
			ctx.Verboseln("Missing soong_env tool, forcing manifest regeneration")
			os.Remove(envFile)
		}
	} else if !os.IsNotExist(err) {
		ctx.Fatalf("Failed to stat %f: %v", envFile, err)
	}

	cmd := Command(ctx, config, "soong", "build/blueprint/blueprint.bash", "-w", "dupbuild=err")
	if config.IsVerbose() {
		cmd.Args = append(cmd.Args, "-v")
	}
	cmd.Environment.Set("BUILDDIR", config.SoongOutDir())
	cmd.Environment.Set("NINJA", config.PrebuiltBuildTool("ninja"))
	cmd.Environment.Set("SKIP_NINJA", "true")
	cmd.Environment.Set("NO_DEPRECATION_WARNING", "true")
	cmd.Sandbox = soongSandbox
	cmd.Stdin = ctx.Stdin()
	cmd.Stdout = ctx.Stdout()
	cmd.Stderr = ctx.Stderr()
	cmd.RunOrFatal()
}
