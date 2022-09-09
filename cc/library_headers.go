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

package cc

import (
	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/bazel"
	"android/soong/bazel/cquery"
)

func init() {
	RegisterLibraryHeadersBuildComponents(android.InitRegistrationContext)

	// Register sdk member types.
	android.RegisterSdkMemberType(headersLibrarySdkMemberType)

}

var headersLibrarySdkMemberType = &librarySdkMemberType{
	SdkMemberTypeBase: android.SdkMemberTypeBase{
		PropertyName:    "native_header_libs",
		SupportsSdk:     true,
		HostOsDependent: true,
		Traits: []android.SdkMemberTrait{
			nativeBridgeSdkTrait,
			ramdiskImageRequiredSdkTrait,
			recoveryImageRequiredSdkTrait,
		},
	},
	prebuiltModuleType: "cc_prebuilt_library_headers",
	noOutputFiles:      true,
}

func RegisterLibraryHeadersBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("cc_library_headers", LibraryHeaderFactory)
	ctx.RegisterModuleType("cc_prebuilt_library_headers", prebuiltLibraryHeaderFactory)
}

type libraryHeaderBazelHandler struct {
	module  *Module
	library *libraryDecorator
}

var _ BazelHandler = (*libraryHeaderBazelHandler)(nil)

func (handler *libraryHeaderBazelHandler) QueueBazelCall(ctx android.BaseModuleContext, label string) {
	bazelCtx := ctx.Config().BazelContext
	bazelCtx.QueueBazelRequest(label, cquery.GetCcInfo, android.GetConfigKey(ctx))
}

func (h *libraryHeaderBazelHandler) ProcessBazelQueryResponse(ctx android.ModuleContext, label string) {
	bazelCtx := ctx.Config().BazelContext
	ccInfo, err := bazelCtx.GetCcInfo(label, android.GetConfigKey(ctx))
	if err != nil {
		ctx.ModuleErrorf(err.Error())
		return
	}

	outputPaths := ccInfo.OutputFiles
	if len(outputPaths) != 1 {
		ctx.ModuleErrorf("expected exactly one output file for %q, but got %q", label, outputPaths)
		return
	}

	outputPath := android.PathForBazelOut(ctx, outputPaths[0])
	h.module.outputFile = android.OptionalPathForPath(outputPath)

	// HeaderLibraryInfo is an empty struct to indicate to dependencies that this is a header library
	ctx.SetProvider(HeaderLibraryInfoProvider, HeaderLibraryInfo{})

	h.library.setFlagExporterInfoFromCcInfo(ctx, ccInfo)

	// Dependencies on this library will expect collectedSnapshotHeaders to be set, otherwise
	// validation will fail. For now, set this to an empty list.
	// TODO(cparsons): More closely mirror the collectHeadersForSnapshot implementation.
	h.library.collectedSnapshotHeaders = android.Paths{}
}

// cc_library_headers contains a set of c/c++ headers which are imported by
// other soong cc modules using the header_libs property. For best practices,
// use export_include_dirs property or LOCAL_EXPORT_C_INCLUDE_DIRS for
// Make.
func LibraryHeaderFactory() android.Module {
	module, library := NewLibrary(android.HostAndDeviceSupported)
	library.HeaderOnly()
	module.sdkMemberTypes = []android.SdkMemberType{headersLibrarySdkMemberType}
	module.bazelable = true
	module.bazelHandler = &libraryHeaderBazelHandler{module: module, library: library}
	return module.Init()
}

// cc_prebuilt_library_headers is a prebuilt version of cc_library_headers
func prebuiltLibraryHeaderFactory() android.Module {
	module, library := NewPrebuiltLibrary(android.HostAndDeviceSupported, "")
	library.HeaderOnly()
	module.bazelable = true
	module.bazelHandler = &ccLibraryBazelHandler{module: module}
	return module.Init()
}

type bazelCcLibraryHeadersAttributes struct {
	Hdrs                     bazel.LabelListAttribute
	Export_includes          bazel.StringListAttribute
	Export_absolute_includes bazel.StringListAttribute
	Export_system_includes   bazel.StringListAttribute
	Deps                     bazel.LabelListAttribute
	Implementation_deps      bazel.LabelListAttribute
	System_dynamic_deps      bazel.LabelListAttribute
	sdkAttributes
}

func libraryHeadersBp2Build(ctx android.TopDownMutatorContext, module *Module) {
	baseAttributes := bp2BuildParseBaseProps(ctx, module)
	exportedIncludes := bp2BuildParseExportedIncludes(ctx, module, &baseAttributes.includes)
	linkerAttrs := baseAttributes.linkerAttributes
	(&linkerAttrs.deps).Append(linkerAttrs.dynamicDeps)
	(&linkerAttrs.deps).Append(linkerAttrs.wholeArchiveDeps)

	attrs := &bazelCcLibraryHeadersAttributes{
		Export_includes:          exportedIncludes.Includes,
		Export_absolute_includes: exportedIncludes.AbsoluteIncludes,
		Export_system_includes:   exportedIncludes.SystemIncludes,
		Deps:                     linkerAttrs.deps,
		System_dynamic_deps:      linkerAttrs.systemDynamicDeps,
		Hdrs:                     baseAttributes.hdrs,
		sdkAttributes:            bp2BuildParseSdkAttributes(module),
	}

	props := bazel.BazelTargetModuleProperties{
		Rule_class:        "cc_library_headers",
		Bzl_load_location: "//build/bazel/rules/cc:cc_library_headers.bzl",
	}

	ctx.CreateBazelTargetModule(props, android.CommonAttributes{Name: module.Name()}, attrs)
}

