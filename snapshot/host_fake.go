// Copyright 2021 The Android Open Source Project
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

package snapshot

import (
	"encoding/json"
	"path/filepath"

	"android/soong/android"
)

// The host-fake-snapshot is a singleton module that will find all host tools
// in the source tree and add a 'fake' version to a snapshot along with a
// JSON meta file.
//
// Similar to the other snapshots (vendor, recovery,..) a round trip method
// is used to determine which modules to include in the actual snapshots.
// The fake host snapshot is created from a full source tree, the snapshot
// is then installed into the minimal source tree to identify which modules
// are needed.  The modules needed are then used to define the actual
// host_snaphot deps modules. See development/vendor_snapshot/update.py
// for more information.

func init() {
	registerHostSnapshotComponents(android.InitRegistrationContext)
}

func registerHostSnapshotComponents(ctx android.RegistrationContext) {
	ctx.RegisterSingletonType("host-fake-snapshot", HostToolsFakeAndroidSingleton)
}

type hostFakeSingleton struct {
	snapshotDir string
	makeVar     string
	zipFile     android.OptionalPath
}

func (c *hostFakeSingleton) init() {
	c.snapshotDir = "host-fake-snapshot"
	c.makeVar = "SOONG_HOST_FAKE_SNAPSHOT_ZIP"

}
func HostToolsFakeAndroidSingleton() android.Singleton {
	singleton := &hostFakeSingleton{}
	singleton.init()
	return singleton
}

func (c *hostFakeSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if !ctx.DeviceConfig().HostFakeSnapshotEnabled() {
		return
	}
	// Find all host binary modules add 'fake' versions to snapshot
	var outputs android.Paths
	seen := make(map[string]bool)
	var jsonData []SnapshotJsonFlags
	ctx.VisitAllModules(func(module android.Module) {
		if module.Target().Os != ctx.Config().BuildOSTarget.Os {
			return
		}
		if module.Target().Arch.ArchType != ctx.Config().BuildOSTarget.Arch.ArchType {
			return
		}

		if android.IsModulePrebuilt(module) {
			return
		}

		if !module.Enabled() || module.IsHideFromMake() {
			return
		}
		apexInfo := ctx.ModuleProvider(module, android.ApexInfoProvider).(android.ApexInfo)
		if !apexInfo.IsForPlatform() {
			return
		}
		path := hostBinToolPath(module)
		if path.Valid() && path.String() != "" {
			outFile := filepath.Join(c.snapshotDir, path.String())
			if !seen[outFile] {
				seen[outFile] = true
				outputs = append(outputs, WriteStringToFileRule(ctx, "", outFile))
				jsonData = append(jsonData, *hostBinJsonDesc(module))
			}
		}
	})

	marsh, err := json.Marshal(jsonData)
	if err != nil {
		ctx.Errorf("host fake snapshot json marshal failure: %#v", err)
		return
	}
	outputs = append(outputs, WriteStringToFileRule(ctx, string(marsh), filepath.Join(c.snapshotDir, "host_snapshot.json")))
	c.zipFile = zipSnapshot(ctx, c.snapshotDir, c.snapshotDir, outputs)

}
func (c *hostFakeSingleton) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict(
		c.makeVar,
		c.zipFile.String(),
	)
}
