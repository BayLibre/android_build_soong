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

type RecoverySnapshotImage struct{}

func (RecoverySnapshotImage) Init(ctx android.RegistrationContext) {
	ctx.RegisterSingletonType("recovery-snapshot", RecoverySnapshotSingleton)
}

func (RecoverySnapshotImage) shouldGenerateSnapshot(ctx android.SingletonContext) bool {
	// RECOVERY_SNAPSHOT_VERSION must be set to 'current' in order to generate a
	// snapshot.
	return ctx.DeviceConfig().RecoverySnapshotVersion() == "current"
}

func (RecoverySnapshotImage) InImage(m SnapshotModuleInterface) func() bool {
	return m.InRecovery
}

// recovery snapshot does not have private libraries.
func (RecoverySnapshotImage) Private(m SnapshotModuleInterface) bool {
	return false
}

func (RecoverySnapshotImage) IsProprietaryPath(dir string, deviceConfig android.DeviceConfig) bool {
	return isDirectoryExcluded(dir, deviceConfig.RecoverySnapshotDirsExcludedMap(), deviceConfig.RecoverySnapshotDirsIncludedMap())
}

func (RecoverySnapshotImage) ExcludeFromSnapshot(m SnapshotModuleInterface) bool {
	return m.ExcludeFromRecoverySnapshot()
}

func (RecoverySnapshotImage) IsUsingSnapshot(cfg android.DeviceConfig) bool {
	recoverySnapshotVersion := cfg.RecoverySnapshotVersion()
	return recoverySnapshotVersion != "current" && recoverySnapshotVersion != ""
}

func (RecoverySnapshotImage) TargetSnapshotVersion(cfg android.DeviceConfig) string {
	return cfg.RecoverySnapshotVersion()
}

func (RecoverySnapshotImage) ExcludeFromDirectedSnapshot(cfg android.DeviceConfig, name string) bool {
	// If we're using full snapshot, not directed snapshot, capture every module
	if !cfg.DirectedRecoverySnapshot() {
		return false
	}
	// Else, checks if name is in RECOVERY_SNAPSHOT_MODULES.
	return !cfg.RecoverySnapshotModules()[name]
}

var RecoverySnapshotImageSingleton RecoverySnapshotImage

func init() {
	RecoverySnapshotImageSingleton.Init(android.InitRegistrationContext)
}
