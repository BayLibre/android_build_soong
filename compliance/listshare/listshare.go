// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"android/soong/compliance/metadata"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s file.meta_lic {file.meta_lic...}

List the projects that must be shared for OS license compliance when releasing
targets whose license metadata is represented by the argument filenames.

Options:
`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	// FIXME: remove debug output
	fmt.Fprintf(os.Stderr, "Environment:\n")
	for _, ev := range os.Environ() {
		fmt.Fprintf(os.Stderr, "%s\n", ev)
	}
	fmt.Fprintf(os.Stderr, "\n")

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	files := append([]string{}, flag.Args()...)
	licenseMetadata, err := metadata.ReadLicenseMetadata(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read license metadata file(s) %q: %v\n", files, err)
		os.Exit(1)
	}
	if licenseMetadata == nil {
		fmt.Fprintf(os.Stderr, "No licenses\n")
		os.Exit(1)
	}
	mustShare := licenseMetadata.WalkDepsForCondition("reciprocal")
	mustShare.Add(licenseMetadata.WalkRestricted())
	for _, p := range mustShare.Projects() {
		fmt.Fprintln(os.Stdout, p)
	}
	os.Exit(0)
}
