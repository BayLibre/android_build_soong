// Copyright 2022 Google Inc. All rights reserved.
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

package bp2build

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"

	"android/soong/shared"
)

// A tree structure that describes what to do at each directory in the created
// symlink tree. Currently it is used to enumerate which files/directories
// should be excluded from symlinking. Each instance of "node" represents a file
// or a directory. If excluded is true, then that file/directory should be
// excluded from symlinking. Otherwise, the node is not excluded, but one of its
// descendants is (otherwise the node in question would not exist)

type instructionsNode struct {
	name     string
	excluded bool // If false, this is just an intermediate node
	children map[string]*instructionsNode
}

type symlinkForestContext struct {
	verbose bool
	topdir  string // $TOPDIR

	// State
	wg    sync.WaitGroup
	depCh chan string
	okay  atomic.Bool // Whether the forest was successfully constructed
}

// A simple thread pool to limit concurrency on system calls.
// Necessary because Go spawns a new OS-level thread for each blocking system
// call. This means that if syscalls are too slow and there are too many of
// them, the hard limit on OS-level threads can be exhausted.
type syscallPool struct {
	shutdownCh []chan<- struct{}
	workCh     chan syscall
}

type syscall struct {
	work func()
	done chan<- struct{}
}

func createSyscallPool(count int) *syscallPool {
	result := &syscallPool{
		shutdownCh: make([]chan<- struct{}, count),
		workCh:     make(chan syscall),
	}

	for i := 0; i < count; i++ {
		shutdownCh := make(chan struct{})
		result.shutdownCh[i] = shutdownCh
		go result.worker(shutdownCh)
	}

	return result
}

func (p *syscallPool) do(work func()) {
	doneCh := make(chan struct{})
	p.workCh <- syscall{work, doneCh}
	<-doneCh
}

func (p *syscallPool) shutdown() {
	for _, ch := range p.shutdownCh {
		ch <- struct{}{} // Blocks until the value is received
	}
}

func (p *syscallPool) worker(shutdownCh <-chan struct{}) {
	for {
		select {
		case <-shutdownCh:
			return
		case work := <-p.workCh:
			work.work()
			work.done <- struct{}{}
		}
	}
}

// Ensures that the node for the given path exists in the tree and returns it.
func ensureNodeExists(root *instructionsNode, path string) *instructionsNode {
	if path == "" {
		return root
	}

	if path[len(path)-1] == '/' {
		path = path[:len(path)-1] // filepath.Split() leaves a trailing slash
	}

	dir, base := filepath.Split(path)

	// First compute the parent node...
	dn := ensureNodeExists(root, dir)

	// then create the requested node as its direct child, if needed.
	if child, ok := dn.children[base]; ok {
		return child
	} else {
		dn.children[base] = &instructionsNode{base, false, make(map[string]*instructionsNode)}
		return dn.children[base]
	}
}

// Turns a list of paths to be excluded into a tree
func instructionsFromExcludePathList(paths []string) *instructionsNode {
	result := &instructionsNode{"", false, make(map[string]*instructionsNode)}

	for _, p := range paths {
		ensureNodeExists(result, p).excluded = true
	}

	return result
}

func mergeBuildFiles(output string, srcBuildFile string, generatedBuildFile string, verbose bool) error {

	srcBuildFileContent, err := os.ReadFile(srcBuildFile)
	if err != nil {
		return err
	}

	generatedBuildFileContent, err := os.ReadFile(generatedBuildFile)
	if err != nil {
		return err
	}

	// There can't be a package() call in both the source and generated BUILD files.
	// bp2build will generate a package() call for licensing information, but if
	// there's no licensing information, it will still generate a package() call
	// that just sets default_visibility=public. If the handcrafted build file
	// also has a package() call, we'll allow it to override the bp2build
	// generated one if it doesn't have any licensing information. If the bp2build
	// one has licensing information and the handcrafted one exists, we'll leave
	// them both in for bazel to throw an error.
	packageRegex := regexp.MustCompile(`(?m)^package\s*\(`)
	packageDefaultVisibilityRegex := regexp.MustCompile(`(?m)^package\s*\(\s*default_visibility\s*=\s*\[\s*"//visibility:public",?\s*]\s*\)`)
	if packageRegex.Find(srcBuildFileContent) != nil {
		if verbose && packageDefaultVisibilityRegex.Find(generatedBuildFileContent) != nil {
			fmt.Fprintf(os.Stderr, "Both '%s' and '%s' have a package() target, removing the first one\n",
				generatedBuildFile, srcBuildFile)
		}
		generatedBuildFileContent = packageDefaultVisibilityRegex.ReplaceAll(generatedBuildFileContent, []byte{})
	}

	outFile, err := os.Create(output)
	if err != nil {
		return err
	}

	_, err = outFile.Write(generatedBuildFileContent)
	if err != nil {
		return err
	}

	if generatedBuildFileContent[len(generatedBuildFileContent)-1] != '\n' {
		_, err = outFile.WriteString("\n")
		if err != nil {
			return err
		}
	}

	_, err = outFile.Write(srcBuildFileContent)
	return err
}

