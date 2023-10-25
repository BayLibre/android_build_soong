// Copyright (C) 2023 The Android Open Source Project
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

package ditto

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"android/soong/android"
	"android/soong/bazel"
	"android/soong/bazel/cquery"
	"android/soong/cc"
	//"github.com/google/blueprint"
)

func init() {
	registerDittoBuildComponents(android.InitRegistrationContext)
	pctx.Import("android/soong/cc/config")
	pctx.StaticVariable("relPwd", cc.PwdPrefix())
}

var (
	pctx = android.NewPackageContext("android/soong/ditto")

	//ccRule = pctx.AndroidRemoteStaticRule("ccRule", android.RemoteRuleSupports{Goma: true},
	//	blueprint.RuleParams{
	//		Depfile:     "${out}.d",
	//		Deps:        blueprint.DepsGCC,
	//		Command:     "$relPwd $ccCmd --target=ditto -c $cFlags -MD -MF ${out}.d -o $out $in",
	//		CommandDeps: []string{"$ccCmd"},
	//	},
	//	"ccCmd", "cFlags")

	//stripRule = pctx.AndroidStaticRule("stripRule",
	//	blueprint.RuleParams{
	//		Command: `$stripCmd --strip-unneeded --remove-section=.rel.BTF ` +
	//			`--remove-section=.rel.BTF.ext --remove-section=.BTF.ext $in -o $out`,
	//		CommandDeps: []string{"$stripCmd"},
	//	},
	//	"stripCmd")
)

func registerDittoBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("ditto_benchmark_standalone", DittoFactory)
}

var PrepareForTestWithDitto = android.FixtureRegisterWithContext(registerDittoBuildComponents)

// DittoModule interface is used by the apex package to gather information from a ditto module.
type DittoModule interface {
	android.Module

	OutputFiles(tag string) (android.Paths, error)

	// Returns the sub install directory if the ditto module is included by apex.
	SubDir() string
}

type DittoProperties struct {
	// source paths to the files.
	Srcs []string `android:"path"`

	// additional cflags that should be used to build the ditto variant of
	// the C/C++ module.
	Cflags []string

	// optional subdirectory under which this module is installed into.
	Sub_dir string

	Benchmarks []string
}

type ditto struct {
	android.ModuleBase
	android.BazelModuleBase

	properties DittoProperties

	objs android.Paths
}

func (ditto *ditto) ImageMutatorBegin(ctx android.BaseModuleContext) {}

func (ditto *ditto) CoreVariantNeeded(ctx android.BaseModuleContext) bool {
	return false
}

func (ditto *ditto) RamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return false
}

func (ditto *ditto) VendorRamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return false
}

func (ditto *ditto) DebugRamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return false
}

func (ditto *ditto) RecoveryVariantNeeded(ctx android.BaseModuleContext) bool {
	return false
}

func (ditto *ditto) ExtraImageVariations(ctx android.BaseModuleContext) []string {
	return nil
}

func (ditto *ditto) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	cflags := []string{
		"-nostdlibinc",

		// Make paths in deps files relative
		"-no-canonical-prefixes",

		"-O2",
		"-isystem bionic/libc/include",
		"-isystem bionic/libc/kernel/uapi",
		// The architecture doesn't matter here, but asm/types.h is included by linux/types.h.
		"-isystem bionic/libc/kernel/uapi/asm-arm64",
		"-isystem bionic/libc/kernel/android/uapi",
		"-I       packages/modules/Connectivity/staticlibs/native/ditto_headers/include/ditto",
		// TODO(b/149785767): only give access to specific file with AID_* constants
		"-I       system/core/libcutils/include",
		"-I " + ctx.ModuleDir(),
	}

	cflags = append(cflags, ditto.properties.Cflags...)

	srcs := android.PathsForModuleSrc(ctx, ditto.properties.Srcs)

	for _, src := range srcs {
		if strings.ContainsRune(filepath.Base(src.String()), '_') {
			ctx.ModuleErrorf("invalid character '_' in source name")
		}
		obj := android.ObjPathWithExt(ctx, "unstripped", src, "o")

		ctx.Build(pctx, android.BuildParams{
			//Rule:   ccRule,
			Input:  src,
			Output: obj,
			Args: map[string]string{
				"cFlags": strings.Join(cflags, " "),
				"ccCmd":  "${config.ClangBin}/clang",
			},
		})

		ditto.objs = append(ditto.objs, obj.WithoutRel())

	}
}

