// Copyright 2017 Google Inc. All rights reserved.
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
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"android/soong/android"
	"android/soong/cc/config"
)

type VndkProperties struct {
	Vndk struct {
		// declared as a VNDK or VNDK-SP module. The vendor variant
		// will be installed in /system instead of /vendor partition.
		//
		// `vendor_available` must be explicitly set to either true or
		// false together with `vndk: {enabled: true}`.
		Enabled *bool

		// declared as a VNDK-SP module, which is a subset of VNDK.
		//
		// `vndk: { enabled: true }` must set together.
		//
		// All these modules are allowed to link to VNDK-SP or LL-NDK
		// modules only. Other dependency will cause link-type errors.
		//
		// If `support_system_process` is not set or set to false,
		// the module is VNDK-core and can link to other VNDK-core,
		// VNDK-SP or LL-NDK modules only.
		Support_system_process *bool

		// Extending another module
		Extends *string
	}
}

type vndkdep struct {
	Properties VndkProperties
}

func (vndk *vndkdep) props() []interface{} {
	return []interface{}{&vndk.Properties}
}

func (vndk *vndkdep) begin(ctx BaseModuleContext) {}

func (vndk *vndkdep) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func (vndk *vndkdep) isVndk() bool {
	return Bool(vndk.Properties.Vndk.Enabled)
}

func (vndk *vndkdep) isVndkSp() bool {
	return Bool(vndk.Properties.Vndk.Support_system_process)
}

func (vndk *vndkdep) isVndkExt() bool {
	return vndk.Properties.Vndk.Extends != nil
}

func (vndk *vndkdep) getVndkExtendsModuleName() string {
	return String(vndk.Properties.Vndk.Extends)
}

func (vndk *vndkdep) typeName() string {
	if !vndk.isVndk() {
		return "native:vendor"
	}
	if !vndk.isVndkExt() {
		if !vndk.isVndkSp() {
			return "native:vendor:vndk"
		}
		return "native:vendor:vndksp"
	}
	if !vndk.isVndkSp() {
		return "native:vendor:vndkext"
	}
	return "native:vendor:vndkspext"
}

func (vndk *vndkdep) vndkCheckLinkType(ctx android.ModuleContext, to *Module, tag DependencyTag) {
	if to.linker == nil {
		return
	}
	if !vndk.isVndk() {
		// Non-VNDK modules (those installed to /vendor) can't depend on modules marked with
		// vendor_available: false.
		violation := false
		if lib, ok := to.linker.(*llndkStubDecorator); ok && !Bool(lib.Properties.Vendor_available) {
			violation = true
		} else {
			if _, ok := to.linker.(libraryInterface); ok && to.VendorProperties.Vendor_available != nil && !Bool(to.VendorProperties.Vendor_available) {
				// Vendor_available == nil && !Bool(Vendor_available) should be okay since
				// it means a vendor-only library which is a valid dependency for non-VNDK
				// modules.
				violation = true
			}
		}
		if violation {
			ctx.ModuleErrorf("Vendor module that is not VNDK should not link to %q which is marked as `vendor_available: false`", to.Name())
		}
	}
	if lib, ok := to.linker.(*libraryDecorator); !ok || !lib.shared() {
		// Check only shared libraries.
		// Other (static and LL-NDK) libraries are allowed to link.
		return
	}
	if !to.UseVndk() {
		ctx.ModuleErrorf("(%s) should not link to %q which is not a vendor-available library",
			vndk.typeName(), to.Name())
		return
	}
	if tag == vndkExtDepTag {
		// Ensure `extends: "name"` property refers a vndk module that has vendor_available
		// and has identical vndk properties.
		if to.vndkdep == nil || !to.vndkdep.isVndk() {
			ctx.ModuleErrorf("`extends` refers a non-vndk module %q", to.Name())
			return
		}
		if vndk.isVndkSp() != to.vndkdep.isVndkSp() {
			ctx.ModuleErrorf(
				"`extends` refers a module %q with mismatched support_system_process",
				to.Name())
			return
		}
		if !Bool(to.VendorProperties.Vendor_available) {
			ctx.ModuleErrorf(
				"`extends` refers module %q which does not have `vendor_available: true`",
				to.Name())
			return
		}
	}
	if to.vndkdep == nil {
		return
	}

	// Check the dependencies of VNDK shared libraries.
	if err := vndkIsVndkDepAllowed(vndk, to.vndkdep); err != nil {
		ctx.ModuleErrorf("(%s) should not link to %q (%s): %v",
			vndk.typeName(), to.Name(), to.vndkdep.typeName(), err)
		return
	}
}

