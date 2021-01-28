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

// The Bazel bp2build code generator is responsible for writing .bzl files that are equivalent to
// Android.bp files that are capable of being built with Bazel.
func Codegen(ctx CodegenContext) {
	outputDir := android.PathForOutput(ctx, "bp2build")
	android.RemoveAllOutputDir(outputDir)

	ruleShims := CreateRuleShims(android.ModuleTypeFactories())

	buildToTargets := GenerateSoongModuleTargets(ctx.Context(), ctx.mode)

	filesToWrite := CreateBazelFiles(ruleShims, buildToTargets, ctx.mode)
	for _, f := range filesToWrite {
		if err := writeFile(outputDir, ctx, f); err != nil {
			fmt.Errorf("Failed to write %q (dir %q) due to %q", f.Basename, f.Dir, err)
		}
	}

	// Workarounds to support running bp2build in a clean AOSP checkout with no
	// prior builds, and exiting early as soon as the BUILD files get generated,
	// therefore not creating build.ninja files that soong_ui and callers of
	// soong_build expects.
	//
	// These files are: build.ninja and build.ninja.d. Since Kati hasn't been
	// ran as well, and `nothing` is defined in a .mk file, there isn't a ninja
	// target called `nothing`, so we manually create it here.
	if ctx.mode == Bp2Build {
		android.WriteFileToOutputDir(android.PathForOutput(ctx, "build.ninja"), []byte("build nothing: phony\n  phony_output = true\n"), 0666)
		// No need to invalidate the ninja file if it doesn't need to be regenerated.
		android.WriteFileToOutputDir(android.PathForOutput(ctx, "build.ninja.d"), []byte(""), 0666)
	}
}

func writeFile(outputDir android.OutputPath, ctx android.PathContext, f BazelFile) error {
	return writeReadOnlyFile(ctx, getOutputPath(outputDir, ctx, f.Dir), f.Basename, f.Contents)
}

func getOutputPath(outputDir android.OutputPath, ctx android.PathContext, dir string) android.OutputPath {
	return outputDir.Join(ctx, dir)
}

// The auto-conversion directory should be read-only, sufficient for bazel query. The files
// are not intended to be edited by end users.
func writeReadOnlyFile(ctx android.PathContext, dir android.OutputPath, baseName, content string) error {
	android.CreateOutputDirIfNonexistent(dir, os.ModePerm)
	pathToFile := dir.Join(ctx, baseName)

	// 0444 is read-only
	err := android.WriteFileToOutputDir(pathToFile, []byte(content), 0444)

	return err
}
