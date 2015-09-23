// Copyright 2015 Google Inc. All rights reserved.
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

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/blueprint/pathtools"
)

type Path interface {
	// Returns the path in string form
	String() string

	// Returns the current file extension of the path
	Ext() string
}

// WritablePath is a type of path that can be used as an output for build rules.
type WritablePath interface {
	Path

	writablePath()
}

type genPathProvider interface {
	genPathWithExt(ctx AndroidModuleContext, ext string) ModuleGenPath
}
type objPathProvider interface {
	objPathWithExt(ctx AndroidModuleContext, subdir, ext string) ModuleObjPath
}
type resPathProvider interface {
	resPathWithName(ctx AndroidModuleContext, name string) ModuleResPath
}

// GenPathWithExt derives a new file path in ctx's generated sources directory
// from the current path, but with the new extension.
func GenPathWithExt(ctx AndroidModuleContext, p Path, ext string) ModuleGenPath {
	if path, ok := p.(genPathProvider); ok {
		return path.genPathWithExt(ctx, ext)
	}
	ctx.ModuleErrorf("Tried to create generated file from unsupported path: %s(%s)", reflect.TypeOf(p).Name(), p)
	return PathForModuleGen(ctx)
}

// ObjPathWithExt derives a new file path in ctx's object directory from the
// current path, but with the new extension.
func ObjPathWithExt(ctx AndroidModuleContext, p Path, subdir, ext string) ModuleObjPath {
	if path, ok := p.(objPathProvider); ok {
		return path.objPathWithExt(ctx, subdir, ext)
	}
	ctx.ModuleErrorf("Tried to create object file from unsupported path: %s (%s)", reflect.TypeOf(p).Name(), p)
	return PathForModuleObj(ctx)
}

// ResPathWithName derives a new path in ctx's output resource directory, using
// the current path to create the directory name, and the `name` argument for
// the filename.
func ResPathWithName(ctx AndroidModuleContext, p Path, name string) ModuleResPath {
	if path, ok := p.(resPathProvider); ok {
		return path.resPathWithName(ctx, name)
	}
	ctx.ModuleErrorf("Tried to create object file from unsupported path: %s (%s)", reflect.TypeOf(p).Name(), p)
	return PathForModuleRes(ctx)
}

// OptionalPath is a container that may or may not contain a valid Path.
type OptionalPath struct {
	valid bool
	path  Path
}

// OptionalPathForPath returns an OptionalPath containing the path.
func OptionalPathForPath(path Path) OptionalPath {
	return OptionalPath{valid: true, path: path}
}

// Valid returns whether there is a valid path
func (p OptionalPath) Valid() bool {
	return p.valid
}

// Path returns the Path embedded in this OptionalPath. You must be sure that
// there is a valid path, since this method will panic if there is not.
func (p OptionalPath) Path() Path {
	if !p.valid {
		panic("Requesting an invalid path")
	}
	return p.path
}

// String returns the string version of the Path, or "" if it isn't valid.
func (p OptionalPath) String() string {
	if p.valid {
		return p.path.String()
	} else {
		return ""
	}
}

// Paths is a slice of Path objects, with helpers to operate on the collection.
type Paths []Path

// PathsForSource returns Paths rooted from SrcDir
func PathsForSource(ctx AndroidModuleContext, paths []string) Paths {
	ret := make(Paths, 0, len(paths))
	for _, path := range paths {
		ret = append(ret, PathForSource(ctx, path))
	}
	return ret
}

// PathsForSourceConfig is the same as PathsForSource, but doesn't require a
// ModuleContext
func PathsForSourceConfig(config Config, paths []string) (Paths, error) {
	ret := make(Paths, 0, len(paths))
	for _, path := range paths {
		p, err := PathForSourceConfig(config, path)
		if err != nil {
			return nil, err
		}
		ret = append(ret, p)
	}
	return ret, nil
}

// PathsForModuleSrc returns Paths rooted from the module's local source
// directory
func PathsForModuleSrc(ctx AndroidModuleContext, paths []string) Paths {
	ret := make(Paths, 0, len(paths))
	for _, path := range paths {
		ret = append(ret, PathForModuleSrc(ctx, path))
	}
	return ret
}

