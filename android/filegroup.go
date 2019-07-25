// Copyright 2016 Google Inc. All rights reserved.
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

package android

import (
	"io"
	"strings"
	"text/template"
)

func init() {
	RegisterModuleType("filegroup", FileGroupFactory)
}

type fileGroupProperties struct {
	// srcs lists files that will be included in this filegroup
	Srcs []string `android:"path"`

	Exclude_srcs []string `android:"path"`

	// The base path to the files.  May be used by other modules to determine which portion
	// of the path to use.  For example, when a filegroup is used as data in a cc_test rule,
	// the base path is stripped off the path and the remaining path is used as the
	// installation directory.
	Path *string

	// Create a make variable with the specified name that contains the list of files in the
	// filegroup, relative to the root of the source tree.
	Export_to_make_var *string

	// List of paths that must not overlap with this filegroup.
	Disjoint_from []string `android:"path"`
}

type fileGroup struct {
	ModuleBase
	properties fileGroupProperties
	srcs       Paths
}

var _ SourceFileProducer = (*fileGroup)(nil)

// filegroup contains a list of files that are referenced by other modules
// properties (such as "srcs") using the syntax ":<name>". filegroup are
// also be used to export files across package boundaries.
func FileGroupFactory() Module {
	module := &fileGroup{}
	module.AddProperties(&module.properties)
	InitAndroidModule(module)
	return module
}

func (fg *fileGroup) GenerateAndroidBuildActions(ctx ModuleContext) {
	fg.srcs = PathsForModuleSrcExcludes(ctx, fg.properties.Srcs, fg.properties.Exclude_srcs)

	// If this module is intended to be disjoint from other groups and paths then check for overlaps
	// before rebasing the paths relative to a different location so that error messages are reported
	// using same values as specified in the blueprint file.
	if fg.properties.Disjoint_from != nil {
		for _, disjoint_src := range fg.properties.Disjoint_from {
			disjoint_srcs := PathsForModuleSrc(ctx, []string{disjoint_src})
			overlaps := []string{}
			for _, p1 := range disjoint_srcs {
				for _, p2 := range fg.srcs {
					// Compare paths relative to the root.
					if p1.String() == p2.String() {
						// Report the paths as specified in this module.
						overlaps = append(overlaps, p2.Rel())
						break
					}
				}
			}

			if len(overlaps) > 0 {
				s := ""
				for _, overlap := range overlaps {
					s = s + "    " + overlap + "\n"
				}
				ctx.ModuleErrorf("overlaps with the following paths from %s\n%s", disjoint_src, s)
			}
		}
	}

	if fg.properties.Path != nil {
		fg.srcs = PathsWithModuleSrcSubDir(ctx, fg.srcs, String(fg.properties.Path))
	}
}

func (fg *fileGroup) Srcs() Paths {
	return append(Paths{}, fg.srcs...)
}

var androidMkTemplate = template.Must(template.New("filegroup").Parse(`
ifdef {{.makeVar}}
  $(error variable {{.makeVar}} set by soong module is already set in make)
endif
{{.makeVar}} := {{.value}}
.KATI_READONLY := {{.makeVar}}
`))

func (fg *fileGroup) AndroidMk() AndroidMkData {
	return AndroidMkData{
		Custom: func(w io.Writer, name, prefix, moduleDir string, data AndroidMkData) {
			if makeVar := String(fg.properties.Export_to_make_var); makeVar != "" {
				androidMkTemplate.Execute(w, map[string]string{
					"makeVar": makeVar,
					"value":   strings.Join(fg.srcs.Strings(), " "),
				})
			}
		},
	}
}
