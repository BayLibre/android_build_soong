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
)

// The Bazel QueryView singleton is responsible for generating the Ninja actions
// for calling the soong_build primary builder in the main build.ninja file.
func init() {
	android.RegisterPreSingletonType("bazel_queryview_generation", BazelQueryViewGeneratorSingleton)
}

func BazelQueryViewGeneratorSingleton() android.Singleton {
	return &bazelQueryViewGeneratorSingleton{}
}

type targetInfo struct {
	name       string
	moduleType string
}

type bazelQueryViewGeneratorSingleton struct {
	buildToTargets map[string][]bazelTarget
}

type BazelAttributes struct {
	Attrs map[string]string
}

type bazelTarget struct {
	target targetInfo
	attrs  BazelAttributes
}

type bazelAttributesGenerator interface {
	GenerateBazelAttributes() BazelAttributes
}

func (c *bazelQueryViewGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// TODO: get packages -- could almost reuse code from soong/writedocs -- singletoncontext implements most of the functions required
	// except for ctx.ModuleTypeFactories
	// moduleTypeFactories := android.ModuleTypeFactories()
	// bpModuleTypeFactories := make(map[string]reflect.Value)
	// for moduleType, factory := range moduleTypeFactories {
	// bpModuleTypeFactories[moduleType] = reflect.ValueOf(factory)
	// }
	// packages, err := bootstrap.ModuleTypeDocs(ctx.Context, bpModuleTypeFactories)

	c.buildToTargets = make(map[string][]bazelTarget)
	ctx.VisitAllModules(func(m android.Module) {
		if qv, ok := m.(bazelAttributesGenerator); ok {
			qv.GenerateBazelAttributes()
		} else {
			targetInfo := targetInfo{
				name:       ctx.ModuleName(m),
				moduleType: ctx.ModuleType(m),
			}
			allProps := ExtractModuleProperties(m)
			dir := ctx.ModuleDir(m)
			c.buildToTargets[dir] = append(c.buildToTargets[dir], bazelTarget{
				target: targetInfo,
				attrs: BazelAttributes{
					Attrs: allProps,
				},
			})
		}
	})
}
