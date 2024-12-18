package android

import (
	"fmt"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func init() {
	InitRegistrationContext.RegisterModuleType("removed_package", removedPackageModuleFactory)
}

type removedPackageModuleProps struct {
	Message *string
}

type removedPackageModule struct {
	ModuleBase
	properties removedPackageModuleProps
	out        Path
}

func removedPackageModuleFactory() Module {
	m := &removedPackageModule{}
	InitAndroidArchModule(m, DeviceSupported, MultilibCommon)
	m.AddProperties(&m.properties)
	return m
}

var removedPackageRule = pctx.AndroidStaticRule("removed_package", blueprint.RuleParams{
	Command: "echo $message && false",
}, "message")

func (m *removedPackageModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	// Unchecked module so that checkbuild doesn't fail
	ctx.UncheckedModule()

	out := PathForModuleOut(ctx, "out.txt")
	message := fmt.Sprintf("%s has been removed, and can no longer be used.", ctx.ModuleName())
	if m.properties.Message != nil {
		message = *m.properties.Message
	}
	ctx.Build(pctx, BuildParams{
		Rule:   removedPackageRule,
		Output: out,
		Args: map[string]string{
			"message": proptools.ShellEscape(message),
		},
	})

	ctx.InstallFile(PathForModuleInstall(ctx, "removed_module"), ctx.ModuleName(), out)
	m.out = out
}

func (m *removedPackageModule) AndroidMkEntries() []AndroidMkEntries {
	entries := AndroidMkEntries{
		OutputFile: OptionalPathForPath(m.out),
		Class:      "EXECUTABLES",
	}
	return []AndroidMkEntries{entries}
}
