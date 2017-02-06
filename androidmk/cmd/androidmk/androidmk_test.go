// Copyright 2016 Google Inc. All rights reserved.
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
	"bytes"
	"fmt"
	"os"
	"testing"

	bpparser "github.com/google/blueprint/parser"
	"math"
)

var testCases = []struct {
	desc     string
	in       string
	expected string
}{
	{
		desc: "basic cc_library_shared with comments",
		in: `
#
# Copyright
#

# Module Comment
include $(CLEAR_VARS)
# Name Comment
LOCAL_MODULE := test
# Source comment
LOCAL_SRC_FILES_EXCLUDE := a.c
# Second source comment
LOCAL_SRC_FILES_EXCLUDE += b.c
include $(BUILD_SHARED_LIBRARY)`,
		expected: `
//
// Copyright
//

// Module Comment
cc_library_shared {
    // Name Comment
    name: "test",
    // Source comment
    exclude_srcs: ["a.c"] + /* Second source comment */ ["b.c"],
}
`,
	},
	{
		desc: "split local/global include_dirs (1)",
		in: `
include $(CLEAR_VARS)
LOCAL_C_INCLUDES := $(LOCAL_PATH)
include $(BUILD_SHARED_LIBRARY)`,
		expected: `cc_library_shared {
    local_include_dirs: ["."],
}
`,
	},
	{
		desc: "split local/global include_dirs (2)",
		in: `
include $(CLEAR_VARS)
LOCAL_C_INCLUDES := $(LOCAL_PATH)/include
include $(BUILD_SHARED_LIBRARY)`,
		expected: `cc_library_shared {
    local_include_dirs: ["include"],
}
`,
	},
	{
		desc: "split local/global include_dirs (3)",
		in: `
include $(CLEAR_VARS)
LOCAL_C_INCLUDES := system/core/include
include $(BUILD_SHARED_LIBRARY)`,
		expected: `cc_library_shared {
    include_dirs: ["system/core/include"],
}
`,
	},
	{
		desc: "split local/global include_dirs (4)",
		in: `
input := testing/include
include $(CLEAR_VARS)
# Comment 1
LOCAL_C_INCLUDES := $(LOCAL_PATH) $(LOCAL_PATH)/include system/core/include $(input)
# Comment 2
LOCAL_C_INCLUDES += $(TOP)/system/core/include $(LOCAL_PATH)/test/include
# Comment 3
include $(BUILD_SHARED_LIBRARY)`,
		expected: `input = ["testing/include"]
cc_library_shared {
    // Comment 1
    include_dirs: ["system/core/include"] + input + /* Comment 2 */ ["system/core/include"],
    local_include_dirs: ["."] + ["include"] + ["test/include"],
    // Comment 3
}
`,
	},
	{
		desc: "comments inside a list",
		in: `
include $(CLEAR_VARS)

LOCAL_SRC_FILES := \
        base.cpp \
        misc.cpp

#V 3.0 source
LOCAL_SRC_FILES += \
        v3/base.cpp \
        v3/extra.cpp

#V 1.0 source
LOCAL_SRC_FILES += \
        v1/base.cpp \
        v1/extra.cpp

include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {

    srcs: [
        "base.cpp",
        "misc.cpp",
    ] + /*V 3.0 source */ [
        "v3/base.cpp",
        "v3/extra.cpp",
    ] + /*V 1.0 source */ [
        "v1/base.cpp",
        "v1/extra.cpp",
    ],

}
`,
	},
	{
		desc: "LOCAL_MODULE_STEM",
		in: `
include $(CLEAR_VARS)
LOCAL_MODULE := libtest
LOCAL_MODULE_STEM := $(LOCAL_MODULE).so
include $(BUILD_SHARED_LIBRARY)

include $(CLEAR_VARS)
LOCAL_MODULE := libtest2
LOCAL_MODULE_STEM := testing.so
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    name: "libtest",
    suffix: ".so",
}

cc_library_shared {
    name: "libtest2",
    stem: "testing.so",
}
`,
	},
	{
		desc: "LOCAL_MODULE_HOST_OS",
		in: `
include $(CLEAR_VARS)
LOCAL_MODULE := libtest
LOCAL_MODULE_HOST_OS := linux darwin windows
include $(BUILD_SHARED_LIBRARY)

include $(CLEAR_VARS)
LOCAL_MODULE := libtest2
LOCAL_MODULE_HOST_OS := linux
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    name: "libtest",
    target: {
        windows: {
            enabled: true,
        },
    },
}

cc_library_shared {
    name: "libtest2",
    target: {
        darwin: {
            enabled: false,
        },
    },
}
`,
	},
	{
		desc: "LOCAL_RTTI_VALUE",
		in: `
include $(CLEAR_VARS)
LOCAL_MODULE := libtest
LOCAL_RTTI_FLAG := # Empty flag
include $(BUILD_SHARED_LIBRARY)

include $(CLEAR_VARS)
LOCAL_MODULE := libtest2
LOCAL_RTTI_FLAG := -frtti
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    name: "libtest",
    rtti: false, // Empty flag
}

cc_library_shared {
    name: "libtest2",
    rtti: true,
}
`,
	},
	{
		desc: "LOCAL_ARM_MODE",
		in: `
include $(CLEAR_VARS)
LOCAL_ARM_MODE := arm
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    arch: {
        arm: {
            instruction_set: "arm",
        },
    },
}
`,
	},
	{
		desc: "*.logtags in LOCAL_SRC_FILES",
		in: `
include $(CLEAR_VARS)
LOCAL_SRC_FILES := events.logtags
LOCAL_SRC_FILES += a.c events2.logtags
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    logtags: ["events.logtags"] + ["events2.logtags"],
    srcs: ["a.c"],
}
`,
	},
	{
		desc: "LOCAL_LOGTAGS_FILES and *.logtags in LOCAL_SRC_FILES",
		in: `
include $(CLEAR_VARS)
LOCAL_LOGTAGS_FILES := events.logtags
LOCAL_SRC_FILES := events2.logtags
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    logtags: ["events.logtags"] + ["events2.logtags"],
}
`,
	},
	{
		desc: "_<OS> suffixes",
		in: `
include $(CLEAR_VARS)
LOCAL_SRC_FILES_darwin := darwin.c
LOCAL_SRC_FILES_linux := linux.c
LOCAL_SRC_FILES_windows := windows.c
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    target: {
        darwin: {
            srcs: ["darwin.c"],
        },
        linux: {
            srcs: ["linux.c"],
        },
        windows: {
            srcs: ["windows.c"],
        },
    },
}
`,
	},
	{
		desc: "LOCAL_SANITIZE := never",
		in: `
include $(CLEAR_VARS)
LOCAL_SANITIZE := never
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    sanitize: {
        never: true,
    },
}
`,
	},
	{
		desc: "LOCAL_SANITIZE unknown parameter",
		in: `
include $(CLEAR_VARS)
LOCAL_SANITIZE := integer asdf
LOCAL_SANITIZE_RECOVER := qwert
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `
// ANDROIDMK TRANSLATION ERROR at 3:1 : unknown sanitize argument: asdf
// integer asdf

cc_library_shared {
    sanitize: {
        integer: true,
        recover: ["qwert"],
    },
}
`,
	},
	{
		desc: "LOCAL_SANITIZE using variable",
		in: `
sanitize_var := never
include $(CLEAR_VARS)
LOCAL_SANITIZE := $(sanitize_var)
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `sanitize_var = ["never"]
cc_library_shared {
    sanitize: sanitize_var,
}
`,
	},
	{
		desc: "LOCAL_SANITIZE_RECOVER",
		in: `
include $(CLEAR_VARS)
LOCAL_SANITIZE_RECOVER := shift-exponent
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    sanitize: {
        recover: ["shift-exponent"],
    },
}
`,
	},
	{
		desc: "version_script in LOCAL_LDFLAGS",
		in: `
include $(CLEAR_VARS)
LOCAL_LDFLAGS := -Wl,--link-opt -Wl,--version-script,$(LOCAL_PATH)/exported32.map
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    ldflags: ["-Wl,--link-opt"],
    version_script: "exported32.map",
}
`,
	},
	{
		desc: "Handle TOP",
		in: `
include $(CLEAR_VARS)
LOCAL_C_INCLUDES := $(TOP)/system/core/include $(TOP)
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    include_dirs: [
        "system/core/include",
        ".",
    ],
}
`,
	},
	{
		desc: "Remove LOCAL_MODULE_TAGS optional",
		in: `
include $(CLEAR_VARS)
LOCAL_MODULE_TAGS := optional
include $(BUILD_SHARED_LIBRARY)
`,

		expected: `cc_library_shared {}
`,
	},
	{
		desc: "Keep LOCAL_MODULE_TAGS non-optional",
		in: `
include $(CLEAR_VARS)
LOCAL_MODULE_TAGS := debug
include $(BUILD_SHARED_LIBRARY)
`,

		expected: `cc_library_shared {
    tags: ["debug"],
}
`,
	},
	{
		desc: "spaces between properties",
		in: `# Comment1

# Comment2
VAR := a

VAR += b

# Comment3
		`,
		expected: `// Comment1

// Comment2
VAR = ["a"]

VAR += ["b"]

// Comment3
`,
	},
	{
		desc: "comments above an unsupported rule",
		in: `LOCAL_PATH := $(call my-dir)

# Add all files to be generated from the source.prop templates to the SDK pre-requisites
ALL_SDK_FILES := development/sdk/source.prop_template

# Rule to convert a source.prop template into the desired source.property
$(HOST_OUT)/development/sys-img-$(TARGET_CPU_ABI)/%_source.properties : $(TOPDIR)development/sys-img/%_source.prop_template
	@echo Generate $@
# ===== SDK jar file of stubs =====
sdk_stub_name := android_stubs_current
`,
		expected: `
// Add all files to be generated from the source.prop templates to the SDK pre-requisites
ALL_SDK_FILES = ["development/sdk/source.prop_template"]

// Rule to convert a source.prop template into the desired source.property

// ANDROIDMK TRANSLATION ERROR at 7:1 : unsupported line
// rule:       $(HOST_OUT)/development/sys-img-$(TARGET_CPU_ABI)/%_source.properties : $(TOPDIR)development/sys-img/%_source.prop_template
// @echo Generate $@
//

// ===== SDK jar file of stubs =====
sdk_stub_name = ["android_stubs_current"]
`,
	},
	{
		desc: `comments above an unsupported definition`,
		in: `# $(1): the Java library name
define _package_sdk_library
$(eval _psm_build_module := $(TARGET_OUT_COMMON_INTERMEDIATES)/JAVA_LIBRARIES/$(1)_intermediates/javalib.jar)
endef
`,
		expected: `// $(1): the Java library name

// ANDROIDMK TRANSLATION ERROR at 2:1 : unsupported directive
// define  _package_sdk_library
// $(eval _psm_build_module := $(TARGET_OUT_COMMON_INTERMEDIATES)/JAVA_LIBRARIES/$(1)_intermediates/javalib.jar)
// endef
`,
	},
	{
		desc: `comment above an unsupported call`,
		in: `# Build and store the android_test.jar.
$(call dist-for-goals,sdk win_sdk,$(full_target):android_test.jar)
`,
		expected: `// Build and store the android_test.jar.

// ANDROIDMK TRANSLATION ERROR at 2:1 : unsupported line
// $(call dist-for-goals,sdk win_sdk,$(full_target):android_test.jar)
`,
	},
	{
		desc: `copy comments for a property that gets split (LOCAL_C_INCLUDES)`,
		in: `LOCAL_C_INCLUDES := $(LOCAL_PATH) $(GLOBAL_PATH) # copy me to each line
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    include_dirs: GLOBAL_PATH, // copy me to each line
    local_include_dirs: ["."], // copy me to each line
}
`,
	},
	{
		desc: `multiline assignment with comments`,
		in: `LOCAL_MODULE_TAGS := tag \
					  you \
					  are \
					  it # I was here
include $(BUILD_SHARED_LIBRARY)
`,
		expected: `cc_library_shared {
    tags: [
        "tag",
        "you",
        "are",
        "it",
    ], // I was here
}
`,
	},
	{
		desc: `unrecognized rule with misindented comments`,
		in: `test-art-target-sync: $(TEST_ART_TARGET_SYNC_DEPS)
	$(TEST_ART_ADB_ROOT_AND_REMOUNT)
	adb wait-for-device push $(ANDROID_PRODUCT_OUT)/system $(ART_TEST_ANDROID_ROOT)
# Push the contents of the data dir into /data on the device
	adb push $(ANDROID_PRODUCT_OUT)/data /
# Ending comment
`,
		expected: `
// ANDROIDMK TRANSLATION ERROR at 1:1 : unsupported line
// rule:       test-art-target-sync: $(TEST_ART_TARGET_SYNC_DEPS)
// $(TEST_ART_ADB_ROOT_AND_REMOUNT)
// adb wait-for-device push $(ANDROID_PRODUCT_OUT)/system $(ART_TEST_ANDROID_ROOT)
// adb push $(ANDROID_PRODUCT_OUT)/data /
//

// Push the contents of the data dir into /data on the device
// Ending comment
`,
	},
	{
		desc: `multiline assignment missing a backslash`,
		in: `#opening comment
include $(CLEAR_VARS)
LOCAL_C_INCLUDES := sample \
                    common \
                    lastSymbolInList
                    oopsIAmAnInvalidSymbol
include $(BUILD_SHARED_LIBRARY)
#trailing comment
`,
		expected: `//opening comment
cc_library_shared {
    include_dirs: [
        "sample",
        "common",
        "lastSymbolInList",
    ],
    // ANDROID MK PARSE ERROR at 6:43: expected directive, rule, or assignment after ident 'oopsIAmAnInvalidSymbol'
}

//trailing comment
`,
	},
	{
		desc: `error line ending in a backslash`,
		in: `include \
update_verifier/Android.mk \
`,
		expected: `
// ANDROIDMK TRANSLATION ERROR at 1:1 : unsupported include
// include  update_verifier/Android.mk
`,
	},
	{
		desc: `module with newlines`,
		in: `include $(CLEAR_VARS)
LOCAL_MODULE := linker-unit-tests

LOCAL_CFLAGS += -g -Wall -Wextra -Wunused -Werror
LOCAL_C_INCLUDES := $(LOCAL_PATH)/../../libc/

LOCAL_SRC_FILES := \
  linker_block_allocator_test.cpp

include $(BUILD_NATIVE_TEST)
`,
		expected: `cc_test {
    name: "linker-unit-tests",

    cflags: [
        "-g",
        "-Wall",
        "-Wextra",
        "-Wunused",
        "-Werror",
    ],
    local_include_dirs: ["../../libc/"],

    srcs: ["linker_block_allocator_test.cpp"],

}
`,
	},
	{
		desc: `nested unsupported ifs`,
		in: `
ifeq ($(I_AM_NOT_A_RECOGNIZED_PROPERTY),true)
ifeq ($(ME_NEITHER),linux)
MY_VARIABLE := hi
endif
endif
MY_OTHER_VARIABLE := bye
	`,
		expected: `
// ANDROIDMK TRANSLATION ERROR at 2:1 : unsupported conditional
// ifeq ($(I_AM_NOT_A_RECOGNIZED_PROPERTY),true)

// ANDROIDMK TRANSLATION ERROR at 3:1 : unsupported conditional
// ifeq ($(ME_NEITHER),linux)

MY_VARIABLE = ["hi"]

// ANDROIDMK TRANSLATION ERROR at 5:1 : endif from unsupported conditional: ($(ME_NEITHER),linux)
// endif

// ANDROIDMK TRANSLATION ERROR at 6:1 : endif from unsupported conditional: ($(I_AM_NOT_A_RECOGNIZED_PROPERTY),true)
// endif

MY_OTHER_VARIABLE = ["bye"]
`,
	},
	{
		desc: `when a variable moves, its comments follow`,
		in: `

include $(CLEAR_VARS)

#property that moves
MY_MOVING_PROPERTY := hi

LOCAL_C_INCLUDES := home

include $(BUILD_EXECUTABLE)
		`,
		expected: `
//property that moves
MY_MOVING_PROPERTY = ["hi"]
cc_binary {

    include_dirs: ["home"],

}
`,
	},
	//	// This test fails but we'd like to re-enable (and possibly slightly adjust) it once it will pass
	//	{
	//		desc: `unsupported 'if' next to properties that get moved`,
	//		in: `
	//
	//	include $(CLEAR_VARS)
	//
	//	MY_MOVING_PROPERTY := hi
	//
	//	ifeq ($(CANNOT_RECOGNIZE_ME),true)
	//	MY_MOVING_PROPERTY2 := bye
	//	LOCAL_SRC_FILES_EXCLUDE := sample
	//	endif
	//
	//	MY_MOVING_PROPERTY3 := bye
	//
	//	include $(BUILD_EXECUTABLE)
	//		`,
	//		expected: `
	//MY_MOVING_PROPERTY = ["hi"]
	//
	//MY_MOVING_PROPERTY2 = ["bye"]
	//
	//MY_MOVING_PROPERTY3 = ["bye"]
	//cc_binary {
	//
	//    // ANDROIDMK TRANSLATION ERROR: unsupported conditional
	//    // ifeq ($(CANNOT_RECOGNIZE_ME),true)
	//
	//    exclude_srcs: ["sample"],
	//
	//    // ANDROIDMK TRANSLATION ERROR: endif from unsupported conditional: ($(CANNOT_RECOGNIZE_ME),true)
	//    // endif
	//
	//}
	//		`,
	//	},
}

