package python

import (
	"android/soong/android"
	"path/filepath"

	"github.com/google/blueprint"
)

// This file contains the module types for building Python test.

func init() {
	android.RegisterModuleType("python_test_host", PythonTestHostFactory)
}

type PythonTestHost struct {
	pythonBinaryBase
}

var _ PythonSubModule = (*PythonTestHost)(nil)

func (p *PythonTestHost) InstallFile(ctx android.ModuleContext,
	input android.Path) android.OutputPath {
	return ctx.InstallFile(android.PathForModuleInstall(
		ctx, filepath.Join("nativetest", ctx.ModuleName())), input)
}

func PythonTestHostFactory() (blueprint.Module, []interface{}) {
	module := &PythonTestHost{}

	module.pythonBinaryBase.subModule = module

	return InitPythonBaseModule(&module.pythonBinaryBase.pythonBaseModule,
		&module.pythonBinaryBase, android.HostSupportedNoCross, &module.binaryProperties)
}