func vndkIsVndkDepAllowed(from *vndkdep, to *vndkdep) error {
	// Check the dependencies of VNDK, VNDK-Ext, VNDK-SP, VNDK-SP-Ext and vendor modules.
	if from.isVndkExt() {
		if from.isVndkSp() {
			if to.isVndk() && !to.isVndkSp() {
				return errors.New("VNDK-SP extensions must not depend on VNDK or VNDK extensions")
			}
			return nil
		}
		// VNDK-Ext may depend on VNDK, VNDK-Ext, VNDK-SP, VNDK-SP-Ext, or vendor libs.
		return nil
	}
	if from.isVndk() {
		if to.isVndkExt() {
			return errors.New("VNDK-core and VNDK-SP must not depend on VNDK extensions")
		}
		if from.isVndkSp() {
			if !to.isVndkSp() {
				return errors.New("VNDK-SP must only depend on VNDK-SP")
			}
			return nil
		}
		if !to.isVndk() {
			return errors.New("VNDK-core must only depend on VNDK-core or VNDK-SP")
		}
		return nil
	}
	// Vendor modules may depend on VNDK, VNDK-Ext, VNDK-SP, VNDK-SP-Ext, or vendor libs.
	return nil
}

var (
	vndkCoreLibrariesKey                = android.NewOnceKey("vndkCoreLibrarires")
	vndkSpLibrariesKey                  = android.NewOnceKey("vndkSpLibrarires")
	llndkLibrariesKey                   = android.NewOnceKey("llndkLibrarires")
	vndkPrivateLibrariesKey             = android.NewOnceKey("vndkPrivateLibrarires")
	vndkUsingCoreVariantLibrariesKey    = android.NewOnceKey("vndkUsingCoreVariantLibrarires")
	modulePathsKey                      = android.NewOnceKey("modulePaths")
	vndkSnapshotZipKey                  = android.NewOnceKey("vndkSnapshotZip")
	vndkMustUseVendorVariantListKey     = android.NewOnceKey("vndkMustUseVendorVariantListKey")
	testVndkMustUseVendorVariantListKey = android.NewOnceKey("testVndkMustUseVendorVariantListKey")
	vndkLibrariesLock                   sync.Mutex

	headerExts = []string{".h", ".hh", ".hpp", ".hxx", ".h++", ".inl", ".inc", ".ipp", ".h.generic"}
)

func vndkCoreLibraries(config android.Config) *[]string {
	return config.Once(vndkCoreLibrariesKey, func() interface{} {
		return &[]string{}
	}).(*[]string)
}

func vndkSpLibraries(config android.Config) *[]string {
	return config.Once(vndkSpLibrariesKey, func() interface{} {
		return &[]string{}
	}).(*[]string)
}

func llndkLibraries(config android.Config) *[]string {
	return config.Once(llndkLibrariesKey, func() interface{} {
		return &[]string{}
	}).(*[]string)
}

func vndkPrivateLibraries(config android.Config) *[]string {
	return config.Once(vndkPrivateLibrariesKey, func() interface{} {
		return &[]string{}
	}).(*[]string)
}

func vndkUsingCoreVariantLibraries(config android.Config) *[]string {
	return config.Once(vndkUsingCoreVariantLibrariesKey, func() interface{} {
		return &[]string{}
	}).(*[]string)
}

