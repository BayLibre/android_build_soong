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

package bazelconverter

import (
	"android/soong/android"
	"os"
	"sort"

	"github.com/google/blueprint"
)

// The Bazel QueryView singleton is responsible for generating the Ninja actions
// for calling the soong_build primary builder in the main build.ninja file.
func init() {
	android.RegisterPreSingletonType("androidbp_to_build", AndroidBpToBuildSingleton)
	android.RegisterSingletonType("bazel_queryview_generation", QueryviewSingleton)
}

var (
	bazelRulesSubDir = "build/bazel/queryview_rules"
)

func AndroidBpToBuildSingleton() android.Singleton {
	return &androidBpToBuildSingleton{
		name: "androidbp2Build",
	}
}

func QueryviewSingleton() android.Singleton {
	return &androidBpToBuildSingleton{
		name: "queryview",
	}
}

type androidBpToBuildSingleton struct {
	name        string
	description string
	outputDir   android.OutputPath

	buildToTargets map[string][]BazelTarget
	ruleShims      map[string]RuleShim
}

func (s *androidBpToBuildSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	s.outputDir = android.PathForOutput(ctx, s.name)
	android.RemoveAllOutputDir(s.outputDir)

	s.ruleShims = createRuleShims()

	s.buildToTargets = make(map[string][]BazelTarget)
	ctx.VisitAllModulesBlueprint(func(m blueprint.Module) {
		dir := ctx.ModuleDir(m)
		t := GenerateSoongModuleTarget(ctx, m)
		s.buildToTargets[ctx.ModuleDir(m)] = append(s.buildToTargets[dir], t)
	})

	// TODO: write to file
	var err error
	err = s.writeBaseFiles(ctx)
	if err != nil {
		ctx.Errorf("%s Failed to write base files %q", s.name, err)
	}
	err = s.writeRules(ctx)
	if err != nil {
		ctx.Errorf("%s Failed to write rule files %q", s.name, err)
	}
	err = s.writeBuildFiles(ctx)
	if err != nil {
		ctx.Errorf("%s Failed to write build files %q", s.name, err)
	}
}

func (s *androidBpToBuildSingleton) writeBaseFiles(ctx android.PathContext) error {
	var err error
	// Write top level files: WORKSPACE and BUILD. These files are empty.
	if err = writeReadOnlyFile(ctx, s.outputDir, "WORKSPACE", ""); err != nil {
		return err
	}

	// Used to denote that the top level directory is a package.
	if err = writeReadOnlyFile(ctx, s.outputDir, "BUILD", ""); err != nil {
		return err
	}

	return nil
}

func (s *androidBpToBuildSingleton) writeRules(ctx android.PathContext) error {
	bazelRulesDir := s.getOutputPath(ctx, bazelRulesSubDir)
	var err error
	if err = writeReadOnlyFile(ctx, bazelRulesDir, "BUILD", ""); err != nil {
		return err
	}

	if err = writeReadOnlyFile(ctx, bazelRulesDir, "providers.bzl", providersBzl); err != nil {
		return err
	}

	for bzlFileName, ruleShim := range s.ruleShims {
		if err = writeReadOnlyFile(ctx, bazelRulesDir, bzlFileName+".bzl", ruleShim.content); err != nil {
			return err
		}
	}

	return writeReadOnlyFile(ctx, bazelRulesDir, "soong_module.bzl", generateSoongModuleBzl(s.ruleShims))
	return nil
}

func (s *androidBpToBuildSingleton) writeBuildFiles(ctx android.PathContext) error {
	for _, dir := range android.SortedStringKeys(s.buildToTargets) {
		content := soongModuleLoad
		targets := s.buildToTargets[dir]
		sort.Slice(targets, func(i, j int) bool { return targets[i].name < targets[j].name })
		for _, t := range targets {
			content += "\n\n"
			content += t.content
		}
		if err := writeReadOnlyFile(ctx, s.getOutputPath(ctx, dir), "BUILD.bazel", content); err != nil {
			return err
		}
	}
	return nil
}

func (s *androidBpToBuildSingleton) getOutputPath(ctx android.PathContext, dir string) android.OutputPath {
	return s.outputDir.Join(ctx, dir)
}

// The auto-conversion directory should be read-only, sufficient for bazel query. The files
// are not intended to be edited by end users.
func writeReadOnlyFile(ctx android.PathContext, dir android.OutputPath, baseName, content string) error {
	android.CreateOutputDirIfNonexistent(dir, os.ModePerm)
	pathToFile := dir.Join(ctx, baseName)
	// 0444 is read-only
	err := android.WriteFileToOutputDir(pathToFile, []byte(content), 0444)

	// TODO touch file via ninja so it's aware?

	return err
}

func createDirectoryIfNonexistent(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// fmt.Println("mkdir " + dir)
		// os.MkdirAll(dir, os.ModePerm)
	}
}
