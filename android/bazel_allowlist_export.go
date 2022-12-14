package android

import (
	"sort"
	"strings"
)

func init() {
	RegisterSingletonType("bazel_allowlist_export",
		func() Singleton { return &allowlistExport{} })
}

type allowlistExport struct {
}

func (a allowlistExport) GenerateBuildActions(ctx SingletonContext) {
	path := PathForOutput(ctx, "bazel_prod_mixed_builds_enabled_list.txt")
	writeBazelEnabledList(ctx, BazelProdMode, path)
	path = PathForOutput(ctx, "bazel_staging_mixed_builds_enabled_list.txt")
	writeBazelEnabledList(ctx, BazelStagingMode, path)
}

func writeBazelEnabledList(ctx SingletonContext, mode SoongBuildMode, output WritablePath) {
	enabledModules, disabledModules := GetBazelEnabledModules(mode, nil)

	enabledList := make([]string, 0)
	for module := range enabledModules {
		if !disabledModules[module] {
			enabledList = append(enabledList, module)
		}
	}
	sort.Strings(enabledList)

	rule := NewRuleBuilder(pctx, ctx)
	rule.Command().Text("(echo " + strings.Join(enabledList, " && echo ") + ") > ").Output(output)
	rule.Build(output.Base(), output.Base())
}