func modulePaths(config android.Config) map[string]string {
	return config.Once(modulePathsKey, func() interface{} {
		return make(map[string]string)
	}).(map[string]string)
}

func vndkSnapshotZip(config android.Config) *android.OptionalPath {
	return config.Once(vndkSnapshotZipKey, func() interface{} {
		return &android.OptionalPath{}
	}).(*android.OptionalPath)
}

func vndkMustUseVendorVariantList(cfg android.Config) []string {
	return cfg.Once(vndkMustUseVendorVariantListKey, func() interface{} {
		override := cfg.Once(testVndkMustUseVendorVariantListKey, func() interface{} {
			return []string(nil)
		}).([]string)
		if override != nil {
			return override
		}
		return config.VndkMustUseVendorVariantList
	}).([]string)
}

// test may call this to override global configuration(config.VndkMustUseVendorVariantList)
// when it is called, it must be before the first call to vndkMustUseVendorVariantList()
func setVndkMustUseVendorVariantListForTest(config android.Config, mustUseVendorVariantList []string) {
	config.Once(testVndkMustUseVendorVariantListKey, func() interface{} {
		return mustUseVendorVariantList
	})
}

func processLlndkLibrary(mctx android.BottomUpMutatorContext, m *Module) {
	lib := m.linker.(*llndkStubDecorator)
	name := strings.TrimSuffix(m.Name(), llndkLibrarySuffix)

	vndkLibrariesLock.Lock()
	defer vndkLibrariesLock.Unlock()

	llndkLibraries := llndkLibraries(mctx.Config())
	if !inList(name, *llndkLibraries) {
		*llndkLibraries = append(*llndkLibraries, name)
		sort.Strings(*llndkLibraries)
	}
	if !Bool(lib.Properties.Vendor_available) {
		vndkPrivateLibraries := vndkPrivateLibraries(mctx.Config())
		if !inList(name, *vndkPrivateLibraries) {
			*vndkPrivateLibraries = append(*vndkPrivateLibraries, name)
			sort.Strings(*vndkPrivateLibraries)
		}
	}
}

func processVndkLibrary(mctx android.BottomUpMutatorContext, m *Module) {
	name := strings.TrimPrefix(m.Name(), "prebuilt_")

	vndkLibrariesLock.Lock()
	defer vndkLibrariesLock.Unlock()

	modulePaths := modulePaths(mctx.Config())
	if inList(name, vndkMustUseVendorVariantList(mctx.Config())) {
		m.Properties.MustUseVendorVariant = true
	}
	if mctx.DeviceConfig().VndkUseCoreVariant() && !inList(name, vndkMustUseVendorVariantList(mctx.Config())) {
		vndkUsingCoreVariantLibraries := vndkUsingCoreVariantLibraries(mctx.Config())
		if !inList(name, *vndkUsingCoreVariantLibraries) {
			*vndkUsingCoreVariantLibraries = append(*vndkUsingCoreVariantLibraries, name)
			sort.Strings(*vndkUsingCoreVariantLibraries)
		}
	}
	if m.vndkdep.isVndkSp() {
		vndkSpLibraries := vndkSpLibraries(mctx.Config())
		if !inList(name, *vndkSpLibraries) {
			*vndkSpLibraries = append(*vndkSpLibraries, name)
			sort.Strings(*vndkSpLibraries)
			modulePaths[name] = mctx.ModuleDir()
		}
	} else {
		vndkCoreLibraries := vndkCoreLibraries(mctx.Config())
		if !inList(name, *vndkCoreLibraries) {
			*vndkCoreLibraries = append(*vndkCoreLibraries, name)
			sort.Strings(*vndkCoreLibraries)
			modulePaths[name] = mctx.ModuleDir()
		}
	}
	if !Bool(m.VendorProperties.Vendor_available) {
		vndkPrivateLibraries := vndkPrivateLibraries(mctx.Config())
		if !inList(name, *vndkPrivateLibraries) {
			*vndkPrivateLibraries = append(*vndkPrivateLibraries, name)
			sort.Strings(*vndkPrivateLibraries)
		}
	}
}

