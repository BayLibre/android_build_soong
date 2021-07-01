// Copyright (C) 2021 The Android Open Source Project
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
	"android/soong/android"
	"encoding/json"
	"path/filepath"
	"sort"
)

func init() {
	registerHostSnapshotComponents(android.InitRegistrationContext)
}

func registerHostSnapshotComponents(ctx android.RegistrationContext) {
	ctx.RegisterSingletonType("host-snapshot", HostToolsAndroidSingleton)
}

const (
	modBinary        = iota
	modSharedLibrary = iota
)

type hostModule struct {
	module      android.Module
	moduleType  int                  // Module type (modBinary, modSharedLibrary)
	installPath string               // install path into the snapshot
	path        android.OptionalPath // local path of module
}

func (mod *hostModule) ModuleName(ctx android.SingletonContext) string {
	return ctx.ModuleName(mod.module)

}

// Get any modules this host module is dependent upon by name, the module
// type has already been validated when creating the HostModule
func (mod *hostModule) requiredModules() []string {
	return append(mod.module.HostRequiredModuleNames(), mod.module.RequiredModuleNames()...)
}

type hostSnapshotSingleton struct {
	snapshotDir string
	noticeDir   string

	manifestName string
	makeVar      string

	noticePaths     android.Paths
	snapshotMap     map[string]bool
	snapshotModules []*hostModule
	zipFile         android.OptionalPath

	notices map[string]bool
}

func HostToolsAndroidSingleton() android.Singleton {
	singleton := &hostSnapshotSingleton{}
	singleton.init()
	return singleton
}

func (c *hostSnapshotSingleton) init() {
	c.manifestName = "host_tools.json"
	c.snapshotDir = "host-snapshot"
	c.noticeDir = filepath.Join(c.snapshotDir, "NOTICE_FILES")

	c.makeVar = "SOONG_HOST_SNAPSHOT_ZIP"

	c.snapshotMap = make(map[string]bool) // Modules that are part of the snapshot

	c.notices = make(map[string]bool)
}

func (c *hostSnapshotSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	modNames := ctx.DeviceConfig().HostSnapshotModules()
	if len(modNames) == 0 {
		return
	}
	mods := make(map[string]bool)
	for _, j := range modNames {
		mods[j] = false
	}

	ctx.VisitAllModules(func(module android.Module) {
		name := ctx.ModuleName(module)
		if _, ok := mods[name]; ok {
			// Validate createHostBin only returns a single variant for a module name
			//  by creating the binary and then checking that the name has not
			//  already been resolved.
			if mod, ok := createHostBin(ctx, module); ok {
				if mods[name] {
					ctx.Errorf("Duplicate host binary variant found for %s", name)
				} else {
					c.resolveModule(ctx, mod)
					mods[name] = true
				}
			}
		}

	})
	// Validate all host snapshot modules have been resolved
	for name, resolved := range mods {
		if !resolved {
			ctx.Errorf("Host module %s not found", name)
		}
	}
	c.build(ctx)
}

func (c *hostSnapshotSingleton) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict(
		c.makeVar,
		c.zipFile.String(),
	)
}

func (c *hostSnapshotSingleton) addNoticeFiles(ctx android.SingletonContext, mod android.Module) {
	// Build notice copy rule if we haven't already for this module
	if len(mod.EffectiveLicenseFiles()) > 0 {
		noticeFile := filepath.Join(c.noticeDir, ctx.ModuleName(mod)+".txt")
		if !c.notices[noticeFile] {
			c.notices[noticeFile] = true
			c.noticePaths = append(c.noticePaths, combineNoticesRule(ctx, mod.EffectiveLicenseFiles(), noticeFile))
		}
	}
}

func (c *hostSnapshotSingleton) resolveModule(ctx android.SingletonContext, mod *hostModule) {
	// Check if module has been resolved
	if _, ret := c.snapshotMap[mod.ModuleName(ctx)]; ret {
		return
	}
	// Add module, mark as resolved
	c.snapshotMap[mod.ModuleName(ctx)] = true
	c.snapshotModules = append(c.snapshotModules, mod)

	ctx.VisitDirectDeps(mod.module, func(dep android.Module) {
		if depMod, ok := createHostModule(ctx, dep); ok {
			c.resolveModule(ctx, depMod)
		} else {
			c.addNoticeFiles(ctx, dep)
		}
	})
	c.addNoticeFiles(ctx, mod.module)
}

