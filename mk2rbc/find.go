package mk2rbc

import (
	"io/fs"
	"os"
	"strings"
)

func match_path(path string, chunks []string) bool {
	switch len(chunks) {
	case 0:
		return false
	case 1:
		return strings.HasSuffix(path, chunks[0]) &&
			(path == chunks[0] || path[len(path)-len(chunks[0])-1] == os.PathSeparator)
	default:
		chunk := chunks[0]
		index := strings.Index(path, chunk)
		for index == 0 || (index > 0 && path[index-1] == os.PathSeparator) {
			after := index + len(chunk)
			if after < len(path) && path[after] == os.PathSeparator && match_path(path[after+1:], chunks[1:]) {
				return true
			}
			index = strings.Index(path[index+1:], chunk)
		}
		return false
	}
}

// findMatchingPaths returns all the paths in given file system that match
// chunk0/**/chunk1/**/.../chunkN pattern. Only plain files, and only those
// outside .xxx directories are returned.
func findMatchingPaths(fsys fs.FS, chunks []string) []string {
	var match []string
	var root string

	if chunks[0] != "" {
		root = chunks[0]
	} else {
		root = "."
	}
	match = chunks[1:]

	var result []string
	fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Do not recurse into dot directories
			if path[0] == '.' && len(path) > 1 {
				return fs.SkipDir
			}
			dotIndex := strings.LastIndex(path, ".")
			if dotIndex > 0 && path[dotIndex-1] == os.PathSeparator {
				return fs.SkipDir
			}
			// We are looking only for the files
			return nil
		}
		if match_path(path, match) {
			result = append(result, path)
		}
		return nil
	})
	return result
}
