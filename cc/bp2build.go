package cc

import (
	"android/soong/android"
	"android/soong/bazel"

	"github.com/google/blueprint/proptools"
)

func init() {
	android.RegisterBp2BuildMutator("cc_defaults", bp2buildMutator)
}

type bazelCcDefaultsAttributes struct {
	Name *string

	Host_supported bool
}

type bazelCcDefaults struct {
	android.BazelTargetModuleBase
	bazelCcDefaultsAttributes
}

func BazelCcDefaultsFactory() android.Module {
	module := &bazelCcDefaults{}
	module.AddProperties(&module.bazelCcDefaultsAttributes)
	android.InitBazelTargetModule(module)
	return module
}

func bp2buildMutator(ctx android.TopDownMutatorContext) {
	if m, ok := ctx.Module().(*Defaults); ok {
		name := "__bp2build__" + m.Name()

		ctx.CreateModule(BazelCcDefaultsFactory, &bazelCcDefaultsAttributes{
			Name:           proptools.StringPtr(name),
			Host_supported: true,
		}, &bazel.BazelTargetModuleProperties{
			Rule_class: "cc_defaults",
			Load:       "//build/bazel/rules/cc:cc_defaults.bzl",
		})
	}
}

func (m *bazelCcDefaults) Name() string {
	return m.BaseModuleName()
}

func (m *bazelCcDefaults) GenerateAndroidBuildActions(ctx android.ModuleContext) {}
