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
)

func init() {
	android.RegisterSingletonType("apex_depsinfo", apexDepsInfoSingletonFactory)
}

var apexDepsInfoSingletonPathsKey = android.NewOnceKey("apexDepsInfoSingletonPathsKey")

type apexDepsInfoPaths struct {
	flatListsPath           android.OutputPath
	updatableQFlatListsPath android.OutputPath
}

func apexDepsInfoSingletonPaths(ctx android.PathContext) apexDepsInfoPaths {
	return ctx.Config().Once(apexDepsInfoSingletonPathsKey, func() interface{} {
		return apexDepsInfoPaths{
			flatListsPath:           android.PathForOutput(ctx, "apex", "depsinfo", "flatlists.txt"),
			updatableQFlatListsPath: android.PathForOutput(ctx, "apex", "depsinfo", "updatableQflatlists.txt"),
		}
	}).(apexDepsInfoPaths)
}

func apexDepsInfoSingletonFactory() android.Singleton {
	return &apexDepsInfoSingleton{}
}

type apexDepsInfoSingleton struct {
}

func (s *apexDepsInfoSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	flatLists := android.Paths{}
	updatableQFlatLists := android.Paths{}
	ctx.VisitAllModules(func(module android.Module) {
		if binaryInfo, ok := module.(android.ApexBundleDepsInfoIntf); ok {
			if path := binaryInfo.FlatListPath(); path != nil {
				if binaryInfo.Updatable() && (binaryInfo.MinSdkVersion() == "29" || binaryInfo.MinSdkVersion() == "28") {
					updatableQFlatLists = append(updatableQFlatLists, path)
				}
				flatLists = append(flatLists, path)
			}
		}
	})

	// TODO(satayev): "arg list too long" for all flatlists, limit to updatable & minSdkVersion < 30 ones.

	rule := android.NewRuleBuilder()
	rule.Command().
		Text("cat").
		Inputs(updatableQFlatLists).
		FlagWithOutput("> ", apexDepsInfoSingletonPaths(ctx).updatableQFlatListsPath)
	rule.Build(pctx, ctx, "Generate out/soong/apex/depsinfo/updatableQflatlists.txt", "Generate out/soong/apex/depsinfo/updatableQflatlists.txt")
}