// pathsForModuleSrcFromFullPath returns Paths rooted from the module's local
// source directory, but strip the local source directory from the beginning of
// each string.
func pathsForModuleSrcFromFullPath(ctx AndroidModuleContext, paths []string) Paths {
	prefix := filepath.Join(ctx.AConfig().srcDir, ctx.ModuleDir()) + "/"
	ret := make(Paths, 0, len(paths))
	for _, p := range paths {
		path := filepath.Clean(p)
		if !strings.HasPrefix(path, prefix) {
			ctx.ModuleErrorf("Path '%s' is not in module source directory '%s'", p, prefix)
			continue
		}
		ret = append(ret, PathForModuleSrc(ctx, path[len(prefix):]))
	}
	return ret
}

// PathsWithOptionalDefaultForModuleSrc returns Paths rooted from the module's
// local source directory. If none are provided, use the default if it exists.
func PathsWithOptionalDefaultForModuleSrc(ctx AndroidModuleContext, input []string, def string) Paths {
	if len(input) > 0 {
		return PathsForModuleSrc(ctx, input)
	}
	// Use Glob so that if the default doesn't exist, a dependency is added so that when it
	// is created, we're run again.
	path := filepath.Join(ctx.AConfig().srcDir, ctx.ModuleDir(), def)
	return ctx.Glob("default", path, []string{})
}

// Strings returns the Paths in string form
func (p Paths) Strings() []string {
	ret := make([]string, 0, len(p))
	for _, path := range p {
		ret = append(ret, path.String())
	}
	return ret
}

// WritablePaths is a slice of WritablePaths, used for multiple outputs.
type WritablePaths []WritablePath

// Strings returns the string forms of the writable paths.
func (p WritablePaths) Strings() []string {
	ret := make([]string, 0, len(p))
	for _, path := range p {
		ret = append(ret, path.String())
	}
	return ret
}

type basePath struct {
	path   string
	config Config
}

func (p basePath) Ext() string {
	return filepath.Ext(p.path)
}

// SourcePath is a Path representing a file path rooted from SrcDir
type SourcePath struct {
	basePath
}
var _ Path = SourcePath{}

// safePathForSourceConfig is for paths that we expect are safe -- only for use
// by go code that is embedding ninja variables in paths
func safePathForSourceConfig(config Config, path string) (SourcePath, error) {
	p, err := validateSafePath(path)
	ret := SourcePath{basePath{p, config}}
	if err != nil {
		return ret, err
	}

	abs, err := filepath.Abs(ret.String())
	if err != nil {
		return ret, err
	}
	buildroot, err := filepath.Abs(config.buildDir)
	if err != nil {
		return ret, err
	}
	if strings.HasPrefix(abs, buildroot) {
		return ret, fmt.Errorf("source path %s is in output", abs)
	}

	return ret, err
}

// PathForSource returns a SourcePath for the provided paths... (which are
// joined together with filepath.Join). This also validates that the path
// doesn't escape the source dir, or is contained in the build dir. On error, it
// will return a usable, but invalid SourcePath, and report a ModuleError.
func PathForSource(ctx AndroidModuleContext, paths ...string) SourcePath {
	p, err := PathForSourceConfig(ctx.AConfig(), paths...)
	if err != nil {
		ctx.ModuleErrorf("%s", err.Error())
	}
	return p
}

// PathForSourceConfig is the same as PathForSource, but does not require an
// AndroidModuleContext
func PathForSourceConfig(config Config, paths ...string) (SourcePath, error) {
	p, err := validatePath(paths...)
	ret := SourcePath{basePath{p, config}}
	if err != nil {
		return ret, err
	}

	abs, err := filepath.Abs(ret.String())
	if err != nil {
		return ret, err
	}
	buildroot, err := filepath.Abs(config.buildDir)
	if err != nil {
		return ret, err
	}
	if strings.HasPrefix(abs, buildroot) {
		return ret, fmt.Errorf("source path %s is in output", abs)
	}

	if _, err = os.Stat(ret.String()); err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("source path %s does not exist", ret)
		} else {
			err = fmt.Errorf("%s: %s", ret, err.Error())
		}
	}
	return ret, err
}

func (p SourcePath) String() string {
	return filepath.Join(p.config.srcDir, p.path)
}

// Join creates a new SourcePath with paths... joined with the current path. The
// provided paths... may not use '..' to escape from the current path.
func (p SourcePath) Join(paths ...string) (SourcePath, error) {
	path, err := validatePath(paths...)
	if err != nil {
		return p, err
	}
	return PathForSourceConfig(p.config, p.path, path)
}

// JoinCtx is the same as Join, but takes an AndroidModuleContext for error reporting.
func (p SourcePath) JoinCtx(ctx AndroidModuleContext, paths ...string) SourcePath {
	path := validatePathCtx(ctx, paths...)
	return PathForSource(ctx, p.path, path)
}

