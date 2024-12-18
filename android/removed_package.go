package android

import (
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
}

func removedPackageModuleFactory() Module {
	m := &removedPackageModule{}
	InitAndroidModule(m)
	m.AddProperties(&m.properties)
	return m
}

var removedPackageRule = pctx.AndroidStaticRule("removed_package", blueprint.RuleParams{
	Command: "echo $message && false",
}, "message")

func (m *removedPackageModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	ctx.UncheckedModule()
	out := PathForModuleOut(ctx, "out.txt")
	ctx.Build(pctx, BuildParams{
		Rule:   removedPackageRule,
		Output: out,
		Args: map[string]string{
			"message": proptools.ShellEscape(proptools.StringDefault(m.properties.Message, "This module has been removed, and can no longer be used.")),
		},
	})

	ctx.InstallFile(PathForModuleInstall(ctx, "removed_module"), m.Name(), out)
}
