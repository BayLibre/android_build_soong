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

package android

import (
	"sync"

	"github.com/google/blueprint"
)

type licensesDependencyTag struct {
	blueprint.BaseDependencyTag
}

var (
	licensesTag = licensesDependencyTag{}
)

// Describes the property provided by a module to reference applicable licenses.
type applicableLicensesPropertyImpl struct {
	name             string
	licensesProperty *[]string
}

type applicableLicensesProperty interface {
	getName() string
	getLicenses() []string
}

func newApplicableLicensesProperty(name string, licensesProperty *[]string) applicableLicensesProperty {
	return applicableLicensesPropertyImpl{
		name: name,
		licensesProperty: licensesProperty,
	}
}

func (p applicableLicensesPropertyImpl) getName() string {
	return p.name
}

func (p applicableLicensesPropertyImpl) getLicenses() []string {
	return *p.licensesProperty
}

type licensesContainer struct {
	licenses []string
}

func newLicensesSet(s map[string]bool) licensesContainer {
	var r licensesContainer
	r.licenses = make([]string,0,len(s))
	for l := range s {
		r.licenses = append(r.licenses, l)
	}
	return r
}

func (r licensesContainer) getLicenses() []string {
	return r.licenses
}

var packageDefaultLicensesMap = NewOnceKey("packageDefaultLicensesMap")

// The map from package dir name to default applicable licenses as a licensesContainer.
func moduleToPackageDefaultLicensesMap(config Config) *sync.Map {
	return config.Once(packageDefaultLicensesMap, func() interface{} {
		return &sync.Map{}
	}).(*sync.Map)
}

// Registers the function that maps each package to its default_applicable_licenses.
//
// This goes before defaults expansion so the defaults can pick up the package default.
func RegisterLicensesPackageMapper(ctx RegisterMutatorsContext) {
	ctx.BottomUp("licensesPackageMapper", licensesPackageMapper).Parallel()
}

// Registers the function that gathers the license dependencies for each module.
//
// This goes after defaults expansion so that it can pick up default licenses and before visibility enforcement.
func RegisterLicensesPropertyGatherer(ctx RegisterMutatorsContext) {
	ctx.BottomUp("licensesPropertyGatherer", licensesPropertyGatherer).Parallel()
}

// Registers the function that verifies the license and license_kind dependency types for each module.
func RegisterLicensesDependencyChecker(ctx RegisterMutatorsContext) {
	ctx.BottomUp("licensesPropertyChecker", licensesDependencyChecker).Parallel()
}

// Maps each package to its default applicable licenses.
func licensesPackageMapper(ctx BottomUpMutatorContext) {
	p, ok := ctx.Module().(*packageModule)
	if !ok {
		return
	}

	licenses := getLicenses(ctx, p)

	dir := ctx.ModuleDir()
	c := makeLicensesContainer(licenses)
	moduleToPackageDefaultLicensesMap(ctx.Config()).Store(dir, c)
}

func makeLicensesContainer(propVals []string) licensesContainer {
	licenses := make([]string, 0, len(propVals))
	licenses = append(licenses, propVals...)

	return licensesContainer{licenses}
}

// Gathers the flattened applicable licenses references after defaults expansion.
func licensesPropertyGatherer(ctx BottomUpMutatorContext) {
	m, ok := ctx.Module().(Module)
	if !ok {
		return
	}

	if exemptFromRequiredApplicableLicensesProperty(m) {
		return
	}

	licenses := getLicenses(ctx, m)

	ctx.AddVariationDependencies(nil, licensesTag, licenses...)
}

// Verifies the license and license_kind dependencies are the correct kind of module.
func licensesDependencyChecker(ctx BottomUpMutatorContext) {
	m, ok := ctx.Module().(Module)
	if !ok {
		return
	}

	// license modules have no licenses, but license_kinds must refer to license_kind modules
	if _, ok := m.(*licenseModule); ok {
		for _, module := range ctx.GetDirectDepsWithTag(licenseKindTag) {
			if _, ok := module.(*licenseKindModule); !ok {
				ctx.ModuleErrorf("license_kind property %q is not a license_kind module", ctx.OtherModuleName(module))
			}
		}
		return
	}

	if exemptFromRequiredApplicableLicensesProperty(m) {
		return
	}

	for _, module := range ctx.GetDirectDepsWithTag(licensesTag) {
		if _, ok := module.(*licenseModule); !ok {
			ctx.ModuleErrorf("applicable licenses property %q is not a license module", ctx.OtherModuleName(module))
		}
	}
}

