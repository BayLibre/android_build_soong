// Copyright 2020 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License")
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

	"github.com/google/blueprint"
)

// PackagingSpec abstracts a request to place a built artifact at a certain path in a package.
// A package can be the traditional <partition>.img, but isn't limited to those. Other examples could
// be a new filesystem image that is a subset of system.img (e.g. for an Android-like mini OS running
// on a VM), or a zip archive for some of the host tools.
type PackagingSpec struct {
	// Path relative to the root of the package
	relPathInPackage string

	// The path to the built artifact
	srcPath Path

	// If this is not empty, then relPathInPackage should be a symlink to this target. (Then
	// srcPath is of course ignored.)
	symlinkTarget string

	// Whether relPathInPackage should be marked as executable or not
	executable bool
}

type PackageModule interface {
	Module
	packagingBase() *PackagingBase

	// SetIgnoreMissingDependencies allows this module to skip missing dependencies. In most
	// cases, this is not required, but for rare cases like when there's a dependency to
	// a module which exists in certain repo checkouts, this is needed.
	SetIgnoreMissingDependencies()

	// AddDeps adds dependencies to the `deps` modules. This should be called in DepsMutator.
	AddDeps(ctx BottomUpMutatorContext)

	// CopyDepsToDir creates rules to copy the built artifacts of the dependencies to the
	// corresponding paths under `dir`. This is expected to be called in GenerateAndroidBuildActions,
	// followed by a build rule that creates the final output (img, zip, tar.gz, etc.) from
	// the copied files
	CopyDepsToDir(ctx ModuleContext, dir OutputPath) WritablePaths
}

// PackagingBase provides basic functionality for packaging dependencies. A module is expected to
// include this struct and call InitPackageModule.
type PackagingBase struct {
	properties PackagingProperties
}

type PackagingProperties struct {
	// Modules to include in this package
	Deps []string `android:"arch_variant"`

	IgnoreMissingDependencies bool `blueprint:"mutated"`
}

type packagingDependencyTag struct{ blueprint.BaseDependencyTag }

var depTag = packagingDependencyTag{}

func InitPackageModule(p PackageModule) {
	base := p.packagingBase()
	p.AddProperties(&base.properties)
}

func (p *PackagingBase) packagingBase() *PackagingBase {
	return p
}

func (p *PackagingBase) SetIgnoreMissingDependencies() {
	p.properties.IgnoreMissingDependencies = true
}

func (p *PackagingBase) AddDeps(ctx BottomUpMutatorContext) {
	for _, dep := range p.properties.Deps {
		if p.properties.IgnoreMissingDependencies && !ctx.OtherModuleExists(dep) {
			continue
		}
		if ctx.OtherModuleDependencyVariantExists(nil, dep) {
			ctx.AddVariationDependencies(nil, depTag, dep)
		} else {
			// If the matching arch variant is not found, try the common arch as a fallback
			variations := []blueprint.Variation{
				{Mutator: "os", Variation: ctx.Target().Os.String()},
				{Mutator: "arch", Variation: Common.String()},
			}
			ctx.AddFarVariationDependencies(variations, depTag, dep)
		}
	}
}

func (p *PackagingBase) CopyDepsToDir(ctx ModuleContext, dir OutputPath) WritablePaths {
	m := make(map[string]PackagingSpec)
	ctx.WalkDeps(func(child Module, parent Module) bool {
		// different arch is not included
		myArch := ctx.Arch().ArchType
		childArch := child.Target().Arch.ArchType
		if childArch != Common && childArch != myArch {
			return false
		}
		for _, ps := range child.PackagingSpecs() {
			if _, ok := m[ps.relPathInPackage]; !ok {
				m[ps.relPathInPackage] = ps
			}
		}
		return true
	})

	builder := NewRuleBuilder()
	builder.Command().Text("mkdir").Flag("-p").Text(dir.String())

	var ret WritablePaths
	for _, k := range SortedStringKeys(m) {
		ps := m[k]
		output := dir.Join(ctx, ps.relPathInPackage)
		ret = append(ret, output)
		if ps.symlinkTarget == "" {
			builder.Command().Text("cp").Input(ps.srcPath).Output(output)
		} else {
			builder.Command().Text("ln").Flag("-sf").Text(ps.symlinkTarget).Output(output)
		}
		if ps.executable {
			builder.Command().Text("chmod").Flag("a+x").Output(output)
		}
	}
	builder.Build(pctx, ctx, "copy_deps", fmt.Sprintf("Copying deps for %s", ctx.ModuleName()))
	return ret
}