// Calls readdir() and returns it as a map from the basename of the files in dir
// to os.FileInfo.
func readdirToMap(dir string) map[string]os.FileInfo {
	entryList, err := ioutil.ReadDir(dir)
	result := make(map[string]os.FileInfo)

	if err != nil {
		if os.IsNotExist(err) {
			// It's okay if a directory doesn't exist; it just means that one of the
			// trees to be merged contains parts the other doesn't
			return result
		} else {
			fmt.Fprintf(os.Stderr, "Cannot readdir '%s': %s\n", dir, err)
			os.Exit(1)
		}
	}

	for _, fi := range entryList {
		result[fi.Name()] = fi
	}

	return result
}

// Creates a symbolic link at dst pointing to src
func symlinkIntoForest(topdir, dst, src string) {
	srcPath := shared.JoinPath(topdir, src)
	dstPath := shared.JoinPath(topdir, dst)

	// Check if a symlink already exists. Remove it and recreate the symlink
	// if the old symlink is pointing to the wrong place.
	_, err := os.Lstat(dstPath)
	if err == nil {
		if origSrcPath, err := os.Readlink(dstPath); err == nil {
			if origSrcPath == srcPath {
				// Symlink already exists and is pointing to the right place.
				return
			} else {
				if err := os.Remove(dstPath); err != nil {
					fmt.Fprintf(os.Stderr, "Cannot unlink '%s': %s\n", dstPath, err)
					os.Exit(1)
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "Cannot resolve existing symlink at '%s': %s", dstPath, err)
			os.Exit(1)
		}
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Cannot determine if symlink exists at '%s': %s", dst, err)
		os.Exit(1)
	}

	// Create symlink.
	if err := os.Symlink(srcPath, dstPath); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create symlink at '%s' pointing to '%s': %s", dst, src, err)
		os.Exit(1)
	}
}

func isDir(path string, fi os.FileInfo) bool {
	if (fi.Mode() & os.ModeSymlink) != os.ModeSymlink {
		return fi.IsDir()
	}

	fi2, statErr := os.Stat(path)
	if statErr == nil {
		return fi2.IsDir()
	}

	// Check if this is a dangling symlink. If so, treat it like a file, not a dir.
	_, lstatErr := os.Lstat(path)
	if lstatErr != nil {
		fmt.Fprintf(os.Stderr, "Cannot stat or lstat '%s': %s\n%s\n", path, statErr, lstatErr)
		os.Exit(1)
	}

	return false
}