func IsForVndkApex(mctx android.BottomUpMutatorContext, m *Module) bool {
	if !m.Enabled() {
		return false
	}

	if !mctx.Device() {
		return false
	}

	if m.Target().NativeBridge == android.NativeBridgeEnabled {
		return false
	}

	// prebuilt vndk modules should match with device
	// TODO(b/142675459): Use enabled: to select target device in vndk_prebuilt_shared
	// When b/142675459 is landed, remove following check
	if p, ok := m.linker.(*vndkPrebuiltLibraryDecorator); ok && !p.matchesWithDevice(mctx.DeviceConfig()) {
		return false
	}

	if lib, ok := m.linker.(libraryInterface); ok {
		useCoreVariant := m.vndkVersion() == mctx.DeviceConfig().PlatformVndkVersion() &&
			mctx.DeviceConfig().VndkUseCoreVariant() && !m.MustUseVendorVariant()
		return lib.shared() && m.UseVndk() && m.IsVndk() && !m.isVndkExt() && !useCoreVariant
	}
	return false
}

// gather list of vndk-core, vndk-sp, and ll-ndk libs
func VndkMutator(mctx android.BottomUpMutatorContext) {
	m, ok := mctx.Module().(*Module)
	if !ok {
		return
	}
	if !m.Enabled() {
		return
	}
	if m.Target().NativeBridge == android.NativeBridgeEnabled {
		// Skip native_bridge modules
		return
	}

	if _, ok := m.linker.(*llndkStubDecorator); ok {
		processLlndkLibrary(mctx, m)
		return
	}

	lib, is_lib := m.linker.(*libraryDecorator)
	prebuilt_lib, is_prebuilt_lib := m.linker.(*prebuiltLibraryLinker)

	if (is_lib && lib.buildShared()) || (is_prebuilt_lib && prebuilt_lib.buildShared()) {
		if m.vndkdep != nil && m.vndkdep.isVndk() && !m.vndkdep.isVndkExt() {
			processVndkLibrary(mctx, m)
			return
		}
	}
}

func init() {
	android.RegisterSingletonType("vndk-snapshot", VndkSnapshotSingleton)
	android.RegisterMakeVarsProvider(pctx, func(ctx android.MakeVarsContext) {
		ctx.Strict("SOONG_VNDK_SNAPSHOT_ZIP", vndkSnapshotZip(ctx.Config()).String())
	})
}

func VndkSnapshotSingleton() android.Singleton {
	return &vndkSnapshotSingleton{}
}

type vndkSnapshotSingleton struct{}

