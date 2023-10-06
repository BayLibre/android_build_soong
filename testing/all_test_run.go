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

package testrun

import "android/soong/android"

func AllTestRunFactory() android.Singleton {
	return &allTestRunSingleton{}
}

type allTestRunSingleton struct {
	intermediatePath android.OutputPath
}

func (this *allTestRunSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	var intermediateMetadataPaths android.Paths

	ctx.VisitAllModules(func(module android.Module) {
		if !ctx.ModuleHasProvider(module, testRunProviderKey) {
			return
		}
		intermediateMetadataPaths = append(intermediateMetadataPaths, ctx.ModuleProvider(module, testRunProviderKey).(testRunProviderData).IntermediatePath)
	})
	// Use intermediateMetadataPaths to pass file paths to the validation tool.
}
