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

package bp2build

import (
	"android/soong/android"
	"fmt"
	"os"
)

// Codegen is the backend of bp2build. The code generator is responsible for
// writing .bzl files that are equivalent to Android.bp files that are capable
// of being built with Bazel.
func Codegen(ctx *CodegenContext) (CodegenMetrics, []string) {
	outputDir := android.PathForOutput(ctx, "bp2build")
	android.RemoveAllOutputDir(outputDir)

	ruleShims := CreateRuleShims(android.ModuleTypeFactories())

	buildToTargets, metrics := GenerateBazelTargets(ctx)

	filesToWrite := CreateBazelFiles(ruleShims, buildToTargets, ctx.mode)
	extraNinjaDeps := []string{}

	for _, f := range filesToWrite {
		p := getOrCreateOutputDir(outputDir, ctx, f.Dir).Join(ctx, f.Basename)
		if err := writeReadOnlyFile(ctx, p, f.Contents); err != nil {
			fmt.Errorf("Failed to write %q (dir %q) due to %q", f.Basename, f.Dir, err)
		}
		// if these generated files are modified, regenerate on next run.
		extraNinjaDeps = append(extraNinjaDeps, p.String())
	}

	return metrics, extraNinjaDeps
}

// Get the output directory and create it if it doesn't exist.
func getOrCreateOutputDir(outputDir android.OutputPath, ctx android.PathContext, dir string) android.OutputPath {
	dirPath := outputDir.Join(ctx, dir)
	android.CreateOutputDirIfNonexistent(dirPath, os.ModePerm)
	return dirPath
}

// The auto-conversion directory should be read-only. The files are not intended
// to be edited by end users.
func writeReadOnlyFile(ctx android.PathContext, pathToFile android.OutputPath, content string) error {
	return android.WriteFileToOutputDir(pathToFile, []byte(content), 0444) // 0444 is read-only
}
