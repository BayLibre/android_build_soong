package mk2star

import (
	"bytes"
	"strings"
	"testing"
)

var testCases = []struct {
	desc     string
	mkname   string
	in       string
	expected string
}{
	{
		desc:   "Comment",
		mkname: "product.mk",
		in: `
# Comment
# FOO= a\
     b
`,
		expected: `# Comment
# FOO= a
#     b
load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "Name conversion",
		mkname: "path/bar-baz.mk",
		in: `
# Comment
`,
		expected: `# Comment
load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
bar_baz = rblf.prodconf(
    "bar_baz",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "Item variable",
		mkname: "pixel3.mk",
		in: `
PRODUCT_NAME := Pixel 3
PRODUCT_MODEL :=
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
_vars["PRODUCT_NAME"] = "Pixel 3"
_vars["PRODUCT_MODEL"] = ""
pixel3 = rblf.prodconf(
    "pixel3",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "List variable",
		mkname: "pixel4.mk",
		in: `
PRODUCT_PACKAGES = package1  package2
PRODUCT_COPY_FILES += file2:target
PRODUCT_PACKAGES += package3
PRODUCT_COPY_FILES =
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
_vars["PRODUCT_PACKAGES"] = [
    "package1",
    "package2",
]
_vars["PRODUCT_COPY_FILES"] = ["file2:target"]
_vars["PRODUCT_PACKAGES"] += ["package3"]
_vars["PRODUCT_COPY_FILES"] = []
pixel4 = rblf.prodconf(
    "pixel4",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "Unknown function",
		mkname: "product.mk",
		in: `
PRODUCT_NAME := $(call foo, bar)
`,
		expected: `# MK2STAR TRANSLATION ERROR: cannot handle invoking foo
# PRODUCT_NAME := $(call foo, bar)
load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
product = rblf.prodconf(
    "product",
    [],
    **_vars,
)

rblf.warning("product.mk")
`,
	},
	{
		desc:   "Inherit configuration always",
		mkname: "product.mk",
		in: `
$(call inherit-product, part.mk)
$(call inherit-product, $(LOCAL_PATH)/part.mk)
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
load(":part.star", _part = "part")
load(":part.star", _part1 = "part")
product = rblf.prodconf(
    "product",
    [_part, _part1],
    **_vars,
)
`,
	},
	{
		desc:   "Inherit configuration if it exists",
		mkname: "product.mk",
		in: `
$(call inherit-product-if-exists, $(SRC_TARGET_DIR)/part.mk)
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
load(":part.star|part", _part = "part")
product = rblf.prodconf(
    "product",
    [_part],
    **_vars,
)
`,
	},
	{
		desc:   "Inherit configuration with include",
		mkname: "product.mk",
		in: `
include part.mk)
-include $(LOCAL_PATH)/part.mk)
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
load(":part.star", _part = "part")
load(":part.star|part", _part1 = "part")
product = rblf.prodconf(
    "product",
    [_part, _part1],
    **_vars,
)
`,
	},
	{
		desc:   "Synonymous inherited configurations",
		mkname: "path/product.mk",
		in: `
$(call inherit-product, foo/font.mk)
$(call inherit-product, bar/font.mk)
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
load("//foo:font.star", _font = "font")
load("//bar:font.star", _font1 = "font")
product = rblf.prodconf(
    "product",
    [_font, _font1],
    **_vars,
)
`,
	},
	{
		desc:   "Directive define",
		mkname: "product.mk",
		in: `
define some-macro
    $(info foo)
endef
`,
		expected: `# MK2STAR TRANSLATION ERROR: define is not supported: some-macro
# define  some-macro
#     $(info foo)
# endef
load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
product = rblf.prodconf(
    "product",
    [],
    **_vars,
)

rblf.warning("product.mk")
`,
	},
	{
		desc:   "Ifdef",
		mkname: "product.mk",
		in: `
ifdef  PRODUCT_NAME
  PRODUCT_NAME = gizmo
else
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if rblf.is_defined("PRODUCT_NAME"):
    _vars["PRODUCT_NAME"] = "gizmo"
  else:
    pass
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "Simple functions",
		mkname: "product.mk",
		in: `
$(warning this is the warning)
$(warning)
$(info this is the info)
$(error this is the error)
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
rblf.mkwarning("product.mk", "this is the warning")
rblf.mkwarning("product.mk", "")
rblf.mkinfo("product.mk", "this is the info")
rblf.mkerror("product.mk", "this is the error")
product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "Empty if",
		mkname: "product.mk",
		in: `
ifdef PRODUCT_NAME
# Comment
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if rblf.is_defined("PRODUCT_NAME"):
    # Comment
    pass
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "if/else/endif",
		mkname: "product.mk",
		in: `
ifndef PRODUCT_NAME
  PRODUCT_NAME=gizmo1
else
  PRODUCT_NAME=gizmo2
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if not rblf.is_defined("PRODUCT_NAME"):
    _vars["PRODUCT_NAME"] = "gizmo1"
  else:
    _vars["PRODUCT_NAME"] = "gizmo2"
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "else if",
		mkname: "product.mk",
		in: `
	ifdef  PRODUCT_NAME
	  PRODUCT_NAME = gizmo
	else ifndef PRODUCT_PACKAGES
	endif
	`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if rblf.is_defined("PRODUCT_NAME"):
    _vars["PRODUCT_NAME"] = "gizmo"
  elif not rblf.is_defined("PRODUCT_PACKAGES"):
    pass
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "ifeq / ifneq",
		mkname: "product.mk",
		in: `
ifeq (aosp_arm, $(TARGET_PRODUCT))
  PRODUCT_MODEL = pix2
else
  PRODUCT_MODEL = pix21
endif
ifneq (aosp_x86, $(TARGET_PRODUCT))
  PRODUCT_MODEL = pix3
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if "aosp_arm" == rblf.soong_var("TARGET_PRODUCT"):
    _vars["PRODUCT_MODEL"] = "pix2"
  else:
    _vars["PRODUCT_MODEL"] = "pix21"
_maybe()

def _maybe1():
  if "aosp_x86" != rblf.soong_var("TARGET_PRODUCT"):
    _vars["PRODUCT_MODEL"] = "pix3"
_maybe1()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "Check filter result",
		mkname: "product.mk",
		in: `
ifeq (,$(filter userdebug eng, $(TARGET_BUILD_VARIANT)))
endif
ifneq (,$(filter userdebug,$(TARGET_BUILD_VARIANT))
endif
ifeq (,$(filter-out userdebug eng,$(TARGET_BUILD_VARIANT))
endif
ifeq ($(TARGET_BUILD_VARIANT), $(filter $(TARGET_BUILD_VARIANT), userdebug eng))
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if rblf.soong_var("TARGET_BUILD_VARIANT") not in ["userdebug", "eng"]:
    pass
_maybe()

def _maybe1():
  if rblf.soong_var("TARGET_BUILD_VARIANT") in ["userdebug"]:
    pass
_maybe1()

def _maybe2():
  if rblf.soong_var("TARGET_BUILD_VARIANT") in ["userdebug", "eng"]:
    pass
_maybe2()

def _maybe3():
  if rblf.soong_var("TARGET_BUILD_VARIANT") in ["userdebug", "eng"]:
    pass
_maybe3()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "filter $(VAR), values",
		mkname: "product.mk",
		in: `
ifeq (,$(filter $(TARGET_PRODUCT), yukawa_gms yukawa_sei510_gms)
  ifneq (,$(filter $(TARGET_PRODUCT), yukawa_gms)
  endif
endif

`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if rblf.soong_var("TARGET_PRODUCT") not in ["yukawa_gms", "yukawa_sei510_gms"]:
    if rblf.soong_var("TARGET_PRODUCT") in ["yukawa_gms"]:
      pass
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "Bad var in filter",
		mkname: "product.mk",
		in: `
ifneq ($(filter sdk win_sdk sdk_addon,$(MAKECMDGOALS)),)
endif`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  # MK2STAR ERROR converting:
  #   ifneq ($(filter sdk win_sdk sdk_addon,$(MAKECMDGOALS)),)
  # unknown variable MAKECMDGOALS
  if False:
    pass
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)

rblf.warning("product.mk")
`,
	},
	{
		desc:   "ifeq",
		mkname: "product.mk",
		in: `
ifeq (aosp, $(TARGET_PRODUCT))
else ifneq (, $(TARGET_PRODUCT))
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if "aosp" == rblf.soong_var("TARGET_PRODUCT"):
    pass
  elif "" != rblf.soong_var("TARGET_PRODUCT"):
    pass
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "Nested if",
		mkname: "product.mk",
		in: `
ifdef PRODUCT_NAME
  PRODUCT_PACKAGES+= pack-if0
  ifdef PRODUCT_MODEL
    PRODUCT_PACKAGES+= pack-if-if
  else ifdef PRODUCT_NAME
    PRODUCT_PACKAGES+= pack-if-elif
  else
    PRODUCT_PACKAGES+= pack-if-else
  endif
  PRODUCT_PACKAGES+= pack-if
else ifneq (,$(TARGET_PRODUCT))
  PRODUCT_PACKAGES+= pack-elif
else
  PRODUCT_PACKAGES+= pack-else
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if rblf.is_defined("PRODUCT_NAME"):
    _vars["PRODUCT_PACKAGES"] = ["pack-if0"]
    if rblf.is_defined("PRODUCT_MODEL"):
      _vars["PRODUCT_PACKAGES"] += ["pack-if-if"]
    elif rblf.is_defined("PRODUCT_NAME"):
      _vars["PRODUCT_PACKAGES"] += ["pack-if-elif"]
    else:
      _vars["PRODUCT_PACKAGES"] += ["pack-if-else"]
    _vars["PRODUCT_PACKAGES"] += ["pack-if"]
  elif "" != rblf.soong_var("TARGET_PRODUCT"):
    _vars["PRODUCT_PACKAGES"] += ["pack-elif"]
  else:
    _vars["PRODUCT_PACKAGES"] += ["pack-else"]
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "Wildcard",
		mkname: "product.mk",
		in: `
ifeq (,$(wildcard foo.mk)
endif
ifneq (,$(wildcard foo*.mk)
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if not rblf.file_exists("foo.mk"):
    pass
_maybe()

def _maybe1():
  if rblf.file_wildcard_exists("foo*.mk"):
    pass
_maybe1()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "ifneq $(X),true",
		mkname: "product.mk",
		in: `
ifneq ($(VARIABLE),true)
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  # MK2STAR ERROR converting:
  #   ifneq ($(VARIABLE),true)
  # unknown variable VARIABLE
  if False:
    pass
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)

rblf.warning("product.mk")
`,
	},
	{
		desc:   "Const neq",
		mkname: "product.mk",
		in: `
ifneq (1,0)
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if "1" != "0":
    pass
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "is-board calls",
		mkname: "product.mk",
		in: `
ifeq ($(call is-board-platform-in-list,msm8998), true)
else ifneq ($(call is-board-platform,copper),true)
else ifneq ($(call is-vendor-board-platform,QCOM),true)
else ifeq ($(call is-product-in-list, $(PLATFORM_LIST)), true)
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if rblf.soong_var("TARGET_BOARD_PLATFORM") in ["msm8998"]:
    pass
  elif rblf.soong_var("TARGET_BOARD_PLATFORM") != "copper":
    pass
  elif rblf.soong_var("TARGET_BOARD_PLATFORM") not in rblf.soong_var("QCOM_BOARD_PLATFORMS"):
    pass
  elif rblf.soong_var("TARGET_PRODUCT") in rblf.soong_var("PLATFORM_LIST"):
    pass
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "findstring call",
		mkname: "product.mk",
		in: `
ifneq ($(findstring foo,$(PRODUCT_MODEL)),)
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if _vars["PRODUCT_MODEL"].find("foo") != -1:
    pass
_maybe()

product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "rhs call",
		mkname: "product.mk",
		in: `
PRODUCT_COPY_FILES += $(call add-to-product-copy-files-if-exists, path:distpath)
PRODUCT_COPY_FILES += $(call find-copy-subdir-files, *, fromdir, todir)`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
_vars["PRODUCT_COPY_FILES"] = rblf.copy_if_exists("path:distpath")
_vars["PRODUCT_COPY_FILES"] += rblf.find_and_copy("*", "fromdir", "todir")
product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "list with vars",
		mkname: "product.mk",
		in: `
PRODUCT_COPY_FILES += path1:$(TARGET_PRODUCT)/path1 path2:$(TARGET_PRODUCT)/path2 
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
_vars["PRODUCT_COPY_FILES"] = [
    "path1:%s/path1" % (rblf.soong_var("TARGET_PRODUCT")),
    "path2:%s/path2" % (rblf.soong_var("TARGET_PRODUCT")),
]
product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "misc calls",
		mkname: "product.mk",
		in: `
$(call enforce-product-packages-exist,)
$(call enforce-product-packages-exist, foo)
$(call require-artifacts-in-path, foo, bar)
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
rblf.enforce_product_packages_exist("")
rblf.enforce_product_packages_exist("foo")
rblf.require_artifacts_in_path("foo", "bar")
product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
	{
		desc:   "list with functions",
		mkname: "product.mk",
		in: `
PRODUCT_COPY_FILES := $(call find-copy-subdir-files,*.kl,from1,to1) \
 $(call find-copy-subdir-files,*.kc,from2,to2) \
 foo bar
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
_vars["PRODUCT_COPY_FILES"] = rblf.find_and_copy("*.kl", "from1", "to1") +
    rblf.find_and_copy("*.kc", "from2", "to2") +
    [
        "foo",
        "bar",
    ]
product = rblf.prodconf(
    "product",
    [],
    **_vars,
)
`,
	},
}

var known_variables = []struct {
	name   string
	class  string
	flavor string
}{
	{"PRODUCT_NAME", "product", "item"},
	{"PRODUCT_MODEL", "product", "item"},
	{"PRODUCT_PACKAGES", "product", "list"},
	{"PRODUCT_COPY_FILES", "product", "list"},
	{"PRODUCT_IS_64BIT", "product", "item"},
	{"TARGET_PRODUCT", "soong", "str"},
	{"TARGET_BUILD_VARIANT", "soong", "str"},
	{"TARGET_BOARD_PLATFORM", "soong", "str"},
	{"QCOM_BOARD_PLATFORMS", "soong", "str"},
	{"PLATFORM_LIST", "soong", "list"}, // TODO(asmundak): make it local instead of soong
}

func TestGood(t *testing.T) {
	for _, v := range known_variables {
		KnownVariables.NewVariable(v.name, v.class, v.flavor)
	}
	for _, test := range testCases {
		t.Run(test.desc,
			func(t *testing.T) {
				ss, err := Convert(test.mkname, bytes.NewBufferString(test.in), false, ".")
				if err != nil {
					t.Error(err)
					return
				}
				got := ss.String()
				if got != test.expected {
					t.Errorf("%q failed\nExpected:\n%s\nActual:\n%s\n", test.desc,
						strings.ReplaceAll(test.expected, "\n", "␤\n"),
						strings.ReplaceAll(got, "\n", "␤\n"))
				}
			})
	}
}
