// Copyright 2020 The Android Open Source Project
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

import "android/soong/android"

type recoverySnapshotImage struct{}

func (recoverySnapshotImage) Init(ctx android.RegistrationContext) {
	ctx.RegisterSingletonType("recovery-snapshot", RecoverySnapshotSingleton)
}

func (recoverySnapshotImage) shouldGenerateSnapshot(ctx android.SingletonContext) bool {
	// RECOVERY_SNAPSHOT_VERSION must be set to 'current' in order to generate a
	// snapshot.
	return ctx.DeviceConfig().RecoverySnapshotVersion() == "current"
}

func (recoverySnapshotImage) InImage(m SnapshotModuleInterface) func() bool {
	return m.InRecovery
}

// recovery snapshot does not have private libraries.
func (recoverySnapshotImage) Private(m SnapshotModuleInterface) bool {
	return false
}

func (recoverySnapshotImage) IsProprietaryPath(dir string, deviceConfig android.DeviceConfig) bool {
	return isDirectoryExcluded(dir, deviceConfig.RecoverySnapshotDirsExcludedMap(), deviceConfig.RecoverySnapshotDirsIncludedMap())
}

// recovery snapshot does NOT treat vndk specially.
func (recoverySnapshotImage) IncludeVndk() bool {
	return false
}

func (recoverySnapshotImage) ExcludeFromSnapshot(m SnapshotModuleInterface) bool {
	return m.ExcludeFromRecoverySnapshot()
}

func (recoverySnapshotImage) IsUsingSnapshot(cfg android.DeviceConfig) bool {
	recoverySnapshotVersion := cfg.RecoverySnapshotVersion()
	return recoverySnapshotVersion != "current" && recoverySnapshotVersion != ""
}

func (recoverySnapshotImage) TargetSnapshotVersion(cfg android.DeviceConfig) string {
	return cfg.RecoverySnapshotVersion()
}

func (recoverySnapshotImage) ExcludeFromDirectedSnapshot(cfg android.DeviceConfig, name string) bool {
	// If we're using full snapshot, not directed snapshot, capture every module
	if !cfg.DirectedRecoverySnapshot() {
		return false
	}
	// Else, checks if name is in RECOVERY_SNAPSHOT_MODULES.
	return !cfg.RecoverySnapshotModules()[name]
}

func (recoverySnapshotImage) ImageVariantName(cfg android.DeviceConfig) string {
	return android.RecoveryVariation
}

func (recoverySnapshotImage) ModuleNameSuffix() string {
	return ".recovery"
}

var RecoverySnapshotImageSingleton recoverySnapshotImage

func init() {
	RecoverySnapshotImageSingleton.Init(android.InitRegistrationContext)
}
