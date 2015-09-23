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

	// Returns whether the path is in a writable area (not in the source dir)
	Writable() bool
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

// Derives a new file path in ctx's generated sources directory from the current
// path, but with the new extension.
func GenPathWithExt(ctx AndroidModuleContext, p Path, ext string) ModuleGenPath {
	if path, ok := p.(genPathProvider); ok {
		return path.genPathWithExt(ctx, ext)
	}
	ctx.ModuleErrorf("Tried to create generated file from unsupported path: %s(%s)", reflect.TypeOf(p).Name(), p)
	return PathForModuleGen(ctx)
}

// Derives a new file path in ctx's object directory from the current path, but
// with the new extension.
func ObjPathWithExt(ctx AndroidModuleContext, p Path, subdir, ext string) ModuleObjPath {
	if path, ok := p.(objPathProvider); ok {
		return path.objPathWithExt(ctx, subdir, ext)
	}
	ctx.ModuleErrorf("Tried to create object file from unsupported path: %s (%s)", reflect.TypeOf(p).Name(), p)
	return PathForModuleObj(ctx)
}

// Derives a new path in ctx's output resource directory, using the current path
// to create the directory name, and the `name` argument for the filename.
func ResPathWithName(ctx AndroidModuleContext, p Path, name string) ModuleResPath {
	if path, ok := p.(resPathProvider); ok {
		return path.resPathWithName(ctx, name)
	}
	ctx.ModuleErrorf("Tried to create object file from unsupported path: %s (%s)", reflect.TypeOf(p).Name(), p)
	return PathForModuleRes(ctx)
}

// A container that may or may not contain a valid Path.
type OptionalPath struct {
	valid bool
	path  Path
}

func OptionalPathForPath(path Path) OptionalPath {
	return OptionalPath{valid: true, path: path}
}

func (p OptionalPath) Valid() bool {
	return p.valid
}

func (p OptionalPath) Path() Path {
	if !p.valid {
		panic("Requesting an invalid path")
	}
	return p.path
}

func (p OptionalPath) String() string {
	if p.valid {
		return p.path.String()
	} else {
		return ""
	}
}

// A slice of Path objects, with helpers to operation on the collection.
type Paths []Path

// Return Paths rooted from SrcDir
func PathsForSource(ctx AndroidModuleContext, paths []string) Paths {
	ret := make(Paths, 0, len(paths))
	for _, path := range paths {
		ret = append(ret, PathForSource(ctx, path))
	}
	return ret
}

// Same as PathsForSource, but doesn't require a ModuleContext
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

// Return Paths rooted from the module's local source directory
func PathsForModuleSrc(ctx AndroidModuleContext, paths []string) Paths {
	ret := make(Paths, 0, len(paths))
	for _, path := range paths {
		ret = append(ret, PathForModuleSrc(ctx, path))
	}
	return ret
}

// Return Paths rooted from the module's local source directory, but
// strip the local source directory from the beginning of each string.
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

// Return Paths rooted from the module's local source directory. If
// none are provided, use the default if it exists.
func PathsWithOptionalDefaultForModuleSrc(ctx AndroidModuleContext, input []string, def string) Paths {
	if len(input) > 0 {
		return PathsForModuleSrc(ctx, input)
	}
	// Use Glob so that if the default doesn't exist, a dependency is added so that when it
	// is created, we're run again.
	path := filepath.Join(ctx.AConfig().srcDir, ctx.ModuleDir(), def)
	return ctx.Glob("default", path, []string{})
}

// Error out if one of the Paths is not writable (if it's in the source dir)
func (p Paths) EnsureWritable(ctx AndroidModuleContext) {
	for _, path := range p {
		if !path.Writable() {
			ctx.ModuleErrorf("Path %s(%s) cannot be written to", reflect.TypeOf(path).Name(), path)
		}
	}
}

