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
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/google/blueprint/proptools"
)

type RewriteNames []RewriteName
type RewriteName struct {
	regexp *regexp.Regexp
	repl   string
}

func (r *RewriteNames) String() string {
	return ""
}

func (r *RewriteNames) Set(v string) error {
	split := strings.SplitN(v, "=", 2)
	if len(split) != 2 {
		return fmt.Errorf("Must be in the form of <regex>=<replace>")
	}
	regex, err := regexp.Compile(split[0])
	if err != nil {
		return nil
	}
	*r = append(*r, RewriteName{
		regexp: regex,
		repl:   split[1],
	})
	return nil
}

func (r *RewriteNames) Rewrite(name string) string {
	for _, r := range *r {
		if r.regexp.MatchString(name) {
			return r.regexp.ReplaceAllString(name, r.repl)
		}
	}
	return name
}

var rewriteNames = RewriteNames{}

type Dependency struct {
	XMLName xml.Name `xml:"dependency"`

	Name string `xml:"-"`

	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Type       string `xml:"type"`
}

type Pom struct {
	XMLName xml.Name `xml:"http://maven.apache.org/POM/4.0.0 project"`

	Name          string `xml:"-"`
	SrcFile       string `xml:"-"`
	SrcFileSuffix string `xml:"-"`

	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
	Packaging  string `xml:"packaging"`

	Dependencies []Dependency `xml:"dependencies>dependency"`
}

var mkTemplate = template.Must(template.New("mk").Parse(`
include $(CLEAR_VARS)
LOCAL_MODULE := {{.Name}}
LOCAL_MODULE_CLASS := JAVA_LIBRARIES
LOCAL_UNINSTALLABLE_MODULE := true
LOCAL_SRC_FILES := {{.SrcFile}}{{.SrcFileSuffix}}
LOCAL_BUILT_MODULE_STEM := javalib.jar
LOCAL_MODULE_SUFFIX := {{.SrcFileSuffix}}
LOCAL_USE_AAPT2 := true
LOCAL_STATIC_JAVA_LIBRARIES := \
{{range .Dependencies}}  {{.Name}} \
{{end}}
include $(BUILD_PREBUILT)
`))

func convert(filename string, out io.Writer) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var pom Pom
	err = xml.Unmarshal(data, &pom)
	if err != nil {
		return err
	}

	pom.Name = rewriteNames.Rewrite(pom.ArtifactId)

	for i, d := range pom.Dependencies {
		pom.Dependencies[i].Name = rewriteNames.Rewrite(d.ArtifactId)
	}

	pom.SrcFileSuffix = ".jar"
	if pom.Packaging == "aar" {
		pom.SrcFileSuffix = ".aar"
	}

	pom.SrcFile = strings.TrimSuffix(filename, ".pom")

	return mkTemplate.Execute(out, pom)
}

func main() {
	flag.Var(&rewriteNames, "rewrite", "Regex(es) to rewrite artifact names")
	flag.Parse()

	fmt.Println("# Automatically generated with:")
	fmt.Println("#", strings.Join(proptools.ShellEscape(os.Args), " "))
	fmt.Println("LOCAL_PATH := $(call my-dir)")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get cwd:", err)
		os.Exit(1)
	}

	var filenames []string
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := info.Name()
		if info.IsDir() {
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(name, ".") {
			return nil
		}

		if !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".pom") {
			path, err = filepath.Rel(cwd, path)
			if err != nil {
				return err
			}
			filenames = append(filenames, path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error walking files:", err)
		os.Exit(1)
	}

	sort.Strings(filenames)

	for _, filename := range filenames {
		err := convert(filename, os.Stdout)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error converting", filename, err)
			os.Exit(1)
		}
	}
}