func (ditto *ditto) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Custom: func(w io.Writer, name, prefix, moduleDir string, data android.AndroidMkData) {
			var names []string
			fmt.Fprintln(w)
			fmt.Fprintln(w, "LOCAL_PATH :=", moduleDir)
			fmt.Fprintln(w)
			var localModulePath string
			localModulePath = "LOCAL_MODULE_PATH := $(TARGET_OUT_ETC)/ditto"
			if len(ditto.properties.Sub_dir) > 0 {
				localModulePath += "/" + ditto.properties.Sub_dir
			}
			for _, obj := range ditto.objs {
				objName := name + "_" + obj.Base()
				names = append(names, objName)
				fmt.Fprintln(w, "include $(CLEAR_VARS)", " # ditto.ditto.obj")
				fmt.Fprintln(w, "LOCAL_MODULE := ", objName)
				data.Entries.WriteLicenseVariables(w)
				fmt.Fprintln(w, "LOCAL_PREBUILT_MODULE_FILE :=", obj.String())
				fmt.Fprintln(w, "LOCAL_MODULE_STEM :=", obj.Base())
				fmt.Fprintln(w, "LOCAL_MODULE_CLASS := ETC")
				fmt.Fprintln(w, localModulePath)
				fmt.Fprintln(w, "include $(BUILD_PREBUILT)")
				fmt.Fprintln(w)
			}
			fmt.Fprintln(w, "include $(CLEAR_VARS)", " # ditto.ditto")
			fmt.Fprintln(w, "LOCAL_MODULE := ", name)
			data.Entries.WriteLicenseVariables(w)
			android.AndroidMkEmitAssignList(w, "LOCAL_REQUIRED_MODULES", names)
			fmt.Fprintln(w, "include $(BUILD_PHONY_PACKAGE)")
		},
	}
}

var _ android.MixedBuildBuildable = (*ditto)(nil)

func (ditto *ditto) IsMixedBuildSupported(ctx android.BaseModuleContext) bool {
	return true
}

func (ditto *ditto) QueueBazelCall(ctx android.BaseModuleContext) {
	bazelCtx := ctx.Config().BazelContext
	bazelCtx.QueueBazelRequest(
		ditto.GetBazelLabel(ctx, ditto),
		cquery.GetOutputFiles,
		android.GetConfigKey(ctx))
}

func (ditto *ditto) ProcessBazelQueryResponse(ctx android.ModuleContext) {
	bazelCtx := ctx.Config().BazelContext
	objPaths, err := bazelCtx.GetOutputFiles(ditto.GetBazelLabel(ctx, ditto), android.GetConfigKey(ctx))
	if err != nil {
		ctx.ModuleErrorf(err.Error())
		return
	}

	bazelOuts := android.Paths{}
	for _, p := range objPaths {
		bazelOuts = append(bazelOuts, android.PathForBazelOut(ctx, p))
	}
	ditto.objs = bazelOuts
}

// Implements OutputFileFileProducer interface so that the obj output can be used in the data property
// of other modules.
func (ditto *ditto) OutputFiles(tag string) (android.Paths, error) {
	switch tag {
	case "":
		return ditto.objs, nil
	default:
		return nil, fmt.Errorf("unsupported module reference tag %q", tag)
	}
}

func (ditto *ditto) SubDir() string {
	return ditto.properties.Sub_dir
}

var _ android.OutputFileProducer = (*ditto)(nil)

func DittoFactory() android.Module {
	module := &ditto{}

	module.AddProperties(&module.properties)

	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	android.InitBazelModule(module)
	return module
}

type bazelDittoAttributes struct {
	Srcs              bazel.LabelListAttribute
	Copts             bazel.StringListAttribute
	Absolute_includes bazel.StringListAttribute
}

// ditto bp2build converter
func (b *ditto) ConvertWithBp2build(ctx android.Bp2buildMutatorContext) {
	if ctx.ModuleType() != "ditto" {
		return
	}

	srcs := bazel.MakeLabelListAttribute(android.BazelLabelForModuleSrc(ctx, b.properties.Srcs))
	copts := bazel.MakeStringListAttribute(b.properties.Cflags)

	attrs := bazelDittoAttributes{
		Srcs:  srcs,
		Copts: copts,
	}
	props := bazel.BazelTargetModuleProperties{
		Rule_class:        "ditto",
		Bzl_load_location: "//build/bazel/rules/ditto:ditto.bzl",
	}

	ctx.CreateBazelTargetModule(props, android.CommonAttributes{Name: b.Name()}, &attrs)
}
