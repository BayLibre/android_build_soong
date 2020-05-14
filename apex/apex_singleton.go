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
	allowedApexDepsInfoCheckResult android.OutputPath
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
		// Diff two given lists while ignoring comments in the allowed deps file
		Description: "Diff ${allowed_flatlists} and ${merged_flatlists}",
		Command: `
			if grep -v '^#' ${allowed_flatlists} | diff -B ${merged_flatlists} -; then
			   touch ${out};
			else
				echo -e "\n******************************";
				echo "ERROR: go/apex-allowed-deps-error";
				echo "******************************";
				echo "Detected changes to allowed dependencies in updatable modules.";
				echo "To fix and update build/soong/apex/allowed_deps.txt, please run:";
				echo "$$ (croot && build/soong/scripts/update-apex-allowed-deps.sh)";
				echo "Members of mainline-modularization@google.com review reflected changes.";
				echo -e "******************************\n";
				exit 1;
			fi;`,
	}, "allowed_flatlists", "merged_flatlists")
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
	updatableFlatListsPath := android.PathForOutput(ctx, "apex", "depsinfo", "updatable-flatlists.txt")
	ctx.Build(pctx, android.BuildParams{
		Rule:   mergeApexDepsInfoFilesRule,
		Inputs: updatableFlatLists,
		Output: updatableFlatListsPath,
	})

	// Build a filtered version of updatable flatlists without external dependencies
	filteredFlatLists := android.PathForOutput(ctx, "apex", "depsinfo", "filtered-updatable-flatlists.txt")
	ctx.Build(pctx, android.BuildParams{
		Rule:   filterOutExternalApexDepsRule,
		Input:  updatableFlatListsPath,
		Output: filteredFlatLists,
	})

	// Check filtered version against allowed deps
	allowedDeps := android.ExistentPathForSource(ctx, "build/soong/apex/allowed_deps.txt").Path()
	s.allowedApexDepsInfoCheckResult = android.PathForOutput(ctx, filteredFlatLists.Rel()+".check")
	ctx.Build(pctx, android.BuildParams{
		Rule:   diffAllowedApexDepsInfoRule,
		Input:  filteredFlatLists,
		Output: s.allowedApexDepsInfoCheckResult,
		Args: map[string]string{
			"allowed_flatlists": allowedDeps.String(),
			"merged_flatlists":  filteredFlatLists.String(),
		},
	})
}

func (s *apexDepsInfoSingleton) MakeVars(ctx android.MakeVarsContext) {
	// Export check result to Make. The path is added to droidcore.
	ctx.Strict("APEX_ALLOWED_DEPS_CHECK", s.allowedApexDepsInfoCheckResult.String())
}