// OverlayPath returns the overlay for `path' if it exists. This assumes that the
// SourcePath is the path to a resource overlay directory.
func (p SourcePath) OverlayPath(ctx AndroidModuleContext, path Path) OptionalPath {
	var relDir string
	if moduleSrcPath, ok := path.(ModuleSrcPath); ok {
		relDir = moduleSrcPath.sourcePath.path
	} else if srcPath, ok := path.(SourcePath); ok {
		relDir = srcPath.path
	} else {
		ctx.ModuleErrorf("Cannot find relative path for %s(%s)", reflect.TypeOf(path).Name(), path)
		return OptionalPath{}
	}
	dir := filepath.Join(p.config.srcDir, p.path, relDir)
	// Use Glob so that we are run again if the directory is added.
	paths, err := Glob(ctx, PathForModuleOut(ctx, "overlay").String(), dir, []string{})
	if err != nil {
		ctx.ModuleErrorf("glob: %s", err.Error())
		return OptionalPath{}
	}
	if len(paths) == 0 {
		return OptionalPath{}
	}
	relPath, err := filepath.Rel(p.config.srcDir, paths[0])
	if err != nil {
		ctx.ModuleErrorf("%s", err.Error())
		return OptionalPath{}
	}
	return OptionalPathForPath(PathForSource(ctx, relPath))
}

// OutputPath is a Path representing a file path rooted from the build directory
type OutputPath struct {
	basePath
}
var _ Path = OutputPath{}

// PathForOutput returns an OutputPath for the provided paths... (which are
// joined together with filepath.Join). This also validates that the path
// does not escape the build dir. On error, it will return a usable, but invalid
// OutputPath, and report a ModuleError.
func PathForOutput(ctx AndroidModuleContext, paths ...string) OutputPath {
	path := validatePathCtx(ctx, paths...)
	return OutputPath{basePath{path,ctx.AConfig()}}
}

// PathForOutputConfig is the same as PathForOutput, but does not require an
// AndroidModuleContext
func PathForOutputConfig(config Config, paths ...string) (OutputPath, error) {
	path, err := validatePath(paths...)
	return OutputPath{basePath{path,config}}, err
}

func (p OutputPath) writablePath() {}

func (p OutputPath) String() string {
	return filepath.Join(p.config.buildDir, p.path)
}

// Join creates a new OutputPath with paths... joined with the current path. The
// provided paths... may not use '..' to escape from the current path.
func (p OutputPath) Join(paths ...string) (OutputPath, error) {
	path, err := validatePath(paths...)
	if err != nil {
		return p, err
	}
	return PathForOutputConfig(p.config, p.path, path)
}

// PathForIntermediatesConfig returns an OutputPath representing the top-level
// intermediates directory.
func PathForIntermediatesConfig(config Config, paths ...string) (OutputPath, error) {
	path, err := validatePath(paths...)
	if err != nil {
		return OutputPath{basePath{"",config}}, err
	}
	return PathForOutputConfig(config, ".intermediates", path)
}

// ModuleSrcPath is a Path representing a file rooted from a module's local source dir
type ModuleSrcPath struct {
	basePath
	sourcePath SourcePath
	moduleDir  string
}
var _ Path = ModuleSrcPath{}
var _ genPathProvider = ModuleSrcPath{}
var _ objPathProvider = ModuleSrcPath{}
var _ resPathProvider = ModuleSrcPath{}

// PathForModuleSrc returns a ModuleSrcPath representing the paths... under the
// module's local source directory.
func PathForModuleSrc(ctx AndroidModuleContext, paths ...string) ModuleSrcPath {
	path := validatePathCtx(ctx, paths...)
	return ModuleSrcPath{basePath{path,ctx.AConfig()}, PathForSource(ctx, ctx.ModuleDir(), path), ctx.ModuleDir()}
}

// OptionalPathForModuleSrc returns an OptionalPath. The OptionalPath contains a
// valid path if p is non-nil.
func OptionalPathForModuleSrc(ctx AndroidModuleContext, p *string) OptionalPath {
	if p == nil {
		return OptionalPath{}
	}
	return OptionalPathForPath(PathForModuleSrc(ctx, *p))
}

func (p ModuleSrcPath) String() string {
	return p.sourcePath.String()
}

func (p ModuleSrcPath) genPathWithExt(ctx AndroidModuleContext, ext string) ModuleGenPath {
	return PathForModuleGen(ctx, p.moduleDir, pathtools.ReplaceExtension(p.path, ext))
}

