package genrule

import "android/soong/android"

func init() {
	android.InitRegistrationContext.RegisterSingletonType("all_genrules", createAllGenrulesSingleton)
}

func createAllGenrulesSingleton() android.Singleton {
	return &allGenrulesSingleton{}
}

type allGenrulesSingleton struct {
}

func (m *allGenrulesSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	ctx.VisitAllModules(func(m android.Module) {
		if _, ok := m.(*Module); ok && m.Enabled(ctx) {
			outputFiles := android.OutputFilesForModule(ctx, m, "")
			ctx.Phony("all_genrules", outputFiles...)
		}
	})
}
