// Copyright 2020 Google Inc. All rights reserved.
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

package android

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/blueprint"
)

func init() {
	RegisterSingletonType("bazel_overlay", BazelOverlaySingleton)
}

func BazelOverlaySingleton() Singleton {
	return &bazelOverlaySingleton{}
}

type bazelOverlaySingleton struct{}

func (c *bazelOverlaySingleton) GenerateBuildActions(ctx SingletonContext) {
	ctx.VisitAllModules(func(module Module) {
	})

	bazelOverlayDirectory := PathForOutput(ctx, "bazel_overlay")
	bazelOverlayWorkspaceFile := bazelOverlayDirectory.Join(ctx, "WORKSPACE")
	primaryBuilder := primaryBuilderPath(ctx)
	bazelOverlay := ctx.Rule(pctx, "bazelOverlay",
		blueprint.RuleParams{
			Command: fmt.Sprintf(
				"rm -rf ${outDir}/* && %s --bazel_overlay_dir %s %s",
				primaryBuilder.String(),
				bazelOverlayDirectory.String(),
				strings.Join(os.Args[1:], " ")),
			CommandDeps: []string{primaryBuilder.String()},
			Description: fmt.Sprintf(
				"Creating the Bazel overlay workspace with %s at $outDir",
				primaryBuilder.Base()),
		},
		"outDir")

	ctx.Build(pctx, BuildParams{
		Rule:   bazelOverlay,
		Output: bazelOverlayWorkspaceFile,
		Args: map[string]string{
			"outDir": bazelOverlayDirectory.String(),
		},
	})

	// Add a phony target for building the bazel overlay
	ctx.Phony("bazel_overlay", bazelOverlayWorkspaceFile)
}
