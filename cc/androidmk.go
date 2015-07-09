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

package cc

import (
	"io"
	"strings"

	"github.com/google/blueprint/pathtools"

	"android/soong/common"
)

func (c *CCLibrary) AndroidMk() (ret common.AndroidMkData) {
	if c.static() {
		ret.Class = "STATIC_LIBRARIES"
	} else {
		ret.Class = "SHARED_LIBRARIES"
	}
	ret.OutputFile = c.outputFile()
	ret.Extra = func(name, prefix string) (ret []string) {
		if len(c.Properties.Export_include_dirs) > 0 {
			ret = append(ret, "LOCAL_EXPORT_C_INCLUDE_DIRS := "+strings.Join(pathtools.PrefixPaths(c.Properties.Export_include_dirs, "$(LOCAL_SRC_PATH)"), " "))
		}
		ret = append(ret, "LOCAL_SYSTEM_SHARED_LIBRARIES := "+strings.Join(c.systemLibs, " "))

		suffix := sharedLibraryExtension
		if c.static() {
			suffix = staticLibraryExtension
		}
		ret = append(ret, "LOCAL_MODULE_SUFFIX := "+suffix+"\n")

		return
	}
	return
}

func (c *ccObject) AndroidMk() (ret common.AndroidMkData) {
	ret.OutputFile = c.outputFile()
	ret.Custom = func(w io.Writer, name, prefix string) {
		out := c.outputFile()

		io.WriteString(w, "\n$(SOONG_OUT_DIR)/"+out+": build-soong ;\n")
		io.WriteString(w, "$("+prefix+"TARGET_OUT_INTERMEDIATE_LIBRARIES)/"+name+objectExtension+": $(SOONG_OUT_DIR)/"+out+" | $(ACP)\n")
		io.WriteString(w, "\t$(copy-file-to-target)\n")
	}
	return
}

func (c *CCBinary) AndroidMk() (ret common.AndroidMkData) {
	ret.Class = "EXECUTABLES"
	ret.OutputFile = c.outputFile()
	return
}
