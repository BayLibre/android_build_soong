package bp2build

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"android/soong/shared"
)

type node struct {
	name     string
	present  bool
	children map[string]*node
}

func ensureNode(root *node, path string) *node {
	if path == "" {
		return root
	}

	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	dir, base := filepath.Split(path)
	dn := ensureNode(root, dir)
	if child, ok := dn.children[base]; ok {
		return child
	} else {
		dn.children[base] = &node{base, false, make(map[string]*node)}
		return dn.children[base]
	}
}

func treeFromPathList(paths []string) *node {
	result := &node{"", false, make(map[string]*node)}

	for _, p := range paths {
		ensureNode(result, p).present = true
	}

	return result
}

func readdirToMap(dir string) ([]os.FileInfo, map[string]os.FileInfo) {
	entryList, err := ioutil.ReadDir(dir)
	entryMap := make(map[string]os.FileInfo)

	if err != nil {
		if os.IsNotExist(err) {
			return make([]os.FileInfo, 0), entryMap
		} else {
			fmt.Fprintf(os.Stderr, "Cannot readdir '%s': %s\n", dir, err)
			os.Exit(1)
		}
	}

	for _, fi := range entryList {
		entryMap[fi.Name()] = fi
	}

	return entryList, entryMap
}

func symlinkIntoForest(topdir, dst, src string, exclude *node) {
	if exclude != nil && exclude.present {
		return
	}

	err := os.Symlink(shared.JoinPath(topdir, src), shared.JoinPath(topdir, dst))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create symlink at '%s' pointing to '%s': %s", dst, src, err)
		os.Exit(1)
	}
}

func plantSymlinkForestRecursive(topdir string, forestDir string, buildFilesDir string, srcDir string, exclude *node, acc *[]string) {
	if exclude != nil && exclude.present {
		return
	}

	*acc = append(*acc, srcDir)
	srcDirEntryList, srcDirEntryMap := readdirToMap(shared.JoinPath(topdir, srcDir))
	buildFilesEntryList, buildFilesEntryMap := readdirToMap(shared.JoinPath(topdir, buildFilesDir))

	allEntries := make(map[string]bool)
	for _, n := range srcDirEntryList {
		allEntries[n.Name()] = true
	}

	for _, n := range buildFilesEntryList {
		allEntries[n.Name()] = true
	}

	err := os.MkdirAll(shared.JoinPath(topdir, forestDir), 0777)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot mkdir '%s': %s\n", forestDir, err)
		os.Exit(1)
	}

	for f, _ := range allEntries {
		if f[0] == '.' {
			continue // Ignore dotfiles
		}

		fp := shared.JoinPath(forestDir, f)
		sp := shared.JoinPath(srcDir, f)
		bp := shared.JoinPath(buildFilesDir, f)
		var ce *node
		if exclude == nil {
			ce = nil
		} else {
			ce = exclude.children[f]
		}

		sf, sExists := srcDirEntryMap[f]
		bf, bExists := buildFilesEntryMap[f]

		if !sExists {
			if bf.IsDir() && ce != nil {
				plantSymlinkForestRecursive(topdir, fp, bp, sp, ce, acc)
			} else {
				// Not in the source tree, symlink BUILD file, carry on
				symlinkIntoForest(topdir, fp, bp, ce)
			}
		} else if !bExists {
			if sf.IsDir() && ce != nil {
				plantSymlinkForestRecursive(topdir, fp, bp, sp, ce, acc)
			} else {
				// Not in the build file tree, symlink source tree, carry on
				symlinkIntoForest(topdir, fp, sp, ce)
			}
		} else if sf.IsDir() && bf.IsDir() {
			// Both are directories. Descend.
			plantSymlinkForestRecursive(topdir, fp, bp, sp, ce, acc)
		} else {
			// Both exists and one is a file. This is an error.
			fmt.Fprintf(os.Stderr,
				"Conflict in workspace symlink tree creation: both '%s' and '%s' exist and at least one of them is a file\n",
				sp, bp)
			os.Exit(1)
		}
	}
}

// Creates a symlink forest by merging the directory tree at "buildFiles" and
// "srcDir" while excluding paths listed in "exclude".
func PlantSymlinkForest(topdir string, forest string, buildFiles string, srcDir string, exclude []string) []string {
	deps := make([]string, 0)
	os.RemoveAll(shared.JoinPath(topdir, forest))
	excludeTree := treeFromPathList(exclude)
	plantSymlinkForestRecursive(topdir, forest, buildFiles, srcDir, excludeTree, &deps)
	return deps
}
