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

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

// TestReadFile reads the test file below and ensures that the GoPackage
// tree is set up with the correct Name/Deps/Files.
func TestReadFile(t *testing.T) {
	t.Parallel()

	setupDir(t, func(dir string, pkg *GoPackage) {
		if pkg.Name != "main" {
			t.Errorf("Expected top package to be called `main`, but is called: %q", pkg.Name)
		}
		if len(pkg.Deps) != 2 {
			t.Fatalf("Expected main package to have two dependencies")
		}
		if pkg.Deps[0].Name != "android/soong/a" {
			t.Errorf("Expected first dependency to be `android/soong/a`, but got: %q", pkg.Deps[0].Name)
		}
		if pkg.Deps[1].Name != "android/soong/b" {
			t.Errorf("Expected first dependency to be `android/soong/b`, but got: %q", pkg.Deps[1].Name)
		}

		if len(pkg.Deps[0].Deps) != 0 {
			t.Errorf("Expected android/soong/a to have no dependencies, but got: %#v", pkg.Deps[0].Deps)
		}

		if len(pkg.Deps[1].Deps) != 1 {
			t.Errorf("Expected android/soong/b to have one dependency to android/soong/a, but got: %#v", pkg.Deps[1].Deps)
		}

		if pkg.Deps[1].Deps[0] != pkg.Deps[0] {
			t.Errorf("Expected android/soong/b to depend on android/soong/a, but got %#v", pkg.Deps[1].Deps[0])
		}

		if !reflect.DeepEqual(pkg.Files, []string{"main/main.go"}) {
			t.Errorf("Expected main package files to be main/main.go, but got: %v", pkg.Files)
		}

		if !reflect.DeepEqual(pkg.Deps[0].Files, []string{"a/a.go", "a/b.go"}) {
			t.Errorf("Expected android/soong/a package files to be a/a.go, a/b.go, but got: %v", pkg.Deps[0].Files)
		}
		if !reflect.DeepEqual(pkg.Deps[1].Files, []string{"b/a.go"}) {
			t.Errorf("Expected android/soong/b package files to be b/a.go, but got: %v", pkg.Deps[1].Files)
		}
	})
}

// TestSingleBuild ensures that just a basic build works.
func TestSingleBuild(t *testing.T) {
	t.Parallel()

	setupDir(t, func(dir string, pkg *GoPackage) {
		// The output binary
		out := filepath.Join(dir, "out", "test")

		if err := pkg.Compile(dir, filepath.Join(dir, "out")); err != nil {
			t.Fatalf("Got error when compiling:", err)
		}

		if err := pkg.Link(out); err != nil {
			t.Fatal("Got error when linking:", err)
		}

		if _, err := os.Stat(out); err != nil {
			t.Error("Cannot stat output:", err)
		}
	})
}

// testBuildAgain triggers two builds, running the modify function in between
// each build. It verifies that the second build did or did not actually need
// to rebuild anything based on the shouldRebuild argument.
func testBuildAgain(t *testing.T,
	shouldRecompile, shouldRelink bool,
	modify func(dir string),
	after func(pkg *GoPackage)) {

	t.Parallel()

	setupDir(t, func(dir string, pkg *GoPackage) {
		// The output binary
		out := filepath.Join(dir, "out", "test")

		if err := pkg.Compile(dir, filepath.Join(dir, "out")); err != nil {
			t.Fatal("Got error when compiling:", err)
		}

		if err := pkg.Link(out); err != nil {
			t.Fatal("Got error when linking:", err)
		}

		var firstTime time.Time
		if stat, err := os.Stat(out); err == nil {
			firstTime = stat.ModTime()
		} else {
			t.Fatal("Failed to stat output file:", err)
		}

		// mtime on HFS+ (the filesystem on darwin) are stored with 1
		// second granularity, so the timestamp checks will fail unless
		// we wait at least a second. Sleeping 1.1s to be safe.
		if runtime.GOOS == "darwin" {
			time.Sleep(1100 * time.Millisecond)
		}

		modify(dir)

		pkg, err := readFile(filepath.Join(dir, "test.fact"))
		if err != nil {
			t.Fatal("Got error while reading test.fact:", err)
		}

		if err := pkg.Compile(dir, filepath.Join(dir, "out")); err != nil {
			t.Fatal("Got error when compiling:", err)
		}
		if shouldRecompile {
			if !pkg.rebuilt {
				t.Fatal("Package should have recompiled, but was not recompiled.")
			}
		} else {
			if pkg.rebuilt {
				t.Fatal("Package should not have needed to be recompiled, but was recompiled.")
			}
		}

		if err := pkg.Link(out); err != nil {
			t.Fatal("Got error while linking:", err)
		}
		if shouldRelink {
			if !pkg.rebuilt {
				t.Error("Package should have relinked, but was not relinked.")
			}
		} else {
			if pkg.rebuilt {
				t.Error("Package should not have needed to be relinked, but was relinked.")
			}
		}

		if stat, err := os.Stat(out); err == nil {
			if shouldRelink {
				if stat.ModTime() == firstTime {
					t.Error("Output timestamp should be different, but both were", firstTime)
				}
			} else {
				if stat.ModTime() != firstTime {
					t.Error("Output timestamp should be the same.")
					t.Error(" first:", firstTime)
					t.Error("second:", stat.ModTime())
				}
			}
		} else {
			t.Fatal("Failed to stat output file:", err)
		}

		after(pkg)
	})
}

