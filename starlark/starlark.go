// Copyright 2023 Google Inc. All rights reserved.
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

package starlark

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

const callerDirKey = "callerDir"

type modentry struct {
	globals starlark.StringDict
	err     error
}

var builtins = starlark.StringDict{
	"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
}

// Takes a module name (the first argument to the load() function) and returns the path
// it's trying to load, stripping out leading //, and handling leading :s.
func cleanModuleName(moduleName string, callerDir string) (string, error) {
	if strings.Count(moduleName, ":") > 1 {
		return "", fmt.Errorf("at most 1 colon must pre present in starlark path: %s", moduleName)
	}

	localLoad := false
	if strings.HasPrefix(moduleName, "//") {
		moduleName = moduleName[2:]
	} else if strings.HasPrefix(moduleName, ":") {
		moduleName = moduleName[1:]
		localLoad = true
	} else {
		return "", fmt.Errorf("load path must start with // or :")
	}

	if ix := strings.LastIndex(moduleName, ":"); ix >= 0 {
		moduleName = moduleName[:ix] + string(os.PathSeparator) + moduleName[ix+1:]
	}

	if filepath.Clean(moduleName) != moduleName {
		return "", fmt.Errorf("load path must be clean, found: %s, expected: %s", moduleName, filepath.Clean(moduleName))
	}
	if strings.HasPrefix(moduleName, "../") {
		return "", fmt.Errorf("load path must not start with ../: %s", moduleName)
	}
	if strings.HasPrefix(moduleName, "/") {
		return "", fmt.Errorf("load path starts with /, use // for a absolute path: %s", moduleName)
	}

	if localLoad {
		return filepath.Join(callerDir, moduleName), nil
	}

	return moduleName, nil
}

// loader implements load statement. The format of the loaded module URI is
//
//	[//path]:base
//
// The file path is $ROOT/path/base if path is present, <caller_dir>/base otherwise.
func loader(thread *starlark.Thread, module string, moduleCache map[string]*modentry, filesystem map[string]string) (starlark.StringDict, error) {
	modulePath, err := cleanModuleName(module, thread.Local(callerDirKey).(string))
	if err != nil {
		return nil, err
	}
	e, ok := moduleCache[modulePath]
	if e == nil {
		if ok {
			return nil, fmt.Errorf("cycle in load graph")
		}

		// Add a placeholder to indicate "load in progress".
		moduleCache[modulePath] = nil

		childThread := &starlark.Thread{Name: "exec " + module, Load: thread.Load}

		// Cheating for the sake of testing:
		// propagate starlarktest's Reporter key, otherwise testing
		// the load function may cause panic in starlarktest code.
		const testReporterKey = "Reporter"
		if v := thread.Local(testReporterKey); v != nil {
			childThread.SetLocal(testReporterKey, v)
		}

		childThread.SetLocal(callerDirKey, filepath.Dir(modulePath))

		if filesystem != nil {
			globals, err := starlark.ExecFile(childThread, modulePath, filesystem[modulePath], builtins)
			e = &modentry{globals, err}
		} else {
			globals, err := starlark.ExecFile(childThread, modulePath, nil, builtins)
			e = &modentry{globals, err}
		}

		// Update the cache.
		moduleCache[modulePath] = e
	}
	return e.globals, e.err
}

// Run runs the given starlark file and returns its global variables. It will assume the current
// directory is the root of the repository, for the purposes of // loads.
func Run(filename string) (starlark.StringDict, error) {
	return runWithFilesystem(filename, nil)
}

func runWithFilesystem(filename string, filesystem map[string]string) (starlark.StringDict, error) {
	if !strings.HasPrefix(filename, "//") && !strings.HasPrefix(filename, ":") {
		filename = "//" + filename
	}
	filename, err := cleanModuleName(filename, "")
	if err != nil {
		return nil, err
	}
	moduleCache := make(map[string]*modentry)
	mainThread := &starlark.Thread{
		Name: "main",
		Print: func(_ *starlark.Thread, msg string) {
			// Ignore prints
		},
		Load: func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
			return loader(thread, module, moduleCache, filesystem)
		},
	}
	mainThread.SetLocal(callerDirKey, filepath.Dir(filename))

	if filesystem != nil {
		return starlark.ExecFile(mainThread, filename, filesystem[filename], builtins)
	} else {
		return starlark.ExecFile(mainThread, filename, nil, builtins)
	}
}
