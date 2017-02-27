// Copyright 2017 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package python

// This file contains the module types for building Python binary.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	android.RegisterModuleType("python_binary_host", PythonBinaryHostFactory)
}

type PythonBinaryProperties struct {
	// the name of the source file that is the main entry point of the program.
	// this file must also be listed in srcs.
	// If left unspecified, module name is used instead.
	// If name doesn’t match any filename in srcs, main must be specified.
	Main string
}

type PythonBinary struct {
	pythonBaseModule

	binaryProperties PythonBinaryProperties
}

func PythonBinaryHostFactory() (blueprint.Module, []interface{}) {
	module := &PythonBinary{}

	return InitPythonBaseModule(&module.pythonBaseModule, module, &module.binaryProperties)
}

func (p *PythonBinary) GeneratePythonBuildActions(ctx android.ModuleContext) {
	srcsRunfiles := p.pythonBaseModule.GeneratePythonBuildActions(ctx)

	if len(srcsRunfiles) == 0 {
		return
	}

	newInitFileDirs := []string{}
	newInitFileDirSet := make(map[string]bool)
	existedInitFileDirSet := make(map[string]bool)

	for _, ele := range srcsRunfiles {
		if filepath.Base(ele) != "__init__.py" {
			continue
		}
		existedInitFileDir := filepath.Dir(ele)
		if _, found := existedInitFileDirSet[existedInitFileDir]; found {
			continue
		} else {
			existedInitFileDirSet[existedInitFileDir] = true
		}
		lastDir := TrimPathFromLastSlash(filepath.Dir(ele))
		PopulateNewInitFileDirs(lastDir, existedInitFileDirSet,
			newInitFileDirSet, &newInitFileDirs)
	}

	for _, ele := range srcsRunfiles {
		if filepath.Base(ele) == "__init__.py" {
			continue
		}
		lastDir := TrimPathFromLastSlash(ele)
		PopulateNewInitFileDirs(lastDir, existedInitFileDirSet,
			newInitFileDirSet, &newInitFileDirs)
	}

	var interpreter string
	switch p.pythonBaseModule.properties.ActualVersion {
	case pyVersion2:
		interpreter = p.properties.Version.Py2.Interpreter
	case pyVersion3:
		interpreter = p.properties.Version.Py3.Interpreter
	default:
		panic(fmt.Errorf("unknown Python actualVersion: %q for module: %q.",
			p.properties.ActualVersion, ctx.ModuleName()))
	}
	interpreter = filepath.Clean(interpreter)
	if !strings.HasPrefix(interpreter, "/") {
		ctx.PropertyErrorf("interpreter", "%q is not a absolute path.", interpreter)
		return
	}

	var main string
	if p.binaryProperties.Main == "" {
		main = p.BaseModuleName() + pyExt
	} else {
		main = p.binaryProperties.Main
	}
	found := false
	for _, ele := range p.pythonBaseModule.srcsPathMappings {
		if main == ele.src.Rel() {
			found = true
			RegisterBuildActionForParFile(ctx, interpreter, ele.dest, newInitFileDirs,
				p.pythonBaseModule.parSpecs)
			break
		}
	}
	if found == false {
		ctx.PropertyErrorf("main", "%q is not listed in srcs.", main)
		return
	}

}

func PopulateNewInitFileDirs(lastDir string,
	existedInitFileDirSet, newInitFileDirSet map[string]bool, newInitFileDirs *[]string) {
	for lastDir != "" {
		if _, found := existedInitFileDirSet[lastDir]; found {
			break
		}
		if _, found := newInitFileDirSet[lastDir]; !found {
			newInitFileDirSet[lastDir] = true
			*newInitFileDirs = append(*newInitFileDirs, lastDir)
			lastDir = TrimPathFromLastSlash(lastDir)
		} else {
			break
		}
	}
}

func TrimPathFromLastSlash(path string) string {
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		return path[:idx]
	}
	return ""
}