// TestRebuildAfterNoChanges ensures that we don't rebuild if nothing
// changes
func TestRebuildAfterNoChanges(t *testing.T) {
	testBuildAgain(t, false, false, func(dir string) {}, func(pkg *GoPackage) {})
}

// TestRebuildAfterTimestamp ensures that we don't rebuild because
// timestamps of important files have changed. We should only rebuild if the
// content hashes are different.
func TestRebuildAfterTimestampChange(t *testing.T) {
	testBuildAgain(t, false, false, func(dir string) {
		// Ensure that we've spent some amount of time asleep
		time.Sleep(100 * time.Millisecond)

		newTime := time.Now().Local()
		os.Chtimes(filepath.Join(dir, "test.fact"), newTime, newTime)
		os.Chtimes(filepath.Join(dir, "main/main.go"), newTime, newTime)
		os.Chtimes(filepath.Join(dir, "a/a.go"), newTime, newTime)
		os.Chtimes(filepath.Join(dir, "a/b.go"), newTime, newTime)
		os.Chtimes(filepath.Join(dir, "b/a.go"), newTime, newTime)
	}, func(pkg *GoPackage) {})
}

// TestRebuildAfterNoopFactChange ensures that we don't rebuild after a change
// to the .fact file that doesn't matter.
func TestRebuildAfterNoopFactChange(t *testing.T) {
	testBuildAgain(t, false, false, func(dir string) {
		if err := ioutil.WriteFile(filepath.Join(dir, "test.fact"), []byte(factFile+"\n"), 0666); err != nil {
			t.Fatal("Error writing test.fact:", err)
		}
	}, func(pkg *GoPackage) {})
}

// TestRebuildAfterFactFileRemoval ensures that we rebuild after a change to
// the .fact file that removes an unnecessary go file.
func TestRebuildAfterFactFileRemoval(t *testing.T) {
	testBuildAgain(t, true, true, func(dir string) {
		if err := ioutil.WriteFile(filepath.Join(dir, "test.fact"), []byte(factFile2), 0666); err != nil {
			t.Fatal("Error writing test.fact:", err)
		}
	}, func(pkg *GoPackage) {})
}

// TestRebuildAfterGoChange ensures that we rebuild after a content change
// to a package's go file.
func TestRebuildAfterGoChange(t *testing.T) {
	testBuildAgain(t, true, true, func(dir string) {
		if err := ioutil.WriteFile(filepath.Join(dir, "a", "a.go"), []byte(go_a_a+"\n"), 0666); err != nil {
			t.Fatal("Error writing a/a.go:", err)
		}
	}, func(pkg *GoPackage) {
		if !pkg.Deps[0].rebuilt {
			t.Fatal("android/soong/a should have rebuilt")
		}
		if !pkg.Deps[1].rebuilt {
			t.Fatal("android/soong/b should have rebuilt")
		}
	})
}

// TestRebuildAfterMainChange ensures that we don't rebuild any dependencies
// if only the main package's go files are touched.
func TestRebuildAfterMainChange(t *testing.T) {
	testBuildAgain(t, true, true, func(dir string) {
		if err := ioutil.WriteFile(filepath.Join(dir, "main", "main.go"), []byte(go_main_main+"\n"), 0666); err != nil {
			t.Fatal("Error writing main/main.go:", err)
		}
	}, func(pkg *GoPackage) {
		if pkg.Deps[0].rebuilt {
			t.Fatal("android/soong/a should not have rebuilt")
		}
		if pkg.Deps[1].rebuilt {
			t.Fatal("android/soong/b should not have rebuilt")
		}
	})
}

// TestRebuildAfterRemoveOut ensures that we rebuild if the output file is
// missing, even if everything else doesn't need rebuilding.
func TestRebuildAfterRemoveOut(t *testing.T) {
	testBuildAgain(t, false, true, func(dir string) {
		if err := os.Remove(filepath.Join(dir, "out", "test")); err != nil {
			t.Fatal("Failed to remove output:", err)
		}
	}, func(pkg *GoPackage) {})
}

