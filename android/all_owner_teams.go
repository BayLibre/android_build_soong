package android

const ownershipDirectory = "ownership"
const fileContainingFilePaths = "all_owner_team_paths.rsp"
const allOwnerTeamsFile = "all_owner_teams.pb"

func AllOwnerTeamsFactory() Singleton {
	return &allOwnerTeamsSingleton{}
}

func init() {
	registerAllTeamBuildComponents(InitRegistrationContext)
}

func registerAllTeamBuildComponents(ctx RegistrationContext) {
	ctx.RegisterParallelSingletonType("all_owner_teams", AllOwnerTeamsFactory)
}

type allOwnerTeamsSingleton struct {
	// Path where the collected metadata is stored after successful validation.
	outputPath OutputPath
}

func (this *allOwnerTeamsSingleton) GenerateBuildActions(ctx SingletonContext) {
	var intermediateMetadataPaths Paths

	ctx.VisitAllModules(func(module Module) {
		if !ctx.ModuleHasProvider(module, OwnerTeamProviderKey) {
			return
		}
		// TODO(ron): Can I just read the module data here rather
		// than read the protos?  Ah, NO. This does not write to out,
		// it writes a rule to read the input protos and collect them.
		// I mean embedding the small trendy_id and name is similar to the path in the .ninja file.
		intermediateMetadataPaths = append(intermediateMetadataPaths, ctx.ModuleProvider(module, OwnerTeamProviderKey).(OwnerTeamProviderData).IntermediatePath)
	})

	rspFile := PathForOutput(ctx, fileContainingFilePaths)
	this.outputPath = PathForOutput(ctx, ownershipDirectory, allOwnerTeamsFile)

	rule := NewRuleBuilder(pctx, ctx)
	cmd := rule.Command().
		BuiltTool("owner_collector").
		FlagWithRspFileInputList("-inputFile ", rspFile, intermediateMetadataPaths)
	cmd.FlagWithOutput("-outputFile ", this.outputPath)
	rule.Build("all_owner_teams_rule", "Generate all test specifications")
	ctx.Phony("all_owner_teams", this.outputPath)
}

func (this *allOwnerTeamsSingleton) MakeVars(ctx MakeVarsContext) {
	ctx.DistForGoal("all_owners", this.outputPath)
}