// Recursively plants a symlink forest at forestDir. The symlink tree will
// contain every file in buildFilesDir and srcDir excluding the files in
// instructions. Collects every directory encountered during the traversal of
// srcDir .
func plantSymlinkForestRecursive(context *symlinkForestContext, instructions *instructionsNode, forestDir string, buildFilesDir string, srcDir string) {
	defer context.wg.Done()

	if instructions != nil && instructions.excluded {
		// Excluded paths are skipped at the level of the non-excluded parent.
		fmt.Fprintf(os.Stderr, "may not specify a root-level exclude directory")
		context.okay.Store(false)
		return
	}

	// We don't add buildFilesDir here because the bp2build files marker files is
	// already a dependency which covers it. If we ever wanted to turn this into
	// a generic symlink forest creation tool, we'd need to add it, too.
	context.depCh <- srcDir

	srcDirMap := readdirToMap(shared.JoinPath(context.topdir, srcDir))
	buildFilesMap := readdirToMap(shared.JoinPath(context.topdir, buildFilesDir))

	renamingBuildFile := false
	if _, ok := srcDirMap["BUILD"]; ok {
		if _, ok := srcDirMap["BUILD.bazel"]; !ok {
			if _, ok := buildFilesMap["BUILD.bazel"]; ok {
				renamingBuildFile = true
				srcDirMap["BUILD.bazel"] = srcDirMap["BUILD"]
				delete(srcDirMap, "BUILD")
			}
		}
	}

	instructionsChildren := map[string]*instructionsNode{}
	if instructions != nil {
		for k, v := range instructions.children {
			instructionsChildren[k] = v
		}
		if renamingBuildFile {
			instructionsChildren["BUILD.bazel"] = instructionsChildren["BUILD"]
		}
	}

	allEntries := make(map[string]struct{})
	for n := range srcDirMap {
		allEntries[n] = struct{}{}
	}

	for n := range buildFilesMap {
		allEntries[n] = struct{}{}
	}

	err := os.MkdirAll(shared.JoinPath(context.topdir, forestDir), 0777)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot mkdir '%s': %s\n", forestDir, err)
		os.Exit(1)
	}

	// Start with a list of items that already exist in the forest, and remove
	// each element as it is processed in allEntries. Any remaining items in
	// forestMapForDeletion must be removed. (This handles files which were
	// removed since the previous forest generation).
	forestMapForDeletion := readdirToMap(shared.JoinPath(context.topdir, forestDir))

	for f := range allEntries {
		if f[0] == '.' {
			continue // Ignore dotfiles
		}
		delete(forestMapForDeletion, f)

		// The full paths of children in the input trees and in the output tree
		forestChild := shared.JoinPath(forestDir, f)
		srcChild := shared.JoinPath(srcDir, f)
		if f == "BUILD.bazel" && renamingBuildFile {
			srcChild = shared.JoinPath(srcDir, "BUILD")
		}
		buildFilesChild := shared.JoinPath(buildFilesDir, f)

		// Descend in the instruction tree if it exists
		instructionsChild := instructionsChildren[f]

		srcChildEntry, sExists := srcDirMap[f]
		buildFilesChildEntry, bExists := buildFilesMap[f]

		if instructionsChild != nil && instructionsChild.excluded {
			if bExists {
				symlinkIntoForest(context.topdir, forestChild, buildFilesChild)
			}
			continue
		}

		sDir := sExists && isDir(shared.JoinPath(context.topdir, srcChild), srcChildEntry)
		bDir := bExists && isDir(shared.JoinPath(context.topdir, buildFilesChild), buildFilesChildEntry)

		if !sExists {
			if bDir && instructionsChild != nil {
				// Not in the source tree, but we have to exclude something from under
				// this subtree, so descend
				context.wg.Add(1)
				go plantSymlinkForestRecursive(context, instructionsChild, forestChild, buildFilesChild, srcChild)
			} else {
				// Not in the source tree, symlink BUILD file
				symlinkIntoForest(context.topdir, forestChild, buildFilesChild)
			}
		} else if !bExists {
			if sDir && instructionsChild != nil {
				// Not in the build file tree, but we have to exclude something from
				// under this subtree, so descend
				context.wg.Add(1)
				go plantSymlinkForestRecursive(context, instructionsChild, forestChild, buildFilesChild, srcChild)
			} else {
				// Not in the build file tree, symlink source tree, carry on
				symlinkIntoForest(context.topdir, forestChild, srcChild)
			}
		} else if sDir && bDir {
			// Both are directories. Descend.
			context.wg.Add(1)
			go plantSymlinkForestRecursive(context, instructionsChild, forestChild, buildFilesChild, srcChild)
		} else if !sDir && !bDir {
			// Neither is a directory. Merge them.
			srcBuildFile := shared.JoinPath(context.topdir, srcChild)
			generatedBuildFile := shared.JoinPath(context.topdir, buildFilesChild)
			// The Android.bp file that codegen used to produce `buildFilesChild` is
			// already a dependency, we can ignore `buildFilesChild`.
			context.depCh <- srcChild
			err = mergeBuildFiles(shared.JoinPath(context.topdir, forestChild), srcBuildFile, generatedBuildFile, context.verbose)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error merging %s and %s: %s",
					srcBuildFile, generatedBuildFile, err)
				context.okay.Store(false)
			}
		} else {
			// Both exist and one is a file. This is an error.
			fmt.Fprintf(os.Stderr,
				"Conflict in workspace symlink tree creation: both '%s' and '%s' exist and exactly one is a directory\n",
				srcChild, buildFilesChild)
			context.okay.Store(false)
		}
	}

	// Remove all files in the forest that do not exist in either the source
	// tree nor the build file tree. (This handles files which were removed
	// since the previous forest generation).
	for f := range forestMapForDeletion {
		instructionsChild := instructionsChildren[f]

		if instructionsChild != nil && instructionsChild.excluded {
			// This directory may be excluded because bazel writes to it under the
			// forest root. Thus this path is intentionally left alone.
			continue
		}
		forestChild := shared.JoinPath(context.topdir, forestDir, f)
		if err := os.Remove(forestChild); err != nil {
			fmt.Fprintf(os.Stderr, "Cannot remove '%s' from symlink forest: %s\n", forestChild, err)
			os.Exit(1)
		}
	}
}

// Creates a symlink forest by merging the directory tree at "buildFiles" and
// "srcDir" while excluding paths listed in "exclude". Returns the set of paths
// under srcDir on which readdir() had to be called to produce the symlink
// forest.
func PlantSymlinkForest(verbose bool, topdir string, forest string, buildFiles string, exclude []string) []string {
	context := &symlinkForestContext{
		verbose: verbose,
		topdir:  topdir,
		depCh:   make(chan string),
	}

	context.okay.Store(true)

	instructions := instructionsFromExcludePathList(exclude)
	go func() {
		context.wg.Add(1)
		plantSymlinkForestRecursive(context, instructions, forest, buildFiles, ".")
		context.wg.Wait()
		close(context.depCh)
	}()

	deps := make([]string, 0)
	for dep := range context.depCh {
		deps = append(deps, dep)
	}

	if !context.okay.Load() {
		os.Exit(1)
	}

	return deps
}
