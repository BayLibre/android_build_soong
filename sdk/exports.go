package sdk

import "android/soong/android"

func init() {
	android.RegisterModuleType("module_exports", ModuleExportsFactory)
	android.RegisterModuleType("module_exports_snapshot", ModuleExportsSnapshotsFactory)
}

// module_exports defines the exports of a mainline module. The exports are Soong modules
// which are required by Soong modules that are not part of the mainline module.
func ModuleExportsFactory() android.Module {
	return newSdkModule(true)
}

// module_exports_snapshot is a versioned snapshot of prebuilt versions of all the exports
// of a mainline module.
func ModuleExportsSnapshotsFactory() android.Module {
	s := newSdkModule(true)
	s.properties.Snapshot = true
	return s
}
