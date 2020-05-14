/*
 * Copyright (C) 2020 The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package apex

import (
	"android/soong/android"

	"github.com/google/blueprint"
)

func init() {
	android.RegisterSingletonType("apex_depsinfo_singleton", apexDepsInfoSingletonFactory)
}

type apexDepsInfoSingleton struct {
	// Output file with all flatlists from updatable modules' deps-info combined
	updatableFlatListsPath android.OutputPath
}

func apexDepsInfoSingletonFactory() android.Singleton {
	return &apexDepsInfoSingleton{}
}

var (
	mergeApexDepsInfoFilesRule = pctx.AndroidStaticRule("mergeApexDepsInfoFilesRule", blueprint.RuleParams{
		Command:        "cat $out.rsp | xargs cat > $out",
		Rspfile:        "$out.rsp",
		RspfileContent: "$in",
	})

	filterOutExternalApexDepsRule = pctx.AndroidStaticRule("filterOutExternalApexDepsRule", blueprint.RuleParams{
		Command: "cat ${in} | grep -v '(external)' > ${out}",
	})

	diffAllowedApexDepsInfoRule = pctx.AndroidStaticRule("diffAllowedApexDepsInfoRule", blueprint.RuleParams{
		// TODO(satayev): fix formatting of the error message seen by users
		// Diff two given lists while ignoring comments in allowed deps file
		Command: `
			grep -v '^#' ${allowed_flatlists} > $tmp;
			if diff -q ${merged_flatlists} $tmp ; then
				touch ${out};
			else
				echo "Allowed list of updatable module dependencies has changed," \
                  	 "please run 'm apex-allowed-deps-update' to update ${allowed_flatlists}.";
				exit 1;
            fi;
			rm $tmp
		`,
		Description: "Diff ${allowed_flatlists} and ${merged_flatlists}",
	}, "allowed_flatlists", "merged_flatlists", "tmp")
)

func (s *apexDepsInfoSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	updatableFlatLists := android.Paths{}
	ctx.VisitAllModules(func(module android.Module) {
		if binaryInfo, ok := module.(android.ApexBundleDepsInfoIntf); ok {
			if path := binaryInfo.FlatListPath(); path != nil {
				if binaryInfo.Updatable() {
					updatableFlatLists = append(updatableFlatLists, path)
				}
			}
		}
	})

	// Merge all individual flatlists of updatable modules into a single output file
	s.updatableFlatListsPath = android.PathForOutput(ctx, "apex", "depsinfo", "updatable-flatlists.txt")
	ctx.Build(pctx, android.BuildParams{
		Rule:        mergeApexDepsInfoFilesRule,
		Description: "Generate " + s.updatableFlatListsPath.String(),
		Inputs:      updatableFlatLists,
		Output:      s.updatableFlatListsPath,
	})

	// Build a filtered version of updatable flatlists without external dependencies
	filteredFlatLists := android.PathForOutput(ctx, "apex", "depsinfo", "filtered-updatable-flatlists.txt")
	ctx.Build(pctx, android.BuildParams{
		Rule:        filterOutExternalApexDepsRule,
		Description: "Generate " + filteredFlatLists.String(),
		Input:       s.updatableFlatListsPath,
		Output:      filteredFlatLists,
	})

	allowedDeps := android.ExistentPathForSource(ctx, "build/soong/apex/allowed_deps.txt").Path()

	// Check filtered version against allowed deps
	// TODO(satayev): add this to droid target; or checkbuild if it runs on presubmit.
	checkAllowedListResult := android.PathForOutput(ctx, filteredFlatLists.Rel()+".check")
	tmp := android.PathForOutput(ctx, checkAllowedListResult.Rel()+".tmp")
	ctx.Build(pctx, android.BuildParams{
		Rule:   diffAllowedApexDepsInfoRule,
		Output: checkAllowedListResult,
		Args: map[string]string{
			"allowed_flatlists": allowedDeps.String(),
			"merged_flatlists":  filteredFlatLists.String(),
			"tmp":               tmp.String(),
		},
	})

	// Update source allowed deps
	tempPath := android.PathForOutput(ctx, filteredFlatLists.Rel()+".tmp")
	fakeOutput := android.PathForOutput(ctx, filteredFlatLists.Rel()+".fake")
	rule := android.NewRuleBuilder()
	rule.Temporary(tempPath)
	rule.Command().
		Text("(").
		// Copy comments from the source version of the list
		Text("grep '^#' " + allowedDeps.String()).
		Text(">").Output(tempPath).Text(";").
		// Append generated flatlists
		Text("cat").Input(filteredFlatLists).
		Text(">>").Output(tempPath).
		Text(")")
	rule.Command().
		Text("cp -f ").Input(tempPath).Text(allowedDeps.String())
	rule.Command().
		Text("touch").Output(fakeOutput)
	rule.DeleteTemporaryFiles()
	// TODO(satayev): add phony? for `m apex-allowed-deps-update` to work.
	rule.Build(pctx, ctx, "apex-allowed-deps-update", "Update "+allowedDeps.String())
}
