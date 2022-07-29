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

// ImageInterface is implemented by modules that need to be split by the imageMutator.
type ImageInterface interface {
	// ImageMutatorBegin is called before any other method in the ImageInterface.
	ImageMutatorBegin(ctx BaseModuleContext)

	// CoreVariantNeeded should return true if the module needs a core variant (installed on the system image).
	CoreVariantNeeded(ctx BaseModuleContext) bool

	// RamdiskVariantNeeded should return true if the module needs a ramdisk variant (installed on the
	// ramdisk partition).
	RamdiskVariantNeeded(ctx BaseModuleContext) bool

	// VendorRamdiskVariantNeeded should return true if the module needs a vendor ramdisk variant (installed on the
	// vendor ramdisk partition).
	VendorRamdiskVariantNeeded(ctx BaseModuleContext) bool

	// DebugRamdiskVariantNeeded should return true if the module needs a debug ramdisk variant (installed on the
	// debug ramdisk partition: $(PRODUCT_OUT)/debug_ramdisk).
	DebugRamdiskVariantNeeded(ctx BaseModuleContext) bool

	// RecoveryVariantNeeded should return true if the module needs a recovery variant (installed on the
	// recovery partition).
	RecoveryVariantNeeded(ctx BaseModuleContext) bool

	// ExtraImageVariations should return a list of the additional variations needed for the module.  After the
	// variants are created the SetImageVariation method will be called on each newly created variant with the
	// its variation.
	ExtraImageVariations(ctx BaseModuleContext) []string

	// SetImageVariation is called for each newly created image variant. The receiver is the original
	// module, "variation" is the name of the newly created variant and "module" is the newly created
	// variant itself.
	SetImageVariation(ctx BaseModuleContext, variation string, module Module)
}

const (
	// CoreVariation is the variant used for framework-private libraries, or
	// SDK libraries. (which framework-private libraries can use), which
	// will be installed to the system image.
	CoreVariation string = ""

	// RecoveryVariation means a module to be installed to recovery image.
	RecoveryVariation string = "recovery"

	// RamdiskVariation means a module to be installed to ramdisk image.
	RamdiskVariation string = "ramdisk"

	// VendorRamdiskVariation means a module to be installed to vendor ramdisk image.
	VendorRamdiskVariation string = "vendor_ramdisk"

	// DebugRamdiskVariation means a module to be installed to debug ramdisk image.
	DebugRamdiskVariation string = "debug_ramdisk"
)

// imageMutator creates variants for modules that implement the ImageInterface that
// allow them to build differently for each partition (recovery, core, vendor, etc.).
func imageMutator(ctx BottomUpMutatorContext) {
	if ctx.Os() != Android {
		return
	}

	if original, ok := ctx.Module().(ImageInterface); ok {
		original.ImageMutatorBegin(ctx)

		helper := VariantCreationHelper{}
		callback := func(variation string, apiDomain string) func(variant Module) {
			return func(variant Module) {
				variant.base().setImageVariation(variation)
				original.SetImageVariation(ctx, variation, variant)
				variant.SetApiDomain(apiDomain)
			}
		}

		// Common Variations
		if original.CoreVariantNeeded(ctx) {
			helper.Add(CoreVariation, callback(CoreVariation, "partition:system"))
		}
		if original.RamdiskVariantNeeded(ctx) {
			helper.Add(RamdiskVariation, callback(RamdiskVariation, "partition:vendor"))
		}
		if original.VendorRamdiskVariantNeeded(ctx) {
			helper.Add(VendorRamdiskVariation, callback(VendorRamdiskVariation, "partition:vendor"))
		}
		if original.DebugRamdiskVariantNeeded(ctx) {
			helper.Add(DebugRamdiskVariation, callback(DebugRamdiskVariation, "partition:vendor"))
		}
		if original.RecoveryVariantNeeded(ctx) {
			helper.Add(RecoveryVariation, callback(RecoveryVariation, "partition:recovery"))
		}
		// TODO: IsProductSpecific?

		// Extra variations
		for _, variation := range original.ExtraImageVariations(ctx) {
			helper.Add(variation, callback(variation, "partition:vendor"))
		}

		helper.Create(ctx)
	}
}