// Flattens license and license_kind dependencies into calculated properties.
//
// Validates applicable licenses properties refer only to license modules and license_kinds properties refer
// only to license_kind modules.
func licensesPropertyFlattener(ctx ModuleContext) {
	m, ok := ctx.Module().(Module)
	if !ok {
		return
	}

	// license modules have no licenses, but license_kinds must refer to license_kind modules
	if l, ok := m.(*licenseModule); ok {
		mergeProps(&m.base().commonProperties.Effective_licenses, ctx.ModuleName())
		mergeProps(&m.base().commonProperties.Effective_license_text, PathsForModuleSrc(ctx, l.properties.License_text).Strings()...)
		for _, module := range ctx.GetDirectDepsWithTag(licenseKindTag) {
			if lk, ok := module.(*licenseKindModule); ok {
				mergeProps(&m.base().commonProperties.Effective_license_conditions, lk.properties.Conditions...)
				mergeProps(&m.base().commonProperties.Effective_license_kinds, ctx.OtherModuleName(module))
			} else {
				ctx.ModuleErrorf("license_kind property %q is not a license_kind module", ctx.OtherModuleName(module))
			}
		}
		return
	}

	if exemptFromRequiredApplicableLicensesProperty(m) {
		return
	}

	for _, module := range ctx.GetDirectDepsWithTag(licensesTag) {
		if l, ok := module.(*licenseModule); ok {
			if m.base().commonProperties.Effective_package_name == nil && l.properties.Package_name != nil {
				m.base().commonProperties.Effective_package_name = l.properties.Package_name
			}
			mergeProps(&m.base().commonProperties.Effective_licenses, module.base().commonProperties.Effective_licenses...)
			mergeProps(&m.base().commonProperties.Effective_license_text, module.base().commonProperties.Effective_license_text...)
			mergeProps(&m.base().commonProperties.Effective_license_kinds, module.base().commonProperties.Effective_license_kinds...)
			mergeProps(&m.base().commonProperties.Effective_license_conditions, module.base().commonProperties.Effective_license_conditions...)
		} else {
			ctx.ModuleErrorf("applicable licenses property %q is not a license module", ctx.OtherModuleName(module))
		}
	}
}

func mergeProps(prop *[]string, values ...string) {
	s := make(map[string]bool)
	for _, v := range *prop {
		s[v] = true
	}
	for _, v := range values {
		s[v] = true
	}
	*prop = []string{}
	for v := range s {
		*prop = append(*prop, v)
	}
}

// Get the licenses property falling back to the package default.
func getLicenses(ctx BaseModuleContext, module Module) []string {
	if exemptFromRequiredApplicableLicensesProperty(module) {
		return nil
	}

	primaryProperty := module.base().primaryLicensesProperty
	if primaryProperty == nil {
		ctx.ModuleErrorf("module type %q must have an applicable licenses property", ctx.OtherModuleType(module))
		return nil
	}

	licenses := primaryProperty.getLicenses()
	if len(licenses) > 0 {
		s := make(map[string]bool)
		for _, l := range licenses {
			if _, ok := s[l]; ok {
				ctx.ModuleErrorf("duplicate %q licenses", l)
			}
			s[l] = true
		}
		return licenses
	}

	dir := ctx.OtherModuleDir(module)

	moduleToApplicableLicenses := moduleToPackageDefaultLicensesMap(ctx.Config())
	value, ok := moduleToApplicableLicenses.Load(dir)
	var c licensesContainer
	if ok {
		c = value.(licensesContainer)
	} else {
		c = licensesContainer{}
	}
	return c.getLicenses()
}

// Get the effective licenses, i.e. both the direct license references and indirect references via deps
func effectiveApplicableLicenses(ctx BaseModuleContext, module Module) []string {
	if exemptFromRequiredApplicableLicensesProperty(module) {
		return nil
	}

	s := make(map[string]bool)
	for _, l := range getLicenses(ctx, module) {
		s[l] = true
	}

	ctx.WalkDeps(func(child, parent Module) bool {
		if _, ok := child.(*licenseModule); ok {
			return false
		}
		if _, ok := child.(*licenseKindModule); ok {
			return false
		}
		for _, l := range getLicenses(ctx, child) {
			s[l] = true
		}
		return true
	})

	return newLicensesSet(s).getLicenses()
}

func exemptFromRequiredApplicableLicensesProperty(module Module) bool {
	// neither license nor license_kind have licenses -- they are licenses
	if _, ok := module.(*licenseModule); ok {
		return true
	}
	if _, ok := module.(*licenseKindModule); ok {
		return true
	}

	// soong_namespace needs no applicable license -- it just partitions things
	if _, ok := module.(*NamespaceModule); ok {
		return true
	}

	// soong_config_module_type and associates create aliases for modules with licenses so don't need licenses themselves
	if _, ok := module.(*soongConfigModuleTypeModule); ok {
		return true
	}
	if _, ok := module.(*soongConfigModuleTypeImport); ok {
		return true
	}
	if _, ok := module.(*soongConfigStringVariableDummyModule); ok {
		return true
	}

	return false
}

// Set the primary applicable licenses property.
func setPrimaryLicensesProperty(module Module, name string, licensesProperty *[]string) {
	module.base().primaryLicensesProperty = newApplicableLicensesProperty(name, licensesProperty)
}