func (c *vndkSnapshotSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// BOARD_VNDK_VERSION must be set to 'current' in order to generate a VNDK snapshot.
	if ctx.DeviceConfig().VndkVersion() != "current" {
		return
	}

	if ctx.DeviceConfig().PlatformVndkVersion() == "" {
		return
	}

	if ctx.DeviceConfig().BoardVndkRuntimeDisable() {
		return
	}

	c.buildVndkLibrariesTxtFiles(ctx)

	var snapshotOutputs android.Paths

	/*
		VNDK snapshot zipped artifacts directory structure:
		{SNAPSHOT_ARCH}/
			arch-{TARGET_ARCH}-{TARGET_ARCH_VARIANT}/
				shared/
					vndk-core/
						(VNDK-core libraries, e.g. libbinder.so)
					vndk-sp/
						(VNDK-SP libraries, e.g. libc++.so)
			arch-{TARGET_2ND_ARCH}-{TARGET_2ND_ARCH_VARIANT}/
				shared/
					vndk-core/
						(VNDK-core libraries, e.g. libbinder.so)
					vndk-sp/
						(VNDK-SP libraries, e.g. libc++.so)
			binder32/
				(This directory is newly introduced in v28 (Android P) to hold
				prebuilts built for 32-bit binder interface.)
				arch-{TARGET_ARCH}-{TARGE_ARCH_VARIANT}/
					...
			configs/
				(various *.txt configuration files)
			include/
				(header files of same directory structure with source tree)
			NOTICE_FILES/
				(notice files of libraries, e.g. libcutils.so.txt)
	*/

	snapshotDir := "vndk-snapshot"
	snapshotArchDir := filepath.Join(snapshotDir, ctx.DeviceConfig().DeviceArch())

	targetArchDirMap := make(map[android.ArchType]string)
	for _, target := range ctx.Config().Targets[android.Android] {
		dir := snapshotArchDir
		if ctx.DeviceConfig().BinderBitness() == "32" {
			dir = filepath.Join(dir, "binder32")
		}
		arch := "arch-" + target.Arch.ArchType.String()
		if target.Arch.ArchVariant != "" {
			arch += "-" + target.Arch.ArchVariant
		}
		dir = filepath.Join(dir, arch)
		targetArchDirMap[target.Arch.ArchType] = dir
	}
	configsDir := filepath.Join(snapshotArchDir, "configs")
	noticeDir := filepath.Join(snapshotArchDir, "NOTICE_FILES")
	includeDir := filepath.Join(snapshotArchDir, "include")

	// set of include paths exported by VNDK libraries
	exportedIncludes := make(map[string]bool)

	// generated header files among exported headers.
	var generatedHeaders android.Paths

	// Actual output library name of modules. It might differ from a module name because of stem and suffix
	moduleStems := make(map[string]string)

	// set of notice files copied.
	noticeBuilt := make(map[string]bool)

	installSnapshotFileFromPath := func(path android.Path, out string) android.OutputPath {
		outPath := android.PathForOutput(ctx, out)
		ctx.Build(pctx, android.BuildParams{
			Rule:        android.Cp,
			Input:       path,
			Output:      outPath,
			Description: "vndk snapshot " + out,
			Args: map[string]string{
				"cpFlags": "-f -L",
			},
		})
		return outPath
	}

	installSnapshotFileFromContent := func(content, out string) android.OutputPath {
		outPath := android.PathForOutput(ctx, out)
		ctx.Build(pctx, android.BuildParams{
			Rule:        android.WriteFile,
			Output:      outPath,
			Description: "vndk snapshot " + out,
			Args: map[string]string{
				"content": content,
			},
		})
		return outPath
	}

	type vndkSnapshotLibraryInterface interface {
		exportedFlagsProducer
		libraryInterface
	}

	var _ vndkSnapshotLibraryInterface = (*prebuiltLibraryLinker)(nil)
	var _ vndkSnapshotLibraryInterface = (*libraryDecorator)(nil)

	installVndkSnapshotLib := func(m *Module, l vndkSnapshotLibraryInterface, vndkType string) (android.Paths, bool) {
		targetArchDir, ok := targetArchDirMap[m.Target().Arch.ArchType]
		if !ok {
			return nil, false
		}

		var ret android.Paths

		libPath := m.outputFile.Path()
		stem := libPath.Base()
		snapshotLibOut := filepath.Join(targetArchDir, "shared", vndkType, stem)
		ret = append(ret, installSnapshotFileFromPath(libPath, snapshotLibOut))

		moduleName := ctx.ModuleName(m)
		if s, ok := moduleStems[moduleName]; ok && s != stem {
			ctx.Errorf("stems of module %q are different among arch", moduleName)
			return nil, false
		}
		moduleStems[moduleName] = stem

		if m.NoticeFile().Valid() {
			noticeName := stem + ".txt"
			// skip already copied notice file
			if _, ok := noticeBuilt[noticeName]; !ok {
				noticeBuilt[noticeName] = true
				ret = append(ret, installSnapshotFileFromPath(
					m.NoticeFile().Path(), filepath.Join(noticeDir, noticeName)))
			}
		}

		if ctx.Config().VndkSnapshotBuildArtifacts() {
			prop := struct {
				ExportedDirs        []string `json:",omitempty"`
				ExportedSystemDirs  []string `json:",omitempty"`
				ExportedFlags       []string `json:",omitempty"`
				RelativeInstallPath string   `json:",omitempty"`
			}{}
			prop.ExportedFlags = l.exportedFlags()
			prop.ExportedDirs = l.exportedDirs().Strings()
			prop.ExportedSystemDirs = l.exportedSystemDirs().Strings()
			prop.RelativeInstallPath = m.RelativeInstallPath()

			propOut := snapshotLibOut + ".json"

			j, err := json.Marshal(prop)
			if err != nil {
				ctx.Errorf("json marshal to %q failed: %#v", propOut, err)
				return nil, false
			}
			ret = append(ret, installSnapshotFileFromContent(string(j), propOut))
		}
		return ret, true
	}

	isVndkSnapshotLibrary := func(m *Module) (i vndkSnapshotLibraryInterface, vndkType string, isVndkSnapshotLib bool) {
		if m.Target().NativeBridge == android.NativeBridgeEnabled {
			return nil, "", false
		}
		if !m.UseVndk() || !m.IsForPlatform() || !m.installable() {
			return nil, "", false
		}
		l, ok := m.linker.(vndkSnapshotLibraryInterface)
		if !ok || !l.shared() {
			return nil, "", false
		}
		if m.vndkVersion() == ctx.DeviceConfig().PlatformVndkVersion() && m.IsVndk() && !m.isVndkExt() {
			if m.isVndkSp() {
				return l, "vndk-sp", true
			} else {
				return l, "vndk-core", true
			}
		}

		return nil, "", false
	}

	ctx.VisitAllModules(func(module android.Module) {
		m, ok := module.(*Module)
		if !ok || !m.Enabled() {
			return
		}

		l, vndkType, ok := isVndkSnapshotLibrary(m)
		if !ok {
			return
		}

		libs, ok := installVndkSnapshotLib(m, l, vndkType)
		if !ok {
			return
		}

		snapshotOutputs = append(snapshotOutputs, libs...)

		// We glob headers from include directories inside source tree. So we first gather
		// all include directories inside our source tree. On the contrast, we manually
		// collect generated headers from dependencies as they can't globbed.
		generatedHeaders = append(generatedHeaders, l.exportedDeps()...)
		for _, dir := range append(l.exportedDirs(), l.exportedSystemDirs()...) {
			exportedIncludes[dir.String()] = true
		}
	})

	if ctx.Config().VndkSnapshotBuildArtifacts() {
		globbedHeaders := make(map[string]bool)

		for _, dir := range android.SortedStringKeys(exportedIncludes) {
			// Skip if dir is for generated headers
			if strings.HasPrefix(dir, android.PathForOutput(ctx).String()) {
				continue
			}
			exts := headerExts
			// Glob all files under this special directory, because of C++ headers.
			if strings.HasPrefix(dir, "external/libcxx/include") {
				exts = []string{""}
			}
			for _, ext := range exts {
				glob, err := ctx.GlobWithDeps(dir+"/**/*"+ext, nil)
				if err != nil {
					ctx.Errorf("%#v\n", err)
					return
				}
				for _, header := range glob {
					if strings.HasSuffix(header, "/") {
						continue
					}
					globbedHeaders[header] = true
				}
			}
		}

		for _, header := range android.SortedStringKeys(globbedHeaders) {
			snapshotOutputs = append(snapshotOutputs, installSnapshotFileFromPath(
				android.PathForSource(ctx, header), filepath.Join(includeDir, header)))
		}

		isHeader := func(path string) bool {
			for _, ext := range headerExts {
				if strings.HasSuffix(path, ext) {
					return true
				}
			}
			return false
		}

		// For generated headers, manually install one by one, rather than glob
		for _, path := range android.PathsToDirectorySortedPaths(android.FirstUniquePaths(generatedHeaders)) {
			header := path.String()

			if !isHeader(header) {
				continue
			}

			snapshotOutputs = append(snapshotOutputs, installSnapshotFileFromPath(
				path, filepath.Join(includeDir, header)))
		}
	}

	snapshotOutputs = append(snapshotOutputs,
		installSnapshotFileFromPath(android.PathForOutput(ctx, "vndk", "vndkcore.libraries.txt"),
			filepath.Join(configsDir, "vndkcore.libraries.txt")),
		installSnapshotFileFromPath(android.PathForOutput(ctx, "vndk", "vndkprivate.libraries.txt"),
			filepath.Join(configsDir, "vndkprivate.libraries.txt")),
		installSnapshotFileFromPath(android.PathForOutput(ctx, "vndk", "vndksp.libraries.txt"),
			filepath.Join(configsDir, "vndksp.libraries.txt")),
		installSnapshotFileFromPath(android.PathForOutput(ctx, "vndk", "llndk.libraries.txt"),
			filepath.Join(configsDir, "llndk.libraries.txt")),
	)

	/*
		module_paths.txt contains paths on which VNDK modules are defined.
		e.g.,
			libbase.so system/core/base
			libc.so bionic/libc
			...
	*/
	var modulePathTxtBuilder strings.Builder
	modulePaths := modulePaths(ctx.Config())
	for idx, lib := range android.SortedStringKeys(modulePaths) {
		if idx > 0 {
			modulePathTxtBuilder.WriteString("\\n")
		}
		modulePathTxtBuilder.WriteString(moduleStems[lib])
		modulePathTxtBuilder.WriteString(" ")
		modulePathTxtBuilder.WriteString(modulePaths[lib])
	}
	snapshotOutputs = append(snapshotOutputs, installSnapshotFileFromContent(
		modulePathTxtBuilder.String(), filepath.Join(configsDir, "module_paths.txt")))

	/*
		module_names.txt contains names as which VNDK modules are defined,
		because output filename and module name can be different with stem and suffix properties.

		e.g.,
			libcutils.so libcutils
			libprotobuf-cpp-full-3.9.2.so libprotobuf-cpp-full
			...
	*/
	var moduleNamesTxtBuilder strings.Builder
	for idx, lib := range android.SortedStringKeys(moduleStems) {
		if idx > 0 {
			moduleNamesTxtBuilder.WriteString("\\n")
		}
		moduleNamesTxtBuilder.WriteString(moduleStems[lib])
		moduleNamesTxtBuilder.WriteString(" ")
		moduleNamesTxtBuilder.WriteString(lib)
	}
	snapshotOutputs = append(snapshotOutputs, installSnapshotFileFromContent(
		moduleNamesTxtBuilder.String(), filepath.Join(configsDir, "module_names.txt")))

	// All artifacts are ready. Sort them to normalize ninja and then zip.
	sort.Slice(snapshotOutputs, func(i, j int) bool {
		return snapshotOutputs[i].String() < snapshotOutputs[j].String()
	})

	zipPath := android.PathForOutput(ctx, snapshotDir, "android-vndk-"+ctx.DeviceConfig().DeviceArch()+".zip")
	zipRule := android.NewRuleBuilder()

	// If output files are too many, soong_zip command can exceed ARG_MAX.
	// So first dump file lists into a single list file, and then feed it to Soong
	snapshotOutputList := android.PathForOutput(ctx, snapshotDir, "android-vndk-"+ctx.DeviceConfig().DeviceArch()+"_list")
	zipRule.Command().
		Text("( xargs").
		FlagWithRspFileInputList("-n1 echo < ", snapshotOutputs).
		FlagWithOutput("| tr -d \\' > ", snapshotOutputList).
		Text(")")

	zipRule.Temporary(snapshotOutputList)

	zipRule.Command().
		BuiltTool(ctx, "soong_zip").
		FlagWithOutput("-o ", zipPath).
		FlagWithArg("-C ", android.PathForOutput(ctx, snapshotDir).String()).
		FlagWithInput("-l ", snapshotOutputList)

	zipRule.Build(pctx, ctx, zipPath.String(), "vndk snapshot "+zipPath.String())
	*vndkSnapshotZip(ctx.Config()) = android.OptionalPathForPath(zipPath)
}

