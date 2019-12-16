package sdk

import "android/soong/android"

func init() {
	android.RegisterModuleType("module_exports", ModuleExportsFactory)
	android.RegisterModuleType("module_exports_snapshot", ModuleExportsSnapshotsFactory)
}

func ModuleExportsFactory() android.Module {
	s := newSdkModule()
	s.properties.Module_exports = true
	return s
}

func ModuleExportsSnapshotsFactory() android.Module {
	s := newSdkModule()
	s.properties.Snapshot = true
	s.properties.Module_exports = true
	return s
}
