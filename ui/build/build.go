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

package build

import (
	"android/soong/finder"
	"android/soong/fs"
	"android/soong/ui/logger"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Ensures the out directory exists, and has the proper files to prevent kati
// from recursing into it.
func SetupOutDir(ctx Context, config Config) {
	ensureEmptyFileExists(ctx, filepath.Join(config.OutDir(), "Android.mk"))
	ensureEmptyFileExists(ctx, filepath.Join(config.OutDir(), "CleanSpec.mk"))
	if !config.SkipMake() {
		ensureEmptyFileExists(ctx, filepath.Join(config.SoongOutDir(), ".soong.in_make"))
	}
	// The ninja_build file is used by our buildbots to understand that the output
	// can be parsed as ninja output.
	ensureEmptyFileExists(ctx, filepath.Join(config.OutDir(), "ninja_build"))
	ensureEmptyFileExists(ctx, filepath.Join(config.OutDir(), ".out-dir"))
}

var combinedBuildNinjaTemplate = template.Must(template.New("combined").Parse(`
builddir = {{.OutDir}}
{{if .HasKatiSuffix}}include {{.KatiNinjaFile}}
{{end -}}
include {{.SoongNinjaFile}}
build {{.CombinedNinjaFile}}: phony {{.SoongNinjaFile}}
`))

func createCombinedBuildNinjaFile(ctx Context, config Config) {
	// If we're in SkipMake mode, skip creating this file if it already exists
	if config.SkipMake() {
		if _, err := os.Stat(config.CombinedNinjaFile()); err == nil || !os.IsNotExist(err) {
			return
		}
	}

	file, err := os.Create(config.CombinedNinjaFile())
	if err != nil {
		ctx.Fatalln("Failed to create combined ninja file:", err)
	}
	defer file.Close()

	if err := combinedBuildNinjaTemplate.Execute(file, config); err != nil {
		ctx.Fatalln("Failed to write combined ninja file:", err)
	}
}

const (
	BuildNone          = iota
	BuildProductConfig = 1 << iota
	BuildSoong         = 1 << iota
	BuildKati          = 1 << iota
	BuildNinja         = 1 << iota
	BuildAll           = BuildProductConfig | BuildSoong | BuildKati | BuildNinja
)

func checkCaseSensitivity(ctx Context, config Config) {
	outDir := config.OutDir()
	lowerCase := filepath.Join(outDir, "casecheck.txt")
	upperCase := filepath.Join(outDir, "CaseCheck.txt")
	lowerData := "a"
	upperData := "B"

	err := ioutil.WriteFile(lowerCase, []byte(lowerData), 0777)
	if err != nil {
		ctx.Fatalln("Failed to check case sensitivity:", err)
	}

	err = ioutil.WriteFile(upperCase, []byte(upperData), 0777)
	if err != nil {
		ctx.Fatalln("Failed to check case sensitivity:", err)
	}

	res, err := ioutil.ReadFile(lowerCase)
	if err != nil {
		ctx.Fatalln("Failed to check case sensitivity:", err)
	}

	if string(res) != lowerData {
		ctx.Println("************************************************************")
		ctx.Println("You are building on a case-insensitive filesystem.")
		ctx.Println("Please move your source tree to a case-sensitive filesystem.")
		ctx.Println("************************************************************")
		ctx.Fatalln("Case-insensitive filesystems not supported")
	}
}

func help(ctx Context, config Config, what int) {
	cmd := Command(ctx, config, "help.sh", "build/make/help.sh")
	cmd.Sandbox = makeSandbox
	cmd.Stdout = ctx.Stdout()
	cmd.Stderr = ctx.Stderr()
	cmd.RunOrFatal()
}

func NewSourceFinder(ctx Context, config Config) (f *finder.Finder) {
	startTime := uint64(time.Now().UnixNano())
	defer func() {
		ctx.CompleteTrace("find modules", startTime, uint64(time.Now().UnixNano()))
	}()

	dir, err := os.Getwd()
	if err != nil {
		ctx.Fatal(err.Error())
	}
	cacheParams := finder.CacheParams{
		WorkingDirectory: dir,
		RootDirs:         []string{"."},
		ExcludeDirs:      []string{".git", ".repo"},
		PruneFiles:       []string{".out-dir", ".find-ignore"},
		IncludeFiles:     []string{"Android.mk", "Android.bp", "Blueprints", "CleanSpec.mk"},
	}
	dumpDir := config.FileListDir()
	f = finder.New(cacheParams, fs.OsFs, logger.New(ioutil.Discard),
		filepath.Join(dumpDir, "files.db"))
	return f
}

func FindSources(ctx Context, config Config, f *finder.Finder) (err error) {
	// note that dumpDir in FindSources may be different than dumpDir in NewSourceFinder
	// if a caller such as multiproduct_kati wants to share one Finder among several builds
	dumpDir := config.FileListDir()
	os.MkdirAll(dumpDir, 0777)

	androidMks := f.FindFirstNamedAt(".", "Android.mk")
	err = dumpListToFile(androidMks, filepath.Join(dumpDir, "Android.mk.list"))
	if err != nil {
		return err
	}

	cleanSpecs := f.FindFirstNamedAt(".", "CleanSpec.mk")
	dumpListToFile(cleanSpecs, filepath.Join(dumpDir, "CleanSpec.mk.list"))
	if err != nil {
		return err
	}

	isBlueprintFile := func(dir finder.DirEntries) (dirs []string, files []string) {
		files = []string{}
		for _, file := range dir.FileNames {
			if file == "Android.bp" || file == "Blueprints" {
				files = append(files, file)
			}
		}

		return dir.DirNames, files
	}
	androidBps := f.FindMatching(".", isBlueprintFile)
	err = dumpListToFile(androidBps, filepath.Join(dumpDir, "Android.bp.list"))
	if err != nil {
		return err
	}

	return nil
}

func dumpListToFile(list []string, filePath string) (err error) {
	desiredText := strings.Join(list, "\n")
	desiredBytes := []byte(desiredText)
	actualBytes, readErr := ioutil.ReadFile(filePath)
	if readErr != nil || !bytes.Equal(desiredBytes, actualBytes) {
		err = ioutil.WriteFile(filePath, desiredBytes, 0777)
	}
	return err
}

// Build the tree. The 'what' argument can be used to chose which components of
// the build to run.
func Build(ctx Context, config Config, what int) {
	ctx.Verboseln("Starting build with args:", config.Arguments())
	ctx.Verboseln("Environment:", config.Environment().Environ())

	if config.SkipMake() {
		ctx.Verboseln("Skipping Make/Kati as requested")
		what = what & (BuildSoong | BuildNinja)
	}

	if inList("help", config.Arguments()) {
		help(ctx, config, what)
		return
	} else if inList("clean", config.Arguments()) || inList("clobber", config.Arguments()) {
		clean(ctx, config, what)
		return
	}

	// Make sure that no other Soong process is running with the same output directory
	buildLock := BecomeSingletonOrFail(ctx, config)
	defer buildLock.Unlock()

	SetupOutDir(ctx, config)

	checkCaseSensitivity(ctx, config)

	ensureEmptyDirectoriesExist(ctx, config.TempDir())

	if what&BuildProductConfig != 0 {
		// Run make for product config
		runMakeProductConfig(ctx, config)
	}

	if inList("installclean", config.Arguments()) {
		installClean(ctx, config, what)
		ctx.Println("Deleted images and staging directories.")
		return
	} else if inList("dataclean", config.Arguments()) {
		dataClean(ctx, config, what)
		ctx.Println("Deleted data files.")
		return
	}

	if what&BuildSoong != 0 {
		// Run Soong
		runSoongBootstrap(ctx, config)
		runSoong(ctx, config)
	}

	if what&BuildKati != 0 {
		// Run ckati
		runKati(ctx, config)

		ioutil.WriteFile(config.LastKatiSuffixFile(), []byte(config.KatiSuffix()), 0777)
	} else {
		// Load last Kati Suffix if it exists
		if katiSuffix, err := ioutil.ReadFile(config.LastKatiSuffixFile()); err == nil {
			ctx.Verboseln("Loaded previous kati config:", string(katiSuffix))
			config.SetKatiSuffix(string(katiSuffix))
		}
	}

	if what&BuildNinja != 0 {
		if !config.SkipMake() {
			installCleanIfNecessary(ctx, config)
		}

		// Write combined ninja file
		createCombinedBuildNinjaFile(ctx, config)

		// Run ninja
		runNinja(ctx, config)
	}
}