func installListFile(ctx android.SingletonContext, list []string, pathComponents ...string) android.OutputPath {
	out := android.PathForOutput(ctx, pathComponents...)
	ctx.Build(pctx, android.BuildParams{
		Rule:        android.WriteFile,
		Output:      out,
		Description: "Writing " + out.String(),
		Args: map[string]string{
			"content": strings.Join(list, "\\n"),
		},
	})
	return out
}

func (c *vndkSnapshotSingleton) buildVndkLibrariesTxtFiles(ctx android.SingletonContext) {
	var (
		llndk, vndkcore, vndksp, vndkprivate, vndkcorevariant, merged []string
	)
	vndkVersion := ctx.DeviceConfig().PlatformVndkVersion()
	config := ctx.Config()
	ctx.VisitAllModules(func(m android.Module) {
		if !m.Enabled() {
			return
		}
		c, ok := m.(*Module)
		if !ok || c.Os().Class != android.Device {
			return
		}
		lib, ok := c.linker.(interface{ shared() bool })
		if !ok || !lib.shared() {
			return
		}

		if !c.OutputFile().Valid() {
			return
		}

		filename := c.OutputFile().Path().Base()
		if c.isLlndk(config) {
			llndk = append(llndk, filename)
			if c.isVndkPrivate(config) {
				vndkprivate = append(vndkprivate, filename)
			}
		} else if c.vndkVersion() == vndkVersion && c.IsVndk() && !c.isVndkExt() {
			if c.isVndkSp() {
				vndksp = append(vndksp, filename)
			} else {
				vndkcore = append(vndkcore, filename)
			}
			if c.isVndkPrivate(config) {
				vndkprivate = append(vndkprivate, filename)
			}
			if ctx.DeviceConfig().VndkUseCoreVariant() && !c.MustUseVendorVariant() {
				vndkcorevariant = append(vndkcorevariant, filename)
			}
		}
	})
	llndk = android.SortedUniqueStrings(llndk)
	vndkcore = android.SortedUniqueStrings(vndkcore)
	vndksp = android.SortedUniqueStrings(vndksp)
	vndkprivate = android.SortedUniqueStrings(vndkprivate)
	vndkcorevariant = android.SortedUniqueStrings(vndkcorevariant)

	installListFile(ctx, llndk, "vndk", "llndk.libraries.txt")
	installListFile(ctx, vndkcore, "vndk", "vndkcore.libraries.txt")
	installListFile(ctx, vndksp, "vndk", "vndksp.libraries.txt")
	installListFile(ctx, vndkprivate, "vndk", "vndkprivate.libraries.txt")
	installListFile(ctx, vndkcorevariant, "vndk", "vndkcorevariant.libraries.txt")

	// merged & tagged & filtered-out(libclang_rt)
	filterOutLibClangRt := func(libList []string) (filtered []string) {
		for _, lib := range libList {
			if !strings.HasPrefix(lib, "libclang_rt.") {
				filtered = append(filtered, lib)
			}
		}
		return
	}
	merged = append(merged, addPrefix(filterOutLibClangRt(llndk), "LLNDK: ")...)
	merged = append(merged, addPrefix(vndksp, "VNDK-SP: ")...)
	merged = append(merged, addPrefix(filterOutLibClangRt(vndkcore), "VNDK-core: ")...)
	merged = append(merged, addPrefix(vndkprivate, "VNDK-private: ")...)

	installListFile(ctx, merged, "vndk", "vndk.libraries.txt")
}
