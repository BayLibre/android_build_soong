package etc

import (
	"android/soong/android"
	"path/filepath"
	"strings"
)

func init() {
	RegisterInstallSymlinkBuildComponents(android.InitRegistrationContext)
}

func RegisterInstallSymlinkBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("install_symlink", InstallSymlinkFactory)
}

// prebuilt_etc is for a prebuilt artifact that is installed in
// <partition>/etc/<sub_dir> directory.
func InstallSymlinkFactory() android.Module {
	module := &InstallSymlink{}
	module.AddProperties(&module.properties)
	android.InitAndroidMultiTargetsArchModule(module, android.DeviceSupported, android.MultilibCommon)
	return module
}

type InstallSymlinkProperties struct {
	// Where to install this symlink, relative to the partition it's installed on.
	// Which partition it's installed on can be controlled by the vendor, system_ext, ramdisk, etc.
	// properties.
	Installed_location string
	// The target of the symlink, aka where the symlink points.
	Symlink_target string
}

type InstallSymlink struct {
	android.ModuleBase
	properties InstallSymlinkProperties

	output        android.Path
	installedPath android.InstallPath
}

func (m *InstallSymlink) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if filepath.Clean(m.properties.Symlink_target) != m.properties.Symlink_target {
		ctx.PropertyErrorf("Symlink_target", "Should be a clean filepath")
		return
	}
	if filepath.Clean(m.properties.Installed_location) != m.properties.Installed_location {
		ctx.PropertyErrorf("installed_location", "Should be a clean filepath")
		return
	}
	if strings.HasPrefix(m.properties.Installed_location, "../") || strings.HasPrefix(m.properties.Installed_location, "/") {
		ctx.PropertyErrorf("installed_location", "Should not start with / or ../")
		return
	}

	out := android.PathForModuleOut(ctx, "out.txt")
	android.WriteFileRuleVerbatim(ctx, out, "")
	m.output = out

	name := filepath.Base(m.properties.Installed_location)
	installDir := android.PathForModuleInstall(ctx, filepath.Dir(m.properties.Installed_location))
	m.installedPath = ctx.InstallAbsoluteSymlink(installDir, name, m.properties.Symlink_target)
}

func (m *InstallSymlink) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{{
		Class: "FAKE",
		// Need at least one output file in order for this to take effect.
		OutputFile: android.OptionalPathForPath(m.output),
		Include:    "$(BUILD_PHONY_PACKAGE)",
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				entries.AddStrings("LOCAL_SOONG_INSTALL_SYMLINKS", m.installedPath.String())
			},
		},
	}}
}