func (p ModuleSrcPath) objPathWithExt(ctx AndroidModuleContext, subdir, ext string) ModuleObjPath {
	return PathForModuleObj(ctx, subdir, p.moduleDir, pathtools.ReplaceExtension(p.path, ext))
}

func (p ModuleSrcPath) resPathWithName(ctx AndroidModuleContext, name string) ModuleResPath {
	// TODO: Use full directory if the new ctx is not the current ctx?
	return PathForModuleRes(ctx, p.path, name)
}

// ModuleOutPath is a Path representing a module's output directory.
type ModuleOutPath struct {
	OutputPath
}
var _ Path = ModuleOutPath{}

// PathForModuleOut returns a Path representing the paths... under the module's
// output directory.
func PathForModuleOut(ctx AndroidModuleContext, paths ...string) ModuleOutPath {
	p := validatePathCtx(ctx, paths...)
	return ModuleOutPath{PathForOutput(ctx, ".intermediates", ctx.ModuleDir(), ctx.ModuleName(), ctx.ModuleSubDir(), p)}
}

// ModuleGenPath is a Path representing the 'gen' directory in a module's output
// directory. Mainly used for generated sources.
type ModuleGenPath struct {
	ModuleOutPath
	path          string
}
var _ Path = ModuleGenPath{}
var _ genPathProvider = ModuleGenPath{}
var _ objPathProvider = ModuleGenPath{}

// PathForModuleGen returns a Path representing the paths... under the module's
// `gen' directory.
func PathForModuleGen(ctx AndroidModuleContext, paths ...string) ModuleGenPath {
	p := validatePathCtx(ctx, paths...)
	return ModuleGenPath{
		PathForModuleOut(ctx, "gen", p),
		p,
	}
}

func (p ModuleGenPath) genPathWithExt(ctx AndroidModuleContext, ext string) ModuleGenPath {
	// TODO: make a different path for local vs remote generated files?
	return PathForModuleGen(ctx, pathtools.ReplaceExtension(p.path, ext))
}

func (p ModuleGenPath) objPathWithExt(ctx AndroidModuleContext, subdir, ext string) ModuleObjPath {
	return PathForModuleObj(ctx, subdir, pathtools.ReplaceExtension(p.path, ext))
}

// ModuleObjPath is a Path representing the 'obj' directory in a module's output
// directory. Used for compiled objects.
type ModuleObjPath struct {
	ModuleOutPath
}
var _ Path = ModuleObjPath{}

// PathForModuleObj returns a Path representing the paths... under the module's
// 'obj' directory.
func PathForModuleObj(ctx AndroidModuleContext, paths ...string) ModuleObjPath {
	p := validatePathCtx(ctx, paths...)
	return ModuleObjPath{PathForModuleOut(ctx, "obj", p)}
}

// ModuleResPath is a a Path representing the 'res' directory in a module's
// output directory.
type ModuleResPath struct {
	ModuleOutPath
}
var _ Path = ModuleResPath{}

// PathForModuleRes returns a Path representing the paths... under the module's
// 'res' directory.
func PathForModuleRes(ctx AndroidModuleContext, paths ...string) ModuleResPath {
	p := validatePathCtx(ctx, paths...)
	return ModuleResPath{PathForModuleOut(ctx, "res", p)}
}


// validateSafePath validates a path that we trust (may contain ninja variables).
// Ensures that it does not attempt to leave the containing directory.
func validateSafePath(paths ...string) (string, error) {
	// TODO: filepath.Join isn't necessarily correct with embedded ninja
	// variables. '..' may remove the entire ninja variable, even if it
	// will be expanded to multiple nested directories.
	p := filepath.Join(paths...)
	if p == ".." || strings.HasPrefix(p, "../") || strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("Path is outside directory: %s", p)
	}
	return p, nil
}

// validatePath validates that a path does not include ninja variables, and does
// not attempt to leave the containing directory.
func validatePath(paths ...string) (string, error) {
	for _, path := range paths {
		if strings.Contains(path, "$") {
			return "", fmt.Errorf("Path contains invalid character($): %s", path)
		}
	}
	return validateSafePath(paths...)
}

// validatePathCtx uses validatePath to validate the paths..., but reports any
// errors using ModuleErrorf.
func validatePathCtx(ctx AndroidModuleContext, paths ...string) string {
	p, err := validatePath(paths...)
	if err != nil {
		ctx.ModuleErrorf("%s", err.Error())
	}
	return p
}
