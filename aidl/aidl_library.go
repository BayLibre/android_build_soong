// Copyright 2023 Google Inc. All rights reserved.
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

package aidl

import (
	"android/soong/android"
	"android/soong/bazel"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func init() {
	registerAidlLibraryBuildComponents(android.InitRegistrationContext)
}

func registerAidlLibraryBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("aidl_library", AidlLibraryFactory)
}

type aidlLibraryProperties struct {
	// srcs lists files that are included in this module for aidl compilation
	Srcs []string `android:"path"`

	// hdrs lists the headers that are imported by srcs but are not compiled by aidl to language binding code
	// hdrs is provided to support Bazel migration. It is a no-op until
	// we enable input sandbox in aidl compilation action
	Hdrs []string `android:"path"`

	// The prefix to strip from the paths of the .aidl files
	// The remaining path is the package path of the aidl interface
	Strip_import_prefix *string

	// List of aidl files or aidl_library depended on by the module
	Deps []string `android:"arch_variant"`

	// Include dirs to all transitive header dependencies
	Include_dirs []string `blueprint:"mutated"`
}

type AidlLibrary struct {
	android.ModuleBase
	android.BazelModuleBase
	properties aidlLibraryProperties
}

type bazelAidlLibraryAttributes struct {
	Srcs                bazel.LabelListAttribute
	Hdrs                bazel.LabelListAttribute
	Strip_import_prefix *string
	Deps                bazel.LabelListAttribute
}

func (lib *AidlLibrary) ConvertWithBp2build(ctx android.TopDownMutatorContext) {
	srcs := bazel.MakeLabelListAttribute(
		android.BazelLabelForModuleSrc(
			ctx,
			lib.properties.Srcs,
		),
	)

	hdrs := bazel.MakeLabelListAttribute(
		android.BazelLabelForModuleSrc(
			ctx,
			lib.properties.Hdrs,
		),
	)

	tags := []string{"apex_available=//apex_available:anyapex"}
	deps := bazel.MakeLabelListAttribute(android.BazelLabelForModuleDeps(ctx, lib.properties.Deps))

	attrs := &bazelAidlLibraryAttributes{
		Srcs:                srcs,
		Hdrs:                hdrs,
		Strip_import_prefix: lib.properties.Strip_import_prefix,
		Deps:                deps,
	}

	props := bazel.BazelTargetModuleProperties{
		Rule_class:        "aidl_library",
		Bzl_load_location: "//build/bazel/rules/aidl:aidl_library.bzl",
	}

	ctx.CreateBazelTargetModule(
		props,
		android.CommonAttributes{
			Name: lib.Name(),
			Tags: bazel.MakeStringListAttribute(tags),
		},
		attrs)
}

func (lib *AidlLibrary) IsMixedBuildSupported(ctx android.BaseModuleContext) bool {
	return false
}

type aidlLibraryInfo struct {
	Srcs        android.Paths
	IncludeDirs android.DepSet
}

var aidlLibraryProvider = blueprint.NewProvider(aidlLibraryInfo{})

func (lib *AidlLibrary) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	srcsDepSetBuilder := android.NewDepSetBuilder(android.PREORDER)
	includeDirsDepSetBuilder := android.NewDepSetBuilder(android.PREORDER)

	srcs := android.PathsForModuleSrc(ctx, lib.properties.Srcs)
	if lib.properties.Strip_import_prefix != nil {
		srcs = android.PathsWithModuleSrcSubDir(
			ctx,
			srcs,
			android.String(lib.properties.Strip_import_prefix))
	}

	includeDir := android.PathForModuleSrc(
		ctx,
		proptools.StringDefault(lib.properties.Strip_import_prefix, ""),
	)

	srcsDepSetBuilder.Direct(srcs...)
	includeDirsDepSetBuilder.Direct(includeDir)

	for _, dep := range ctx.GetDirectDepsWithTag(aidlHeaderTag) {
		if ctx.OtherModuleHasProvider(dep, aidlLibraryProvider) {
			info := ctx.OtherModuleProvider(dep, aidlLibraryProvider).(aidlLibraryInfo)
			includeDirsDepSetBuilder.Transitive(&info.IncludeDirs)
		}
	}

	// TODO(b/279960133) Propagate headers and transitive srcs when aidl action sandboxes inputs
	ctx.SetProvider(aidlLibraryProvider, aidlLibraryInfo{
		Srcs:        srcs,
		IncludeDirs: *includeDirsDepSetBuilder.Build(),
	})
}

// aidl_library contains a list of .aidl files and the strip_import_prefix to
// to strip from the paths of the .aidl files. The sub-path left-over after stripping
// corresponds to the aidl package path the aidl interfaces are scoped in
func AidlLibraryFactory() android.Module {
	module := &AidlLibrary{}
	module.AddProperties(&module.properties)
	android.InitAndroidModule(module)
	android.InitBazelModule(module)
	return module
}

const aidlHeader = iota

type aidlDependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

var aidlHeaderTag = aidlDependencyTag{name: "aidl header"}

func (lib *AidlLibrary) DepsMutator(ctx android.BottomUpMutatorContext) {
	for _, dep := range lib.properties.Deps {
		ctx.AddDependency(lib, aidlHeaderTag, dep)
	}
}