// TestRebuildAfterPartialBuild ensures that even if the build was interrupted
// between the recompile and relink stages, we'll still relink when we run again.
func TestRebuildAfterPartialBuild(t *testing.T) {
	testBuildAgain(t, false, true, func(dir string) {
		if err := ioutil.WriteFile(filepath.Join(dir, "main", "main.go"), []byte(go_main_main+"\n"), 0666); err != nil {
			t.Fatal("Error writing main/main.go:", err)
		}

		pkg, err := readFile(filepath.Join(dir, "test.fact"))
		if err != nil {
			t.Fatal("Got error while reading test.fact:", err)
		}

		if err := pkg.Compile(dir, filepath.Join(dir, "out")); err != nil {
			t.Fatal("Got error when compiling:", err)
		}
		if !pkg.rebuilt {
			t.Fatal("Package should have recompiled, but was not recompiled.")
		}
	}, func(pkg *GoPackage) {})
}

// BenchmarkInitialBuild computes how long a clean build takes (for tiny test
// inputs).
func BenchmarkInitialBuild(b *testing.B) {
	for i := 0; i < b.N; i++ {
		setupDir(b, func(dir string, pkg *GoPackage) {
			if err := pkg.Compile(dir, filepath.Join(dir, "out")); err != nil {
				b.Fatal("Got error when compiling:", err)
			}

			if err := pkg.Link(filepath.Join(dir, "out", "test")); err != nil {
				b.Fatal("Got error when linking:", err)
			}
		})
	}
}

// BenchmarkMinIncrementalBuild computes how long an incremental build that
// doesn't actually need to build anything takes.
func BenchmarkMinIncrementalBuild(b *testing.B) {
	setupDir(b, func(dir string, pkg *GoPackage) {
		if err := pkg.Compile(dir, filepath.Join(dir, "out")); err != nil {
			b.Fatal("Got error when compiling:", err)
		}

		if err := pkg.Link(filepath.Join(dir, "out", "test")); err != nil {
			b.Fatal("Got error when linking:", err)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			pkg, err := readFile(filepath.Join(dir, "test.fact"))
			if err != nil {
				b.Fatal("Got error while reading test.fact:", err)
			}

			if err := pkg.Compile(dir, filepath.Join(dir, "out")); err != nil {
				b.Fatal("Got error when compiling:", err)
			}

			if err := pkg.Link(filepath.Join(dir, "out", "test")); err != nil {
				b.Fatal("Got error when linking:", err)
			}

			if pkg.rebuilt {
				b.Fatal("Should not have rebuilt anything")
			}
		}
	})
}

///////////////////////////////////////////////////////
// Templates used to create fake compilable packages //
///////////////////////////////////////////////////////

const factFile = `
file "main/main.go"
package "android/soong/a"
file "a/a.go"
file "a/b.go"
package "android/soong/b"
dep "android/soong/a"
file "b/a.go"
`

// Equivalent to factFile, but missing an unnecessary file. Should require a
// rebuild.
const factFile2 = `// long comment
file "main/main.go"
package "android/soong/a"
file "a/a.go"
package "android/soong/b"
dep "android/soong/a"
file "b/a.go"
`

const go_main_main = `
package main
import (
	"fmt"
	"android/soong/a"
	"android/soong/b"
)
func main() {
	fmt.Println(a.Stdout, b.Stdout)
}
`

const go_a_a = `
package a
import "os"
var Stdout = os.Stdout
`

const go_a_b = `
package a
`

const go_b_a = `
package b
import "android/soong/a"
var Stdout = a.Stdout
`

type T interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func setupDir(t T, test func(dir string, pkg *GoPackage)) {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %#v", err)
	}
	defer os.RemoveAll(dir)

	writeFile := func(name, contents string) {
		if err := ioutil.WriteFile(filepath.Join(dir, name), []byte(contents), 0666); err != nil {
			t.Fatalf("Error writing %q: %#v", name, err)
		}
	}
	mkdir := func(name string) {
		if err := os.Mkdir(filepath.Join(dir, name), 0777); err != nil {
			t.Fatalf("Error creating %q directory: %#v", name, err)
		}
	}
	mkdir("main")
	mkdir("a")
	mkdir("b")
	writeFile("test.fact", factFile)
	writeFile("main/main.go", go_main_main)
	writeFile("a/a.go", go_a_a)
	writeFile("a/b.go", go_a_b)
	writeFile("b/a.go", go_b_a)

	pkg, err := readFile(filepath.Join(dir, "test.fact"))
	if err != nil {
		t.Fatalf("Got error while reading test.fact: %v", err)
	}

	test(dir, pkg)
}