// Return Paths in string form
func (p Paths) Strings() []string {
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

func (p basePath) Writable() bool {
	return false
}

// A Path representing a file path rooted from SrcDir
type SourcePath struct {
	basePath
}
var _ Path = SourcePath{}

// For paths that we expect are safe -- only for use by go code that is
// embedding ninja variables in paths
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

// Return a SourcePath for the provided paths... (which are joined together
// with filepath.Join). This also validates that the path doesn't escape the
// source dir, or is contained in the build dir. On error, it will return a
// usable, but invalid SourcePath, and report a ModuleError.
func PathForSource(ctx AndroidModuleContext, paths ...string) SourcePath {
	p, err := PathForSourceConfig(ctx.AConfig(), paths...)
	if err != nil {
		ctx.ModuleErrorf("%s", err.Error())
	}
	return p
}

// Same as PathForSource, but doesn't require an AndroidModuleContext
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

// Creates a new SourcePath with paths... joined with the current path. The
// provided paths... may not use '..' to escape from the current path.
func (p SourcePath) Join(paths ...string) (SourcePath, error) {
	path, err := validatePath(paths...)
	if err != nil {
		return p, err
	}
	return PathForSourceConfig(p.config, p.path, path)
}

// Same as Join, but takes an AndroidModuleContext for error reporting.
func (p SourcePath) JoinCtx(ctx AndroidModuleContext, paths ...string) SourcePath {
	path := validatePathCtx(ctx, paths...)
	return PathForSource(ctx, p.path, path)
}

// Assuming that we're a resource overlay directory, return the overlay for
// `path`, if it exists.
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

// A Path representing a file path rooted from the build directory
type OutputPath struct {
	basePath
}
var _ Path = OutputPath{}

// Return a OutputPath for the provided paths... (which are joined together
// with filepath.Join). This also validates that the path doesn't escape
// the build dir. On error, it will return a usable, but invalid OutputPath,
// and report a ModuleError.
func PathForOutput(ctx AndroidModuleContext, paths ...string) OutputPath {
	path := validatePathCtx(ctx, paths...)
	return OutputPath{basePath{path,ctx.AConfig()}}
}

// Same as PathForOutput, but doesn't require an AndroidModuleContext
func PathForOutputConfig(config Config, paths ...string) (OutputPath, error) {
	path, err := validatePath(paths...)
	return OutputPath{basePath{path,config}}, err
}

func (p OutputPath) Writable() bool {
	return true
}

func (p OutputPath) String() string {
	return filepath.Join(p.config.buildDir, p.path)
}

// Creates a new OutputPath with paths... joined with the current path. The
// provided paths... may not use '..' to escape from the current path.
func (p OutputPath) Join(paths ...string) (OutputPath, error) {
	path, err := validatePath(paths...)
	if err != nil {
		return p, err
	}
	return PathForOutputConfig(p.config, p.path, path)
}

// Returns the OutputPath representing the top-level intermediates directory.
func PathForIntermediatesConfig(config Config, paths ...string) (OutputPath, error) {
	path, err := validatePath(paths...)
	if err != nil {
		return OutputPath{basePath{"",config}}, err
	}
	return PathForOutputConfig(config, ".intermediates", path)
}

// A Path representing a file path rooted from a module's local source dir
type ModuleSrcPath struct {
	basePath
	sourcePath SourcePath
	moduleDir  string
}
var _ Path = ModuleSrcPath{}
var _ genPathProvider = ModuleSrcPath{}
var _ objPathProvider = ModuleSrcPath{}
var _ resPathProvider = ModuleSrcPath{}

// Returns a ModuleSrcPath representing the path p... under the module's
// local source directory.
func PathForModuleSrc(ctx AndroidModuleContext, p ...string) ModuleSrcPath {
	path := validatePathCtx(ctx, p...)
	return ModuleSrcPath{basePath{path,ctx.AConfig()}, PathForSource(ctx, ctx.ModuleDir(), path), ctx.ModuleDir()}
}

// Returns an OptionalPath wrapping a ModuleSrcPath if p is non-nil.
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

// A Path representing a module's output directory.
type ModuleOutPath struct {
	OutputPath
}
var _ Path = ModuleOutPath{}

func PathForModuleOut(ctx AndroidModuleContext, paths ...string) ModuleOutPath {
	p := validatePathCtx(ctx, paths...)
	return ModuleOutPath{PathForOutput(ctx, ".intermediates", ctx.ModuleDir(), ctx.ModuleName(), ctx.ModuleSubDir(), p)}
}

// A Path representing the 'gen' directory in a module's output directory.
// Mainly used for generated sources.
type ModuleGenPath struct {
	ModuleOutPath
	path          string
}
var _ Path = ModuleGenPath{}
var _ genPathProvider = ModuleGenPath{}
var _ objPathProvider = ModuleGenPath{}

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

// A Path representing the 'obj' directory in a module's output directory.
// Used for compiled objects.
type ModuleObjPath struct {
	ModuleOutPath
}
var _ Path = ModuleObjPath{}

func PathForModuleObj(ctx AndroidModuleContext, paths ...string) ModuleObjPath {
	p := validatePathCtx(ctx, paths...)
	return ModuleObjPath{PathForModuleOut(ctx, "obj", p)}
}

// A Path representing the 'res' directory in a module's output directory.
type ModuleResPath struct {
	ModuleOutPath
}
var _ Path = ModuleResPath{}

func PathForModuleRes(ctx AndroidModuleContext, paths ...string) ModuleResPath {
	p := validatePathCtx(ctx, paths...)
	return ModuleResPath{PathForModuleOut(ctx, "res", p)}
}


// Returns a func to pass to blueprint.PackageContext.VariableFunc that will
// return the joined SrcDir and path.
func SourcePathVariableFunc(path string) func(config interface{}) (string, error) {
	return func(config interface{}) (string, error) {
		p, err := safePathForSourceConfig(config.(Config), path)
		if err != nil {
			return "", err
		}
		return p.String(), nil
	}
}

// Returns a func to pass to blueprint.PackageContext.VariableFunc that
// will return the HostBinTool path.
func HostBinToolVariableFunc(path string) func(config interface{}) (string, error) {
	return func(config interface{}) (string, error) {
		p, err := config.(Config).HostBinTool(path)
		if err != nil {
			return "", err
		}
		return p.String(), nil
	}
}

// Returns a func to pass to blueprint.PackageContext.VariableFunc that
// will return the HostJavaTool path.
func HostJavaToolVariableFunc(path string) func(config interface{}) (string, error) {
	return func(config interface{}) (string, error) {
		p, err := config.(Config).HostJavaTool(path)
		if err != nil {
			return "", err
		}
		return p.String(), nil
	}
}

// Returns a func to pass to blueprint.PackageContext.VariableFunc that
// will return a string representing the intermediates path
func IntermediatesVariableFunc(path string) func(config interface{}) (string, error) {
	return func(config interface{}) (string, error) {
		p, err := PathForIntermediatesConfig(config.(Config), path)
		if err != nil {
			return "", err
		}
		return p.String(), err
	}
}


// Validate a path that we trust (may contain ninja variables). Ensures that it
// does not attempt to leave the containing directory.
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

// Validate that a path does not include ninja variables, and does not attempt
// to leave the containing directory.
func validatePath(paths ...string) (string, error) {
	for _, path := range paths {
		if strings.Contains(path, "$") {
			return "", fmt.Errorf("Path contains invalid character($): %s", path)
		}
	}
	return validateSafePath(paths...)
}

func validatePathCtx(ctx AndroidModuleContext, paths ...string) string {
	p, err := validatePath(paths...)
	if err != nil {
		ctx.ModuleErrorf("%s", err.Error())
	}
	return p
}
