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

package java

import (
	"android/soong/android"
	"fmt"
	"io"
	"strings"
)

func (library *Library) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Class:      "JAVA_LIBRARIES",
		OutputFile: android.OptionalPathForPath(library.outputFile),
		Extra: []android.AndroidMkExtraFunc{
			func(w io.Writer, outputFile android.Path) {
				fmt.Fprintln(w, "LOCAL_MODULE_SUFFIX := .jar")
			},
		},
	}
}

func (prebuilt *Import) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Class:      "JAVA_LIBRARIES",
		OutputFile: android.OptionalPathForPath(prebuilt.combinedClasspathFile),
		Extra: []android.AndroidMkExtraFunc{
			func(w io.Writer, outputFile android.Path) {
				fmt.Fprintln(w, "LOCAL_MODULE_SUFFIX := .jar")
			},
		},
	}
}

func (binary *Binary) AndroidMk() (ret android.AndroidMkData, err error) {
	ret = binary.Library.AndroidMk()
	if err != nil {
		return ret, err
	}

	ret.SubName = ".jar"

	ret.Custom = func(w io.Writer, name, prefix, moduleDir string, data android.AndroidMkData) {
		android.WriteAndroidMkData(w, data)

		fmt.Fprintln(w, "library_built_module := $(LOCAL_BUILT_MODULE)")
		fmt.Fprintln(w, "include $(CLEAR_VARS)")
		fmt.Fprintln(w, "LOCAL_MODULE := "+name)
		fmt.Fprintln(w, "LOCAL_MODULE_CLASS := EXECUTABLES")
		if strings.Contains(prefix, "HOST_") {
			fmt.Fprintln(w, "LOCAL_IS_HOST_MODULE := true")
		}
		fmt.Fprintln(w, "include $(BUILD_SYSTEM)/base_rules.mk")
		fmt.Fprintln(w, "$(LOCAL_BUILT_MODULE): $(library_built_module)")
		fmt.Fprintln(w, "$(LOCAL_BUILT_MODULE): "+binary.wrapperFile.String())
		fmt.Fprintln(w, "\t$(copy-file-to-new-target)")
		fmt.Fprintln(w, "\t$(chmod 755 $@)")
	}

	return
}
