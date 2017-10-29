// Copyright 2018 Google Inc. All rights reserved.
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

package build

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/blueprint/microfactory"

	"android/soong/ui/build/paths"
)

func parsePathDir(dir string) []string {
	f, err := os.Open(dir)
	if err != nil {
		return nil
	}
	defer f.Close()

	if s, err := f.Stat(); err != nil || !s.IsDir() {
		return nil
	}

	infos, err := f.Readdir(-1)
	if err != nil {
		return nil
	}

	ret := make([]string, 0, len(infos))
	for _, info := range infos {
		if m := info.Mode(); !m.IsDir() && m&0111 != 0 {
			ret = append(ret, info.Name())
		}
	}
	return ret
}

func SetupPath(ctx Context, config Config) {
	if config.pathReplaced {
		return
	}

	ctx.BeginTrace("path")
	defer ctx.EndTrace()

	origPath, _ := config.Environment().Get("PATH")
	myPath := filepath.Join(config.OutDir(), ".path")
	interposer := myPath + "_interposer"

	var cfg microfactory.Config
	cfg.Map("android/soong", "build/soong")
	cfg.TrimPath, _ = filepath.Abs(".")
	if _, err := microfactory.Build(&cfg, interposer, "android/soong/cmd/path_interposer"); err != nil {
		ctx.Fatalln("Failed to build path interposer:", err)
	}

	if err := ioutil.WriteFile(interposer+"_origpath", []byte(origPath), 0777); err != nil {
		ctx.Fatalln("Failed to write original path:", err)
	}

	entries, err := paths.LogListener(ctx.Context, interposer+"_log")
	if err != nil {
		ctx.Fatalln("Failed to listen for path logs:", err)
	}

	go func() {
		for log := range entries {
			if allowed, ok := paths.AllowedPathEntries[log.Basename]; ok && !allowed {
				ctx.Fatalf("Disallowed PATH tool %q used: %#v\n", log.Basename, log.Args)
			} else {
				ctx.Verbosef("Unknown PATH tool %q used: %#v\n", log.Basename, log.Args)
			}
		}
	}()

	ensureEmptyDirectoriesExist(ctx, myPath)

	var execs []string
	for _, pathEntry := range filepath.SplitList(origPath) {
		if pathEntry == "" {
			// Ignore the current directory
			continue
		}
		// TODO(dwillemsen): remove path entries under TOP? or anything
		// that looks like an android source dir? They won't exist on
		// the build servers, since they're added by envsetup.sh.

		execs = append(execs, parsePathDir(pathEntry)...)
	}

	for _, name := range execs {
		if allowed, ok := paths.AllowedPathEntries[name]; ok && !allowed {
			continue
		}

		err := os.Symlink("../.path_interposer", filepath.Join(myPath, name))
		// Intentionally ignore existing files -- that means that we
		// just created it, and the first one should win.
		if err != nil && !os.IsExist(err) {
			ctx.Fatalln("Failed to create symlink:", err)
		}
	}

	myPath, _ = filepath.Abs(myPath)
	config.Environment().Set("PATH", myPath)
	config.pathReplaced = true
}
