// Copyright 2019 Google Inc. All rights reserved.
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

type ImageInterface interface {
	ImageMutatorBegin(BaseModuleContext)

	CoreVariantNeeded(BaseModuleContext) bool
	RecoveryVariantNeeded(BaseModuleContext) bool
	ExtraImageVariants(BaseModuleContext) []string

	SetRecoveryVariant(BaseModuleContext, Module)
	SetExtraImageVariants(BaseModuleContext, []string, []Module)
}

const (
	// coreMode is the variant used for framework-private libraries, or
	// SDK libraries. (which framework-private libraries can use), which
	// will be installed to the system image.
	coreMode = "core"

	// recoveryMode means a module to be installed to recovery image.
	recoveryMode = "recovery"
)

func ImageMutator(ctx BottomUpMutatorContext) {
	if ctx.Os() != Android {
		return
	}

	if m, ok := ctx.Module().(ImageInterface); ok {
		m.ImageMutatorBegin(ctx)

		var variants []string

		if m.CoreVariantNeeded(ctx) {
			variants = append(variants, coreMode)
		}
		if m.RecoveryVariantNeeded(ctx) {
			variants = append(variants, recoveryMode)
		}

		extraVariants := m.ExtraImageVariants(ctx)
		extraVariantsOffset := len(variants)
		variants = append(variants, extraVariants...)

		if len(variants) == 0 {
			return
		}

		mod := ctx.CreateVariations(variants...)
		for i, v := range variants {
			if v == recoveryMode {
				m.SetRecoveryVariant(ctx, mod[i])
			}
		}

		if len(extraVariants) > 0 {
			m.SetExtraImageVariants(ctx, extraVariants, mod[extraVariantsOffset:])
		}
	}
}
