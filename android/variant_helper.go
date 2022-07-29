// Copyright 2022 Google Inc. All rights reserved.
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

// Helper to create variants of modules and perform actions on the newly created variants
type VariantCreationHelper struct {
	variations []string
	actions []func(module Module)
}

// CreateVariation schedules creation of the variation (which will be created when Create is
// called). After the variant of the module is created, action will be called on the variant.
func (vch *VariantCreationHelper) Add(variation string, action func(module Module)) {
	vch.variations = append(vch.variations, variation)
	vch.actions = append(vch.actions, action)
}

func (vch *VariantCreationHelper) Create(ctx BottomUpMutatorContext) {
	if len(vch.variations) > 0 {
		variants := ctx.CreateVariations(vch.variations...)
		for i, variant := range variants {
			vch.actions[i](variant)
		}
	}
}


