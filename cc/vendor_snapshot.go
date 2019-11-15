// Copyright 2019 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cc

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

func init() {
	android.RegisterSingletonType("vendor-snapshot", VendorSnapshotSingleton)
}

func VendorSnapshotSingleton() android.Singleton {
	return &vendorSnapshotSingleton{}
}

type vendorSnapshotSingleton struct {
	vendorSnapshotZipFile android.OptionalPath
}

/*
Determine if a module is going to be included in vendor snapshot or not.
- All "vendor: true", "vendor_available: true" libraries and binaries are captured.
- VNDK static libraries are captured.

Ignores modules under following directories:
	device/
	vendor/
	cts/
	external/llvm/
	external/clang/
	hardware/ (except for interfaces/, libhardware/, libhardware_legacy/, ril/)
*/
func isVendorSnapshotModule(ctx android.SingletonContext, m *Module) bool {
	dir := ctx.ModuleDir(m)

	for _, p := range []string{"vendor", "device", "cts", "external/llvm", "external/clang"} {
		if strings.HasPrefix(dir, p) {
			return false
		}
	}
	if strings.HasPrefix(dir, "hardware/") {
		aosp := false
		for _, p := range []string{"interfaces", "libhardware", "libhardware_legacy", "ril"} {
			if strings.HasPrefix(dir, p) {
				aosp = true
				break
			}
		}
		if !aosp {
			return false
		}
	}
	if m.Target().NativeBridge == android.NativeBridgeEnabled {
		return false
	}
	if !m.UseVndk() || !m.IsForPlatform() || !m.installable() || m.isSnapshotPrebuilt() {
		return false
	}
	if m.sanitize != nil && !m.sanitize.isUnsanitizedVariant() {
		return false
	}

	// Libraries
	if l, ok := m.linker.(snapshotLibraryInterface); ok {
		if l.static() {
			return proptools.BoolDefault(m.VendorProperties.Vendor_available, true)
		}
		if l.shared() {
			return !m.IsVndk()
		}
		return true
	}

	// Binaries
	_, ok := m.linker.(*binaryDecorator)
	if !ok {
		if _, ok := m.linker.(*prebuiltBinaryLinker); !ok {
			return false
		}
	}
	return proptools.BoolDefault(m.VendorProperties.Vendor_available, true)
}

