package testing

import (
	"android/soong/android"
)

func init() {
	RegisterBuildComponents(android.InitRegistrationContext)
}

func RegisterBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("test_spec", TestSpecFactory)
}