func reformatBlueprint(input string) string {
	parse, errs := bpparser.Parse("<testcase>", bytes.NewBufferString(input), bpparser.NewScope(nil))
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		panic(fmt.Sprintf("%d parse error while trying to parse expected Blueprint output: %s", len(errs), input))
	}

	res := bpparser.PrintTree(parse.SyntaxTree)

	return string(res)
}

func TestEndToEnd(t *testing.T) {
	for i, test := range testCases {
		expected := test.expected

		got, _, mkParse, bpTree := convertFileImpl(fmt.Sprintf("<androidmk_test testcase %d>", i), bytes.NewBufferString(test.in))
		// We could put the expected errors into the test and check those too, but we haven't done that yet because any expected errors will show up in the output

		if got != expected {
			// Do a diff to clarify what is wrong with the output
			var diff = fmt.Sprintf("got.len() = %v whereas expected.len() = %v", len(got), len(expected))
			var line = 0
			var column = 0
			for j := 0; j < int(math.Min(float64(len(got)), float64(len(expected)))); j++ {
				if got[j] != expected[j] {
					diff = fmt.Sprintf("got %#v instead of %#v (at %v, %v)\nMatching portions of strings before mismatch: '%v'",
						string(got[j]), string(expected[j]), line, column, got[:j])
					break
				}
				if got[j] == '\n' {
					line++
					column = 0
				} else {
					column++
				}
			}
			bpPrint := bpparser.VerbosePrint(bpTree)
			t.Errorf(
				"\nfailed test case %v:\n"+
					"input   :\n"+
					"%s\n"+
					"mkParse : %s\n"+
					"bpTree  : %s\n"+
					"expected:\n%s\n"+
					"got     :\n%s\n"+
					"diff    :%v",
				i, test.in, mkParse, bpPrint, expected, got, diff)
		}
	}
}
