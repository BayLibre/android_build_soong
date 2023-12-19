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

// Create a rule to run a tool to collect all the intermediate files
// which list the team per module into one proto file.
func (this *allTeamsSingleton) GenerateBuildActions(ctx SingletonContext) {
	var intermediateMetadataPaths Paths

	ctx.VisitAllModules(func(module Module) {
		if metadata, ok := SingletonModuleProvider(ctx, module, TeamProviderKey); ok {
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
	ctx.Phony("all_teams_proto", this.outputPath)
}

func (this *allTeamsSingleton) MakeVars(ctx MakeVarsContext) {
	ctx.DistForGoal("all_teams", this.outputPath)
}
