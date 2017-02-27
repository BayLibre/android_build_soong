package python

// This file contains the module types for compiling Python binary.

import (
	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	android.RegisterModuleType("python_binary_host", PythonBinaryHostFactory)
}

type pythonBinaryProperties struct {
	// the python interpreter chosen to execute the program.
	// valid values are “PY2” or “PY3” (default).
	Py_interpreter_version string
}

type PythonBinary struct {
	pythonBaseModule

	binaryProperties pythonBinaryProperties
}

func PythonBinaryHostFactory() (blueprint.Module, []interface{}) {
	module := &PythonBinary{}

	return InitPythonBaseModule(&module.pythonBaseModule, module,
		&module.binaryProperties)
}

func (p *PythonBinary) GeneratePythonBuildActions(ctx android.ModuleContext) {
	p.pythonBaseModule.GeneratePythonBuildActions(ctx)
	// Above function returns all required srcs hashmap + data hashmap for this bin.
	// TODO(nanzhang): implement binary build actions generation, including zipping, and etc.
}
