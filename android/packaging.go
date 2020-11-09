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

type depsProperty struct {
	// Modules to include in this package
	Deps []string `android:"arch_variant"`
}

type packagingMultilibProperties struct {
	First  depsProperty `android:"arch_variant"`
	Common depsProperty `android:"arch_variant"`
	Lib32  depsProperty `android:"arch_variant"`
	Lib64  depsProperty `android:"arch_variant"`
}

type PackagingProperties struct {
	Deps     []string                    `android:"arch_variant"`
	Multilib packagingMultilibProperties `android:"arch_variant"`

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

// From deps and multilib.*.deps, select the dependencies that are for the given arch
// deps is for the current archicture when this module is not configured for multi target.
// When configured for multi target, deps is selected for each of the targets and is NOT
// selected for the current architecture which would be Common.
func (p *PackagingBase) getDepsForArch(ctx BaseModuleContext, arch ArchType) []string {
	var ret []string
	if arch == ctx.Target().Arch.ArchType && len(ctx.MultiTargets()) == 0 {
		ret = append(ret, p.properties.Deps...)
	} else if arch.Multilib == "lib32" {
		ret = append(ret, p.properties.Multilib.Lib32.Deps...)
	} else if arch.Multilib == "lib64" {
		ret = append(ret, p.properties.Multilib.Lib64.Deps...)
	} else if arch == Common {
		ret = append(ret, p.properties.Multilib.Common.Deps...)
	}
	for i, t := range ctx.MultiTargets() {
		if t.Arch.ArchType == arch {
			ret = append(ret, p.properties.Deps...)
			if i == 0 {
				ret = append(ret, p.properties.Multilib.First.Deps...)
			}
		}
	}
	return FirstUniqueStrings(ret)
}

func (p *PackagingBase) getSupportedTargets(ctx BaseModuleContext) []Target {
	var ret []Target
	// The current and the common OS targets are always supported
	ret = append(ret, ctx.Target())
	if ctx.Arch().ArchType != Common {
		ret = append(ret, Target{Os: ctx.Os(), Arch: Arch{ArchType: Common}})
	}
	// If this module is configured for multi targets, those should be supported as well
	ret = append(ret, ctx.MultiTargets()...)
	return ret
}

func (p *PackagingBase) AddDeps(ctx BottomUpMutatorContext) {
	for _, t := range p.getSupportedTargets(ctx) {
		for _, dep := range p.getDepsForArch(ctx, t.Arch.ArchType) {
			if p.properties.IgnoreMissingDependencies && !ctx.OtherModuleExists(dep) {
				continue
			}
			ctx.AddFarVariationDependencies(t.Variations(), depTag, dep)
		}
	}
}

func (p *PackagingBase) CopyDepsToDir(ctx ModuleContext, dir OutputPath) WritablePaths {
	var supportedArches []string
	for _, t := range p.getSupportedTargets(ctx) {
		supportedArches = append(supportedArches, t.Arch.ArchType.String())
	}
	m := make(map[string]PackagingSpec)
	ctx.WalkDeps(func(child Module, parent Module) bool {
		// Don't track modules with unsupported arch
		if !InList(child.Target().Arch.ArchType.String(), supportedArches) {
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