type hostJsonFlags struct {
	snapshotJsonFlags
	ModuleStemName string `json`
	InstallPath    string `json`
}

// Build host snapshot rules
func (c *hostSnapshotSingleton) build(ctx android.SingletonContext) {
	// Sort notice paths and snapshot modules to produce repeatable build
	sort.Slice(c.noticePaths, func(i, j int) bool {
		return (c.noticePaths[i].String() < c.noticePaths[j].String())
	})
	sort.Slice(c.snapshotModules, func(i, j int) bool {
		return (c.snapshotModules[i].ModuleName(ctx) < c.snapshotModules[j].ModuleName(ctx))
	})

	snapshotOutputPaths := c.noticePaths
	type jsonData struct {
		Bin []hostJsonFlags `json:"bin"`
	}
	var oJson jsonData
	var manOut = filepath.Join(c.snapshotDir, c.manifestName)
	for _, mod := range c.snapshotModules {
		paths, json := mod.build(ctx, c.snapshotDir)

		if json != nil {
			oJson.Bin = append(oJson.Bin, *json)
		}

		snapshotOutputPaths = append(snapshotOutputPaths, paths...)

	}

	if marsh, err := json.Marshal(oJson); err != nil {
		ctx.Errorf("host_snapshot json marshal failure: %#v", err)
	} else {
		snapshotOutputPaths = append(snapshotOutputPaths, writeStringToFileRule(ctx, string(marsh), manOut))
	}

	c.zipFile = zipSnapshot(ctx, c.snapshotDir, c.snapshotDir, snapshotOutputPaths)

}

// Create a host module from a given android module
func createHostBin(ctx android.SingletonContext, module android.Module) (*hostModule, bool) {

	out := &hostModule{
		module:      module,
		moduleType:  modBinary,
		installPath: "bin/",
	}
	if !module.Enabled() || module.IsHideFromMake() {
		return out, false
	}
	if module.Target().Os != android.BuildOs {
		return out, false
	}
	if android.IsModulePrebuilt(module) {
		return out, false
	}
	if ctx.PrimaryModule(module) != module {
		return out, false
	}
	switch t := module.(type) {
	case android.HostToolProvider:
		if t.HostToolPath().Valid() {
			out.path = t.HostToolPath()
		}
	}
	// Some variants provide a valid empty string, discard these here
	if out.path.Valid() && out.path.String() != "" {
		return out, true
	}
	return out, false
}

// Create a host module from a given android module
//   this only supports shared CC shared libraries will need to expand to handle more modules
func createHostModule(ctx android.SingletonContext, module android.Module) (*hostModule, bool) {
	libPath := "lib/"
	if module.Target().Arch.ArchType.Multilib == "lib64" {
		libPath = "lib64/"
	}
	out := &hostModule{
		module:      module,
		moduleType:  modSharedLibrary,
		installPath: libPath,
	}
	valid := false
	if android.IsModulePrebuilt(module) {
		return out, valid
	}
	if cmod, ok := module.(LinkableInterface); ok {
		if cmod.CcLibrary() && cmod.Shared() && cmod.OutputFile().Valid() {
			out.path = cmod.OutputFile()
			valid = true
		}
	}
	return out, valid
}

// Get the build command(s) for a host module, along with optional json description
func (mod *hostModule) build(ctx android.SingletonContext, snapshotDir string) (android.Paths, *hostJsonFlags) {
	localPath := filepath.Join(snapshotDir, mod.installPath)

	var jsonData *hostJsonFlags = nil
	var paths android.Paths

	if mod.moduleType == modBinary {
		reqMods := mod.requiredModules()
		baseData := &snapshotJsonFlags{ModuleName: mod.ModuleName(ctx), Required: reqMods}
		jsonData = &hostJsonFlags{snapshotJsonFlags: *baseData,
			ModuleStemName: mod.path.Path().Base(),
			InstallPath:    mod.installPath,
		}

	}

	paths = append(paths, copyFileRule(ctx, mod.path.Path(), filepath.Join(localPath, mod.path.Path().Base())))

	return paths, jsonData

}
