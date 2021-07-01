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

	noticePaths android.Paths
	snapshotMap map[string]*hostModule

	zipFile android.OptionalPath

	notices     map[string]bool
	moduleMap   map[string]*hostModule
	reqModNames []string
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
	c.moduleMap = make(map[string]*hostModule)   // Discovered modules
	c.snapshotMap = make(map[string]*hostModule) // Modules that are part of the snapshot
	c.notices = make(map[string]bool)
}

func (c *hostSnapshotSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	c.reqModNames = ctx.DeviceConfig().HostSnapshotModules()
	if 0 == len(c.reqModNames) {
		return
	}

	// Populate map of all host binaries
	ctx.VisitAllModules(func(module android.Module) {
		if mod, ok := createHostBin(module); ok {
			if _, ok := c.moduleMap[module.Name()]; !ok {
				c.moduleMap[module.Name()] = mod
			}
		}

	})

	c.resolveModules(ctx)
	c.build(ctx)
}

func (c *hostSnapshotSingleton) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict(
		c.makeVar,
		c.zipFile.String(),
	)
}

func (c *hostSnapshotSingleton) addModule(mod *hostModule) {
	c.snapshotMap[mod.module.Name()] = mod
}

func (c *hostSnapshotSingleton) addNoticeFiles(ctx android.SingletonContext, mod android.Module) {
	// Build notice copy rule if we haven't already for this module
	if len(mod.EffectiveLicenseFiles()) > 0 {
		noticeFile := filepath.Join(c.noticeDir, mod.Name()+".txt")
		if !c.notices[noticeFile] {
			c.notices[noticeFile] = true
			c.noticePaths = append(c.noticePaths, combineNoticesRule(ctx, mod.EffectiveLicenseFiles(), noticeFile))
		}
	}
}

// Resolve all modules in reqModNames
func (c *hostSnapshotSingleton) resolveModules(ctx android.SingletonContext) {

	resolvedNames := make(map[string]bool)

	for len(c.reqModNames) > 0 {
		// Pop name and resolve dependencies if not already resolved
		l := len(c.reqModNames)
		name := c.reqModNames[l-1]
		c.reqModNames = c.reqModNames[:l-1]

		if !resolvedNames[name] {
			if mod, ok := c.moduleMap[name]; ok {
				c.addModule(mod)
				// Required modules are not part of direct dependents, add to stack by name
				c.reqModNames = append(c.reqModNames, mod.requiredModules()...)
				ctx.VisitDirectDeps(mod.module, func(dep android.Module) {
					if depMod, ok := createHostModule(ctx, dep); ok {
						c.addModule(depMod)
						c.moduleMap[dep.Name()] = depMod
						c.reqModNames = append(c.reqModNames, dep.Name())
					}
					c.addNoticeFiles(ctx, dep)
				})
				c.addNoticeFiles(ctx, mod.module)
			}
			resolvedNames[name] = true
		}

	}

}

type hostJsonFlags struct {
	snapshotJsonFlags
	ModuleStemName string `json`
	InstallPath    string `json`
}

// Build host snapshot rules
func (c *hostSnapshotSingleton) build(ctx android.SingletonContext) {
	snapshotOutputPaths := c.noticePaths
	type json_data struct {
		Bin []hostJsonFlags `json:"bin"`
	}
	oJson := json_data{}
	var manOut = filepath.Join(c.snapshotDir, c.manifestName)

	for _, mod := range c.snapshotMap {
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
func createHostBin(module android.Module) (*hostModule, bool) {

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
//   this only supports shared CC shared libraries will need to expand to handle prebuilt_etc...
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
			valid = out.path.String() != ""

		}
	}
	return out, valid
}

// Get the build command(s) for a host module, along with optional json description
func (mod *hostModule) build(ctx android.SingletonContext, snapshotDir string) (android.Paths, *hostJsonFlags) {
	localPath := filepath.Join(snapshotDir, mod.installPath)

	var json_data *hostJsonFlags = nil
	paths := android.Paths{}

	if mod.moduleType == modBinary {
		reqMods := mod.requiredModules()
		if reqMods == nil {
			reqMods = []string{}
		}
		base_data := &snapshotJsonFlags{ModuleName: mod.module.Name(), Required: reqMods}
		json_data = &hostJsonFlags{snapshotJsonFlags: *base_data,
			ModuleStemName: mod.path.Path().Base(),
			InstallPath:    mod.installPath,
		}

	}

	paths = append(paths, copyFileRule(ctx, mod.path.Path(), filepath.Join(localPath, mod.path.Path().Base())))

	return paths, json_data

}
