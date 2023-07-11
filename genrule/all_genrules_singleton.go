package genrule

import "android/soong/android"

func init() {
	android.RegisterSingletonType("all_genrules_singleton", allGenrulesSingletonFactory)
}

func allGenrulesSingletonFactory() android.Singleton {
	return &allGenrulesSingleton{}
}

type allGenrulesSingleton struct{}

func (a allGenrulesSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	var outputFiles android.Paths
	ctx.VisitAllModules(func(module android.Module) {
		if m, ok := module.(*Module); ok && m.Enabled() {
			files, err := m.OutputFiles("")
			if err != nil {
				ctx.Errorf("%s", err.Error())
			}
			outputFiles = append(outputFiles, files...)
		}
	})
	ctx.Phony("all_genrules", outputFiles...)
}
