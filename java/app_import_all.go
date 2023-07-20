package java

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
		if m, ok := module.(*AndroidAppImport); ok && m.Enabled() {
			outputFiles = append(outputFiles, m.outputFile)
		}
	})
	ctx.Phony("all_android_app_imports", outputFiles...)
}
