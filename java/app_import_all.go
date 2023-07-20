package java

import (
	"android/soong/android"
)

func init() {
	android.RegisterSingletonType("all_app_imports_singleton", allAppImportsSingletonFactory)
}

func allAppImportsSingletonFactory() android.Singleton {
	return &allAppImportsSingleton{}
}

type allAppImportsSingleton struct{}

func (a allAppImportsSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	var outputFiles android.Paths
	ctx.VisitAllModules(func(module android.Module) {
		if m, ok := module.(*AndroidAppImport); ok {
			outputFiles = append(outputFiles, m.OutputFile())
		} else if m, ok := module.(*AndroidTestImport); ok {
			outputFiles = append(outputFiles, m.OutputFile())
		}
	})
	ctx.Phony("all_android_app_imports", outputFiles...)
}
