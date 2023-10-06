package testrun

import (
	"android/soong/android"
)

func init() {
	RegisterBuildComponents(android.InitRegistrationContext)
}

func RegisterBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("test_run", TestRunFactory)
	ctx.RegisterParallelSingletonType("all_test_run", AllTestRunFactory)
}
