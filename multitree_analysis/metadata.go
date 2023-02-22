/*
 * Copyright (C) 2023 The Android Open Source Project
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
	"encoding/json"
	"fmt"
	"sort"

	"android/soong/android"

	"github.com/google/blueprint"
)

var (
	pctx = android.NewPackageContext("android/soong/multitree_apex")
)

func init() {
	RegisterModulesSingleton(android.InitRegistrationContext)
}

func RegisterModulesSingleton(ctx android.RegistrationContext) {
	ctx.RegisterSingletonType("apex_multitree_singleton", multitreeAnalysisSingletonFactory)
}

var PrepareForTestWithApexMultitreeSingleton = android.FixtureRegisterWithContext(RegisterModulesSingleton)

func multitreeAnalysisSingletonFactory() android.Singleton {
	return &multitreeAnalysisSingleton{}
}

type multitreeAnalysisSingleton struct {
	multitreeApexMetadataPath android.OutputPath
}

type ApexMultitreeMetadata struct {
	// The names of installable apexes.
	Installable []string
}

func (p *multitreeAnalysisSingleton) GenerateBuildActions(context android.SingletonContext) {
	data := ApexMultitreeMetadata{
		Installable: make([]string, 0),
	}
	context.VisitAllModulesIf(moduleFilter, func(module android.Module) {
		if p, ok := module.(android.ApexModule); ok {
			data.Installable = append(data.Installable, p.Name())
		}
	})
	p.multitreeApexMetadataPath = android.PathForOutput(context, "multitree_apex_metadata.json")
	sort.Strings(data.Installable)

	jsonStr, err := json.Marshal(data)
	if err != nil {
		context.Errorf(err.Error())
	}
	android.WriteFileRule(context, p.multitreeApexMetadataPath, string(jsonStr))
	// This seems cleaner, but doesn't emit the phony rule in testing.
	// context.Phony("multitree_apex_metadata", p.multitreeApexMetadataPath)
	// return
	context.Build(pctx, android.BuildParams{
		Rule:        blueprint.Phony,
		Description: "phony rule for multitree_apex_metadata",
		Inputs:      []android.Path{p.multitreeApexMetadataPath},
		Output:      android.PathForPhony(context, "multitree_apex_metadata"),
	})
}

func moduleFilter(module android.Module) bool {
	if !module.Enabled() || module.IsSkipInstall() {
		return false
	}
	_, ok := module.(android.ApexModule)
	return ok
}
