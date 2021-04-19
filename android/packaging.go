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
	"path/filepath"

	"github.com/google/blueprint"
)

// PackagingSpec abstracts a request to place a built artifact at a certain path in a package. A
// package can be the traditional <partition>.img, but isn't limited to those. Other examples could
// be a new filesystem image that is a subset of system.img (e.g. for an Android-like mini OS
// running on a VM), or a zip archive for some of the host tools.
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

// Get file name of installed package
func (p *PackagingSpec) FileName() string {
	if p.relPathInPackage != "" {
		return filepath.Base(p.relPathInPackage)
	}

	return ""
}

// Path relative to the root of the package
func (p *PackagingSpec) RelPathInPackage() string {
	return p.relPathInPackage
}

type PackageModule interface {
	Module
	packagingBase() *PackagingBase

	// AddDeps adds modules in the `deps` property and those returned from `GatherExtraDeps` as
	// dependencies so that they are packaged into this module. This should be called in
	// DepsMutator. When installTogether is set to true, the dependencies are also separately
	// installed when this module is installed.
	AddDeps(ctx BottomUpMutatorContext, installTogether bool)

	// ExtraDeps returns extra dependencies that need to be packaged but are from properties
	// other than the `deps` property.
	GatherExtraDeps(ctx BaseModuleContext) []PackageDep

	// CopyDepsToZip zips the built artifacts of the dependencies into the given zip file and
	// returns zip entries in it. This is expected to be called in GenerateAndroidBuildActions,
	// followed by a build rule that unzips it and creates the final output (img, zip, tar.gz,
	// etc.) from the extracted files
	CopyDepsToZip(ctx ModuleContext, zipOut WritablePath) []string
}

