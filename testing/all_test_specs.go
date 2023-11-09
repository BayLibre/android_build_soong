package testing

import (
	"strings"

	"android/soong/android"
)

func AllTestSpecsFactory() android.Singleton {
	return &allTestSpecsSingleton{}
}

type allTestSpecsSingleton struct {
	// Paths where individual test-spec proto metadata files are stored after marshaling.
	intermediateFileContainingFilePaths android.WritablePath
	// Path where the collected metadata is stored after successful validation.
	intermediateOutputPath android.OutputPath
}

func (this *allTestSpecsSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	var intermediateMetadataPaths android.Paths

	ctx.VisitAllModules(func(module android.Module) {
		if !ctx.ModuleHasProvider(module, testSpecProviderKey) {
			return
		}
		intermediateMetadataPaths = append(intermediateMetadataPaths, ctx.ModuleProvider(module, testSpecProviderKey).(testSpecProviderData).IntermediatePath)
	})

	intermediateMetadataPathStrings := make([]string, len(intermediateMetadataPaths))
	for i, item := range intermediateMetadataPaths {
		intermediateMetadataPathStrings[i] = item.String()
	}
	intermediateFilePaths := strings.Join(intermediateMetadataPathStrings, "\n") + "\n"
	this.intermediateFileContainingFilePaths = android.PathForOutput(ctx, "ownership", "all_test_spec_paths.txt")
	android.WriteFileRuleWithInput(ctx, intermediateMetadataPaths, this.intermediateFileContainingFilePaths, intermediateFilePaths)

	this.intermediateOutputPath = android.PathForOutput(ctx, "ownership", "all_test_secs.pb")
	ctx.Build(pctx, android.BuildParams{
		Rule:        allTestSpecsRule,
		Input:       this.intermediateFileContainingFilePaths,
		Output:      this.intermediateOutputPath,
		Description: "all_test_specs",
		Args: map[string]string{
			"input_file": this.intermediateFileContainingFilePaths.String(),
		},
	})

	ctx.Phony("all_test_specs", this.intermediateOutputPath)
}

func (this *allTestSpecsSingleton) MakeVars(ctx android.MakeVarsContext) {
	ctx.DistForGoal("test_specs", this.intermediateOutputPath)
}
