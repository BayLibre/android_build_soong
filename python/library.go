package python

// This file contains the module types for compiling Python library.

import (
	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	android.RegisterModuleType("python_library_host", PythonLibraryHostFactory)
}

type PythonLibrary struct {
	pythonBaseModule
}

func PythonLibraryHostFactory() (blueprint.Module, []interface{}) {
	module := &PythonLibrary{}

	return InitPythonBaseModule(&module.pythonBaseModule, module)
}

func (p *PythonLibrary) GeneratePythonBuildActions(ctx android.ModuleContext) (srcs, data map[string]string) {
	srcs, data = p.pythonBaseModule.GeneratePythonBuildActions(ctx)
	return
}