type bazelCcApiLibraryHeadersAttributes struct {
	bazelCcLibraryHeadersAttributes

	Arch *string
}

func (a *bazelCcApiLibraryHeadersAttributes) IsEmpty() bool {
	return a.Export_includes.IsEmpty() &&
		a.Export_system_includes.IsEmpty() &&
		a.Deps.IsEmpty()
}

func apiBazelTargets(ll bazel.LabelList) bazel.LabelList {
	labels := make([]bazel.Label, 0)
	for _, l := range ll.Includes {
		labels = append(labels, bazel.Label{
			Label: apiContributionTargetName(l.Label),
		})
	}
	return bazel.MakeLabelList(labels)
}

// TODO: Add documentation
func removeStaticExports(ctx android.BazelConversionPathContext, allExports bazel.LabelList, staticModules []string) bazel.LabelList {
	staticExports := bazelLabelForStaticDeps(ctx, staticModules)
	return bazel.SubtractBazelLabelList(allExports, staticExports)
}

func archAgnosticApiExports(ctx android.TopDownMutatorContext, a baseAttributes, i BazelIncludes) *bazelCcApiLibraryHeadersAttributes {
	// Base values for NoConfigAxis
	includes := i.Includes.Value
	systemIncludes := i.SystemIncludes.Value
	deps := a.deps.Value
	// os == android
	includes = append(includes, i.Includes.SelectValue(
		bazel.OsConfigurationAxis,
		bazel.OsAndroid)...,
	)
	systemIncludes = append(systemIncludes, i.SystemIncludes.SelectValue(
		bazel.OsConfigurationAxis,
		bazel.OsAndroid)...,
	)
	deps.Append(a.deps.SelectValue(
		bazel.OsConfigurationAxis,
		bazel.OsAndroid),
	)
	nonStaticDeps := removeStaticExports(ctx, deps, ctx.Module().(*Module).linker.(*libraryDecorator).baseLinker.Properties.Export_static_lib_headers)
	ret := &bazelCcApiLibraryHeadersAttributes{}
	ret.Export_includes = bazel.MakeStringListAttribute(includes)
	ret.Export_system_includes = bazel.MakeStringListAttribute(systemIncludes)
	depAttr := bazel.MakeLabelListAttribute(apiBazelTargets(nonStaticDeps)) // append .contribution suffix to `otherlib`'s label
	if !depAttr.IsEmpty() {
		ret.Deps = depAttr
	}
	return ret
}

type bazelArchApiAttributes map[string]*bazelCcApiLibraryHeadersAttributes

func archSpecificApiExports(ctx android.TopDownMutatorContext, a baseAttributes, i BazelIncludes) bazelArchApiAttributes {
	ret := bazelArchApiAttributes{}
	for _, arch := range allArches {
		includes := i.Includes.SelectValue(
			bazel.ArchConfigurationAxis,
			arch)
		systemIncludes := i.SystemIncludes.SelectValue(
			bazel.ArchConfigurationAxis,
			arch)
		deps := a.deps.SelectValue(
			bazel.ArchConfigurationAxis,
			arch)
		attrs := &bazelCcApiLibraryHeadersAttributes{}
		attrs.Export_includes = bazel.MakeStringListAttribute(includes)
		attrs.Export_system_includes = bazel.MakeStringListAttribute(systemIncludes)
		depAttr := bazel.MakeLabelListAttribute(apiBazelTargets(deps)) // append .contribution suffix to `otherlib`'s label
		if !depAttr.IsEmpty() {
			attrs.Deps = depAttr
		}
		attrs.Arch = proptools.StringPtr(arch)
		ret[arch] = attrs
	}
	return ret
}

var (
	allArches = []string{"arm", "arm64", "x86", "x86_64"}
)

// TODO: Add documentation
func apiLibraryHeadersBp2Build(ctx android.TopDownMutatorContext, module *Module) {
	createTarget := func(ctx android.TopDownMutatorContext, name string, a *bazelCcApiLibraryHeadersAttributes) {
		props := bazel.BazelTargetModuleProperties{
			Rule_class:        "cc_api_library_headers",
			Bzl_load_location: "//build/bazel/rules/apis:cc_api_contribution.bzl",
		}
		ctx.CreateBazelTargetModule(props, android.CommonAttributes{Name: name}, a)
	}
	// cc_api_library_headers have a 1:1 mapping to arch/no-arch
	// For API export,create a top-level arch-agnostic target and list the arch-specific targets as its deps
	// TODO: Add more doc about how this allows 1 target per arch and not 1 target per (arch,export_include_dir) (arch,export_system_include_dir) etc
	baseProps := bp2BuildParseBaseProps(ctx, module)
	exportedIncludes := bp2BuildParseExportedIncludes(ctx, module, &baseProps.includes)
	commonExports := archAgnosticApiExports(ctx, baseProps, exportedIncludes)
	archExports := archSpecificApiExports(ctx, baseProps, exportedIncludes)

	// For each archExport, create a Bazel target and add it to the deps of `commonExports`
	for _, arch := range allArches { // sorted iteration
		dep := archExports[arch]
		if !dep.IsEmpty() {
			name := apiContributionTargetName(module.Name()) + "." + arch // Add an arch suffix for arch-specific target
			label := bazel.Label{
				Label: ":" + name,
			}
			createTarget(ctx, name, dep)
			commonExports.Deps.Value.Add(&label)
		}
	}

	//Create a target only if this module exports headers
	if !commonExports.IsEmpty() {
		createTarget(ctx, apiContributionTargetName(module.Name()), commonExports)
	}
}
