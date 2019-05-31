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

import (
	"fmt"
	"sync/atomic"

	"github.com/google/blueprint"
)

func init() {
	RegisterModuleType("package", PackageFactory)
}

// The information maintained about each package.
type PackageInfo struct {
	// The module from which this information was populated. If `duplicated` = true then this is the
	// module that has been renamed and must be used to report errors.
	module PackageModule

	// If true this indicates that there are two package statements in the same package which is not
	// allowed and will cause the build to fail. This flag is set by packageRenamer and checked in
	// packageInfoGatherer
	duplicated bool

	// The default_visibility property supplied to the
	default_visibility []string

	// The default visibility rules, nil if none specified.
	defaultVisibilityRule compositeRule
}

type packageProperties struct {
	Name string `blueprint:"mutated"`

	// Specifies the default visibility for all modules defined in this package.
	Default_visibility []string
}

type Package struct {
	ModuleBase

	properties packageProperties
}

type PackageModule interface {
	Module
	DefaultVisibility() []string

	setName(name string)
}

func (p *Package) GenerateAndroidBuildActions(ModuleContext) {
	// Nothing to do.
}

func (n *Package) GenerateBuildActions(ctx blueprint.ModuleContext) {
	// Nothing to do.
}

func (p *Package) DefaultVisibility() []string {
	return p.properties.Default_visibility
}

// Override to ensure that the visibility rule syntax is checked during the checking phase.
func (p *Package) visibility() []string {
	return p.DefaultVisibility()
}

func (p *Package) Name() string {
	return p.properties.Name
}

func (p *Package) setName(name string) {
	p.properties.Name = name
}

// Counter to ensure package modules are created with a unique name within whatever namespace they
// belong.
var packageCount uint32 = 0

func PackageFactory() Module {
	module := &Package{}

	// Get a unique if for the package. Has to be done atomically as the creation of the modules are
	// done in parallel.
	id := atomic.AddUint32(&packageCount, 1)
	name := fmt.Sprintf("soong_package_%d", id)

	module.properties.Name = name

	module.AddProperties(&module.properties)
	return module
}

// Registers the function that gathers the information for each package.
func registerPackageInfoGatherer(ctx RegisterMutatorsContext) {
	ctx.BottomUp("packageRenamer", packageRenamer).Parallel()
	ctx.BottomUp("packageInfoGatherer", packageInfoGatherer).Parallel()
}

// Renames the package to match the package directory.
//
// This also creates a PackageInfo object for each package and uses that to detect and remember
// duplicates for later error reporting.
func packageRenamer(ctx BottomUpMutatorContext) {
	m, ok := ctx.Module().(PackageModule)
	if !ok {
		return
	}

	packageName := "//" + ctx.ModuleDir()

	pi := newPackageInfo(ctx, packageName, m)
	if pi.module != m {
		// Remember that the package was duplicated but do not rename as that will cause an error to
		// be logged with the generated name. Similarly, reporting the error here will use the generated
		// name as renames are only processed after this phase.
		pi.duplicated = true
	} else {
		// This is the first package module in this package so rename it to match the package name.
		m.setName(packageName)
		ctx.Rename(packageName)
	}
}

// Gathers the information for each package and populates the PackageInfo structure.
//
// Also logs any deferred errors.
func packageInfoGatherer(ctx BottomUpMutatorContext) {
	m, ok := ctx.Module().(PackageModule)
	if !ok {
		return
	}

	packageDir := ctx.ModuleDir()
	packageName := "//" + packageDir

	// Get the PackageInfo for the package. Should have been populated in the packageRenamer phase.
	pi := packageInfo(ctx, packageName)
	if pi == nil {
		ctx.ModuleErrorf("internal error, expected package info to be present for package '%s'",
			packageName)
		return
	}

	if pi.module != m {
		// The package module has been duplicated but this is not the module that has been renamed so
		// ignore it. An error will be logged for the renamed module which will ensure that the error
		// message uses the correct name.
		return
	}

	// Check to see whether there are duplicate package modules in the package.
	if pi.duplicated {
		ctx.ModuleErrorf("package {...} specified multiple times")
		return
	}

	pi.default_visibility = m.DefaultVisibility()

	// Parse the visibility and store the information in the PackageInfo.
	visibility := m.DefaultVisibility()
	if visibility != nil {
		rule := parseRules(ctx, packageDir, visibility)
		if rule != nil {
			pi.defaultVisibilityRule = rule
		}
	}
}

type defaultPackageInfoKey string

func newPackageInfo(ctx BaseModuleContext, packageName string, module PackageModule) *PackageInfo {
	key := NewCustomOnceKey(defaultPackageInfoKey(packageName))

	return ctx.Config().Once(key, func() interface{} {
		return &PackageInfo{module: module}
	}).(*PackageInfo)
}

// Get the PackageInfo for the package name (starts with //, no trailing /), is nil if no package
// module type was specified.
func packageInfo(ctx BaseModuleContext, packageName string) *PackageInfo {
	key := NewCustomOnceKey(defaultPackageInfoKey(packageName))

	pi := ctx.Config().Once(key, func() interface{} {
		return nil
	})

	if pi == nil {
		return nil
	} else {
		return pi.(*PackageInfo)
	}
}
