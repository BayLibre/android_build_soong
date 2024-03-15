// Copyright 2024 Google Inc. All rights reserved.
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

package elf

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func UpdateBuildIdDir(path string) error {
	var symbolFiles []string
	path = filepath.Clean(path)
	buildIdPath := path + "/.build-id"
	var symbolsMtime, buildIdMtime time.Time
	filepath.WalkDir(path, func(path string, entry fs.DirEntry, err error) error {
		if entry == nil || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		mtime := info.ModTime()
		if strings.HasPrefix(path, buildIdPath) {
			if buildIdMtime.Compare(mtime) < 0 {
				buildIdMtime = mtime
			}
		} else {
			if symbolsMtime.Compare(mtime) < 0 {
				symbolsMtime = mtime
			}
			symbolFiles = append(symbolFiles, path)
		}
		return nil
	})
	if symbolsMtime.Compare(buildIdMtime) < 0 {
		return nil
	}

	concurrency := 8
	done := make(chan error)
	buildIdToFile := make(map[string]string)
	var mu sync.Mutex
	for i := 0; i != concurrency; i++ {
		go func(paths []string) {
			for _, path := range paths {
				id, err := Identifier(path, true)
				if err != nil {
					done <- err
					return
				}
				if id == "" {
					continue
				}
				mu.Lock()
				oldPath := buildIdToFile[id]
				if oldPath == "" || oldPath > path {
					buildIdToFile[id] = path
				}
				mu.Unlock()
			}
			done <- nil
		}(symbolFiles[len(symbolFiles)*i/concurrency : len(symbolFiles)*(i+1)/concurrency])
	}
	for i := 0; i != concurrency; i++ {
		err := <-done
		if err != nil {
			return err
		}
	}
	err := os.RemoveAll(buildIdPath)
	if err != nil {
		return err
	}
	for id, path := range buildIdToFile {
		symlinkDir := buildIdPath + "/" + id[:2]
		symlinkPath := symlinkDir + "/" + id[2:] + ".debug"
		if err := os.MkdirAll(symlinkDir, 0755); err != nil {
			return err
		}

		target, err := filepath.Rel(symlinkDir, path)
		if err != nil {
			return err
		}

		if err := os.Symlink(target, symlinkPath); err != nil {
			return err
		}
	}
	return nil
}