func (c *vendorSnapshotSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// BOARD_VNDK_VERSION must be set to 'current' in order to generate a vendor snapshot.
	if ctx.DeviceConfig().VndkVersion() != "current" {
		return
	}

	var snapshotOutputs android.Paths

	/*
		Vendor snapshot zipped artifacts directory structure:
		{SNAPSHOT_ARCH}/
			arch-{TARGET_ARCH}-{TARGET_ARCH_VARIANT}/
				shared/
					(.so shared libraries)
				static/
					(.a static libraries)
				header/
					(header only libraries)
				binary/
					(executable binaries)
			arch-{TARGET_2ND_ARCH}-{TARGET_2ND_ARCH_VARIANT}/
				shared/
					(.so shared libraries)
				static/
					(.a static libraries)
				header/
					(header only libraries)
				binary/
					(executable binaries)
			NOTICE_FILES/
				(notice files, e.g. libbase.txt)
			include/
				(header files of same directory structure with source tree)
	*/

	snapshotDir := "vendor-snapshot"
	snapshotArchDir := filepath.Join(snapshotDir, ctx.DeviceConfig().DeviceArch())

	targetArchMap := targetArchMap(ctx.Config())
	includeDir := filepath.Join(snapshotArchDir, "include")
	noticeDir := filepath.Join(snapshotArchDir, "NOTICE_FILES")

	noticeBuilt := make(map[string]bool)

	var headers android.Paths

	type vendorSnapshotLibraryInterface interface {
		exportedFlagsProducer
		libraryInterface
	}

	var _ vendorSnapshotLibraryInterface = (*prebuiltLibraryLinker)(nil)
	var _ vendorSnapshotLibraryInterface = (*libraryDecorator)(nil)

	installLibrary := func(m *Module, l vendorSnapshotLibraryInterface) android.Paths {
		targetArch := targetArchMap[m.Target().Arch.ArchType]

		var ret android.Paths

		var libType string
		if l.static() {
			libType = "static"
		} else if l.shared() {
			libType = "shared"
		} else {
			libType = "header"
		}

		var stem string

		if libType != "header" {
			libPath := m.outputFile.Path()
			stem = libPath.Base()
			snapshotLibOut := filepath.Join(snapshotArchDir, targetArch, libType, stem)
			ret = append(ret, copyFile(ctx, libPath, snapshotLibOut))
		} else {
			stem = ctx.ModuleName(m)
		}

		// TODO: add required and shared_libs dependencies
		prop := struct {
			ExportedDirs        []string `json:",omitempty"`
			ExportedSystemDirs  []string `json:",omitempty"`
			ExportedFlags       []string `json:",omitempty"`
			ModuleName          string   `json:",omitempty"`
			RelativeInstallPath string   `json:",omitempty"`
		}{}
		prop.ExportedFlags = l.exportedFlags()
		prop.ExportedDirs = l.exportedDirs().Strings()
		prop.ExportedSystemDirs = l.exportedSystemDirs().Strings()
		prop.ModuleName = ctx.ModuleName(m)
		prop.RelativeInstallPath = m.RelativeInstallPath()
		propOut := filepath.Join(snapshotArchDir, targetArch, libType, stem+".json")

		j, err := json.Marshal(prop)
		if err != nil {
			ctx.Errorf("json marshal to %q failed: %#v", propOut, err)
			return nil
		}
		ret = append(ret, writeStringToFile(ctx, string(j), propOut))

		return ret
	}

	installBinary := func(m *Module) android.Paths {
		targetArch := targetArchMap[m.Target().Arch.ArchType]

		var ret android.Paths

		binPath := m.outputFile.Path()
		snapshotBinOut := filepath.Join(snapshotArchDir, targetArch, "binary", binPath.Base())
		ret = append(ret, copyFile(ctx, binPath, snapshotBinOut))

		// TODO: add required and shared_libs dependencies
		prop := struct {
			ModuleName          string `json:",omitempty"`
			RelativeInstallPath string `json:",omitempty"`
		}{}
		prop.ModuleName = ctx.ModuleName(m)
		prop.RelativeInstallPath = m.RelativeInstallPath()
		propOut := snapshotBinOut + ".json"

		j, err := json.Marshal(prop)
		if err != nil {
			ctx.Errorf("json marshal to %q failed: %#v", propOut, err)
			return nil
		}
		ret = append(ret, writeStringToFile(ctx, string(j), propOut))

		return ret
	}

	ctx.VisitAllModules(func(module android.Module) {
		m, ok := module.(*Module)
		if !ok || !m.Enabled() || !isVendorSnapshotModule(ctx, m) {
			return
		}

		if _, ok := targetArchMap[m.Target().Arch.ArchType]; !ok {
			return
		}

		if l, ok := m.linker.(vendorSnapshotLibraryInterface); ok {
			snapshotOutputs = append(snapshotOutputs, installLibrary(m, l)...)
			headers = append(headers, exportedHeaders(ctx, l)...)
		} else {
			snapshotOutputs = append(snapshotOutputs, installBinary(m)...)
		}

		if m.NoticeFile().Valid() {
			noticeName := ctx.ModuleName(m) + ".txt"
			// skip already copied notice file
			if _, ok := noticeBuilt[noticeName]; !ok {
				noticeBuilt[noticeName] = true
				snapshotOutputs = append(snapshotOutputs, copyFile(
					ctx, m.NoticeFile().Path(), filepath.Join(noticeDir, noticeName)))
			}
		}
	})

	// install all headers after removing duplicates
	for _, header := range android.FirstUniqueStringPaths(headers) {
		snapshotOutputs = append(snapshotOutputs, copyFile(
			ctx, header, filepath.Join(includeDir, header.String())))
	}

	// All artifacts are ready. Sort them to normalize ninja and then zip.
	snapshotOutputs = android.FirstUniqueStringPaths(snapshotOutputs)
	sort.Slice(snapshotOutputs, func(i, j int) bool {
		return snapshotOutputs[i].String() < snapshotOutputs[j].String()
	})

	zipPath := android.PathForOutput(ctx, snapshotDir, "vendor-"+ctx.Config().DeviceName()+".zip")
	zipRule := android.NewRuleBuilder()

	// filenames in rspfile from FlagWithRspFileInputList might be single-quoted. Remove it with tr
	snapshotOutputList := android.PathForOutput(ctx, snapshotDir, "vendor-"+ctx.Config().DeviceName()+"_list")
	zipRule.Command().
		Text("tr").
		FlagWithArg("-d ", "\\'").
		FlagWithRspFileInputList("< ", snapshotOutputs).
		FlagWithOutput("> ", snapshotOutputList)

	zipRule.Temporary(snapshotOutputList)

	zipRule.Command().
		BuiltTool(ctx, "soong_zip").
		FlagWithOutput("-o ", zipPath).
		FlagWithArg("-C ", android.PathForOutput(ctx, snapshotDir).String()).
		FlagWithInput("-l ", snapshotOutputList)

	zipRule.Build(pctx, ctx, zipPath.String(), "vendor snapshot "+zipPath.String())
	zipRule.DeleteTemporaryFiles()
	c.vendorSnapshotZipFile = android.OptionalPathForPath(zipPath)
}

func (c *vendorSnapshotSingleton) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict("SOONG_VENDOR_SNAPSHOT_ZIP", c.vendorSnapshotZipFile.String())
}