// PackagingBase provides basic functionality for packaging dependencies. A module is expected to
// include this struct and call InitPackageModule.
type PackagingBase struct {
	properties PackagingProperties

	// Allows this module to skip missing dependencies. In most cases, this is not required, but
	// for rare cases like when there's a dependency to a module which exists in certain repo
	// checkouts, this is needed.
	IgnoreMissingDependencies bool
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

type packagingArchProperties struct {
	Arm64  depsProperty
	Arm    depsProperty
	X86_64 depsProperty
	X86    depsProperty
}

type PackagingProperties struct {
	Deps     []string                    `android:"arch_variant"`
	Multilib packagingMultilibProperties `android:"arch_variant"`
	Arch     packagingArchProperties
}

func InitPackageModule(p PackageModule) {
	base := p.packagingBase()
	p.AddProperties(&base.properties)
}

func (p *PackagingBase) packagingBase() *PackagingBase {
	return p
}

// From deps and multilib.*.deps, select the dependencies that are for the given arch deps is for
// the current archicture when this module is not configured for multi target. When configured for
// multi target, deps is selected for each of the targets and is NOT selected for the current
// architecture which would be Common.
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

	if ctx.Arch().ArchType == Common {
		switch arch {
		case Arm64:
			ret = append(ret, p.properties.Arch.Arm64.Deps...)
		case Arm:
			ret = append(ret, p.properties.Arch.Arm.Deps...)
		case X86_64:
			ret = append(ret, p.properties.Arch.X86_64.Deps...)
		case X86:
			ret = append(ret, p.properties.Arch.X86.Deps...)
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

type packagingDependencyTag struct {
	blueprint.BaseDependencyTag
	installTogether bool
}

var _ InstallNeededDependencyTag = packagingDependencyTag{}

func (tag packagingDependencyTag) InstallDepNeeded() bool {
	return tag.installTogether
}

// See PackageModule.AddDeps
func (p *PackagingBase) AddDeps(ctx BottomUpMutatorContext, installTogether bool) {
	depTag := packagingDependencyTag{installTogether: installTogether}
	for _, d := range p.gatherDeps(ctx) {
		ctx.AddFarVariationDependencies(d.Target.Variations(), depTag, d.Name)
	}
}

// PackageDep represents a dependency from a PackageModule that can be passed to PackageAggregator
// modules.
type PackageDep struct {
	Name   string // name of the dependency
	Target        // android.Target for which the dependency is added
}

// See PackageModule.GatherExtraDeps
func (p *PackagingBase) GatherExtraDeps(ctx BaseModuleContext) []PackageDep {
	return nil
}

func (p *PackagingBase) gatherDeps(ctx BaseModuleContext) []PackageDep {
	var deps []PackageDep
	for _, t := range p.getSupportedTargets(ctx) {
		for _, dep := range p.getDepsForArch(ctx, t.Arch.ArchType) {
			if p.IgnoreMissingDependencies && !ctx.OtherModuleExists(dep) {
				continue
			}
			deps = append(deps, PackageDep{Name: dep, Target: t})
		}
	}
	if pm, ok := ctx.Module().(PackageModule); ok {
		deps = append(deps, pm.GatherExtraDeps(ctx)...)
	}
	return deps
}

// See PackageModule.CopyDepsToZip
func (p *PackagingBase) CopyDepsToZip(ctx ModuleContext, zipOut WritablePath) (entries []string) {
	m := make(map[string]PackagingSpec)
	ctx.VisitDirectDeps(func(child Module) {
		if _, ok := ctx.OtherModuleDependencyTag(child).(packagingDependencyTag); !ok {
			return
		}
		for _, ps := range child.TransitivePackagingSpecs() {
			if _, ok := m[ps.relPathInPackage]; !ok {
				m[ps.relPathInPackage] = ps
			}
		}
	})

	builder := NewRuleBuilder(pctx, ctx)

	dir := PathForModuleOut(ctx, ".zip")
	builder.Command().Text("rm").Flag("-rf").Text(dir.String())
	builder.Command().Text("mkdir").Flag("-p").Text(dir.String())

	seenDir := make(map[string]bool)
	for _, k := range SortedStringKeys(m) {
		ps := m[k]
		destPath := dir.Join(ctx, ps.relPathInPackage).String()
		destDir := filepath.Dir(destPath)
		entries = append(entries, ps.relPathInPackage)
		if _, ok := seenDir[destDir]; !ok {
			seenDir[destDir] = true
			builder.Command().Text("mkdir").Flag("-p").Text(destDir)
		}
		if ps.symlinkTarget == "" {
			builder.Command().Text("cp").Input(ps.srcPath).Text(destPath)
		} else {
			builder.Command().Text("ln").Flag("-sf").Text(ps.symlinkTarget).Text(destPath)
		}
		if ps.executable {
			builder.Command().Text("chmod").Flag("a+x").Text(destPath)
		}
	}

	builder.Command().
		BuiltTool("soong_zip").
		FlagWithOutput("-o ", zipOut).
		FlagWithArg("-C ", dir.String()).
		Flag("-L 0"). // no compression because this will be unzipped soon
		FlagWithArg("-D ", dir.String())
	builder.Command().Text("rm").Flag("-rf").Text(dir.String())

	builder.Build("zip_deps", fmt.Sprintf("Zipping deps for %s", ctx.ModuleName()))
	return entries
}

// packagingSpecsDepSet is a thin type-safe wrapper around the generic depSet.  It always uses
// topological order.
type packagingSpecsDepSet struct {
	depSet
}

// newPackagingSpecsDepSet returns an immutable packagingSpecsDepSet with the given direct and
// transitive contents.
func newPackagingSpecsDepSet(direct []PackagingSpec, transitive []*packagingSpecsDepSet) *packagingSpecsDepSet {
	return &packagingSpecsDepSet{*newDepSet(TOPOLOGICAL, direct, transitive)}
}

// ToList returns the packagingSpecsDepSet flattened to a list in topological order.
func (d *packagingSpecsDepSet) ToList() []PackagingSpec {
	if d == nil {
		return nil
	}
	return d.depSet.ToList().([]PackagingSpec)
}

// PackageAggregator is an interface that a module can implement to depend on other modules in the
// same package.
type PackageAggregator interface {
	// AggregatePackageDeps passes the dependency list of a PackageModule down to this module
	// implementing PackageAggregator interface. Each dependency comes with the target that it
	// is added for. The dependency list doesn't include PackageAggregator modules, because
	// otherwise there will be circular dependency among them.
	AggregatePackageDeps(deps []PackageDep)

	// PackageDepsMutator may add dependencies to the dependency list passed via the prior call
	// to AggregatePackageDeps.
	PackageDepsMutator(mctx BottomUpMutatorContext)
}

var PrepareForTestWithPackagingComponents = FixtureRegisterWithContext(registerPackagingComponents)

func init() {
	registerPackagingComponents(InitRegistrationContext)
}

func registerPackagingComponents(ctx RegistrationContext) {
	ctx.PostDepsMutators(registerPostDepsMutators)
}

func registerPostDepsMutators(ctx RegisterMutatorsContext) {
	ctx.TopDown("package_deps_gather", packageAggregationMutator).Parallel()
	ctx.BottomUp("package_deps", packageDepsMutator).Parallel()
}

func packageAggregationMutator(mctx TopDownMutatorContext) {
	p, ok := mctx.Module().(PackageModule)
	if !ok {
		return
	}
	pb := p.packagingBase()

	// Gather aggregators so that we can filter them out from deps. Otherwise, there will be
	// circular dependencies among aggregators.
	var aggregators []string
	mctx.VisitDirectDeps(func(child Module) {
		if _, ok := child.(PackageAggregator); ok {
			name := mctx.OtherModuleName(child)
			aggregators = append(aggregators, name)
		}
	})

	// Gather deps
	var deps []PackageDep
	for _, d := range pb.gatherDeps(mctx) {
		if InList(d.Name, aggregators) {
			continue
		}
		deps = append(deps, d)
	}

	// Pass deps down to aggregators if any
	mctx.VisitDirectDeps(func(child Module) {
		if pa, ok := child.(PackageAggregator); ok {
			pa.AggregatePackageDeps(deps)
		}
	})
}

func packageDepsMutator(mctx BottomUpMutatorContext) {
	if p, ok := mctx.Module().(PackageAggregator); ok {
		p.PackageDepsMutator(mctx)
	}
}
