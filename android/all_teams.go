package android

const ownershipDirectory = "ownership"
const fileContainingFilePaths = "all_team_paths.rsp"
const allTeamsFile = "all_teams.pb"

func AllTeamsFactory() Singleton {
	return &allTeamsSingleton{}
}

func init() {
	registerAllTeamBuildComponents(InitRegistrationContext)
}

func registerAllTeamBuildComponents(ctx RegistrationContext) {
	ctx.RegisterParallelSingletonType("all_teams", AllTeamsFactory)
}

type allTeamsSingleton struct {
	// Path where the collected metadata is stored after successful validation.
	outputPath OutputPath
}

func (this *allTeamsSingleton) GenerateBuildActions(ctx SingletonContext) {
	var intermediateMetadataPaths Paths

	ctx.VisitAllModules(func(module Module) {
		if metadata, ok := SingletonModuleProvider(ctx, module, TeamProviderKey); ok {
			// TODO(ron): Can I just read the module data here rather
			// than read the protos?  Ah, NO. This does not write to out,
			// it writes a rule to read the input protos and collect them.
			// I mean embedding the small trendy_id and name is similar to the path in the .ninja file.
			intermediateMetadataPaths = append(intermediateMetadataPaths, metadata.IntermediatePath)
		}
	})

	rspFile := PathForOutput(ctx, fileContainingFilePaths)
	this.outputPath = PathForOutput(ctx, ownershipDirectory, allTeamsFile)

	rule := NewRuleBuilder(pctx, ctx)
	cmd := rule.Command().
		BuiltTool("team-collector").
		FlagWithRspFileInputList("-inputFile ", rspFile, intermediateMetadataPaths)
	cmd.FlagWithOutput("-outputFile ", this.outputPath)
	rule.Build("all_teams_rule", "Generate all test specifications")
	ctx.Phony("all_teams_", this.outputPath)
}

func (this *allTeamsSingleton) MakeVars(ctx MakeVarsContext) {
	ctx.DistForGoal("all_teams", this.outputPath)
}
