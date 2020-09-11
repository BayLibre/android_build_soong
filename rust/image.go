// Copyright 2020 The Android Open Source Project
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

package rust

import (
	"strings"

	"android/soong/android"
	"android/soong/cc"
)

var _ android.ImageInterface = (*Module)(nil)

func (mod *Module) CoreVariantNeeded(ctx android.BaseModuleContext) bool {
	return mod.Properties.CoreVariantNeeded
}

func (mod *Module) RamdiskVariantNeeded(android.BaseModuleContext) bool {
	return mod.InRamdisk()
}

func (mod *Module) RecoveryVariantNeeded(android.BaseModuleContext) bool {
	return mod.InRecovery()
}

func (mod *Module) ExtraImageVariations(android.BaseModuleContext) []string {
	return mod.Properties.ExtraVariants
}

func (ctx *moduleContext) ProductSpecific() bool {
	return ctx.ModuleContext.ProductSpecific() ||
		(ctx.RustModule().HasVendorVariant() && ctx.RustModule().inProduct() && !ctx.RustModule().IsVndk())
}

func (mod *Module) InRecovery() bool {
	// TODO(b/165791368)
	return false
}

func (mod *Module) OnlyInRamdisk() bool {
	// TODO(b/165791368)
	return false
}

func (mod *Module) OnlyInRecovery() bool {
	// TODO(b/165791368)
	return false
}

func (mod *Module) HasVendorVariant() bool {
	// TODO(b/165791368)
	return false
}

// Returns true if the module is "product" variant. Usually these modules are installed in /product
func (mod *Module) inProduct() bool {
	return mod.Properties.ImageVariationPrefix == cc.ProductVariationPrefix
}

func (mod *Module) SetImageVariation(ctx android.BaseModuleContext, variant string, module android.Module) {
	m := module.(*Module)
	if strings.HasPrefix(variant, cc.ProductVariationPrefix) {
		m.Properties.ImageVariationPrefix = cc.ProductVariationPrefix
		m.Properties.VndkVersion = strings.TrimPrefix(variant, cc.ProductVariationPrefix)
	}
}

func (mod *Module) ImageMutatorBegin(mctx android.BaseModuleContext) {
	productSpecific := mctx.ProductSpecific()

	var coreVariantNeeded bool = false
	var productVariants []string

	platformVndkVersion := mctx.DeviceConfig().PlatformVndkVersion()
	boardVndkVersion := mctx.DeviceConfig().VndkVersion()
	productVndkVersion := mctx.DeviceConfig().ProductVndkVersion()
	if boardVndkVersion == "current" {
		boardVndkVersion = platformVndkVersion
	}

	if productVndkVersion == "current" {
		productVndkVersion = platformVndkVersion
	}

	// TODO(b/165791368): For now we create a core variant by default.
	coreVariantNeeded = true

	if boardVndkVersion != "" && productVndkVersion != "" {
		if coreVariantNeeded && productSpecific && mod.SdkVersion() == "" {
			// The module has "product_specific: true" that does not create core variant.
			coreVariantNeeded = false
			productVariants = append(productVariants, productVndkVersion)
		}
	} else {
		// Unless PRODUCT_PRODUCT_VNDK_VERSION is set, product partition has no
		// restriction to use system libs.
		// No product variants defined in this case.
		productVariants = []string{}
	}

	for _, variant := range android.FirstUniqueStrings(productVariants) {
		mod.Properties.ExtraVariants = append(mod.Properties.ExtraVariants, cc.ProductVariationPrefix+variant)
	}

	mod.Properties.CoreVariantNeeded = coreVariantNeeded
}
