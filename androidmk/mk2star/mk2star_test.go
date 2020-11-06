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
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
_vars["PRODUCT_NAME"] = "Pixel 3"
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
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
_vars["PRODUCT_PACKAGES"] = [
    "package1",
    "package2",
]
_vars["PRODUCT_COPY_FILES"] = ["file2:target"]
_vars["PRODUCT_PACKAGES"] += ["package3"]
pixel4 = rblf.prodconf(
    "pixel4",
    [],
    **_vars,
)
`,
	},
	/*
	   	{
	   		desc: "Bad func",
	   		mkname: "product.mk",
	   		in: `
	   PRODUCT_NAME := $(call foo, bar)
	   `,
	   		expected: `
	   `,
	   	},
	*/
	{
		desc:   "config",
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
		desc:   "conditional subconfig",
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
		desc:   "Same submodule name",
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
		desc:   "other call",
		mkname: "product.mk",
		in: `
$(call macro, part.mk)
`,
		expected: `# MK2STAR TRANSLATION ERROR: cannot convert calling 'macro'
# $(call macro, part.mk)
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
		desc:   "Simple if",
		mkname: "product.mk",
		in: `
ifdef  PRODUCT_NAME
  PRODUCT_NAME = dummy
else
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if rblf.is_defined("PRODUCT_NAME"):
    _vars["PRODUCT_NAME"] = "dummy"
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
$(info this is the info)
$(error this is the error)
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
rblf.mkwarning("product.mk", "this is the warning")
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
  PRODUCT_NAME=dummy
else
  PRODUCT_NAME=defined
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if not rblf.is_defined("PRODUCT_NAME"):
    _vars["PRODUCT_NAME"] = "dummy"
  else:
    _vars["PRODUCT_NAME"] = "defined"
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
	  PRODUCT_NAME = dummy
	else ifndef PRODUCT_PACKAGES
	endif
	`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if rblf.is_defined("PRODUCT_NAME"):
    _vars["PRODUCT_NAME"] = "dummy"
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
	/*
	       {
	   		desc:     "Call in RHS",
	   		mkname:   "product.mk",
	   		in:       `
	   PRODUCT_NAME=$(call foo, arg1,arg2)`,
	   		expected: `
	   `,
	   	},
	*/
	{
		desc:   "Ifeq / ifneq",
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
  if rblf.soong_var("TARGET_PRODUCT") == "aosp_arm":
    _vars["PRODUCT_MODEL"] = "pix2"
  else:
    _vars["PRODUCT_MODEL"] = "pix21"
_maybe()

def _maybe1():
  if rblf.soong_var("TARGET_PRODUCT") != "aosp_x86":
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
		desc:   "ifeq filter",
		mkname: "product.mk",
		in: `
ifeq (,$(filter userdebug eng, $(TARGET_BUILD_VARIANT)))
endif
ifneq (,$(filter userdebug,$(TARGET_BUILD_VARIANT))
endif
ifeq (,$(filter-out userdebug eng,$(TARGET_BUILD_VARIANT))
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
  if rblf.soong_var("TARGET_PRODUCT") == "aosp":
    pass
  elif rblf.soong_var("TARGET_PRODUCT") != "":
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
  elif rblf.soong_var("TARGET_PRODUCT") != "":
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
		desc:   "filter $(VAR), values",
		mkname: "product.mk",
		in: `
ifeq (,$(filter $(TARGET_PRODUCT), yukawa_gms yukawa_sei510_gms)
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
_vars = dict()
def _maybe():
  if rblf.soong_var("TARGET_PRODUCT") in ["yukawa_gms", "yukawa_sei510_gms"]:
    pass
_maybe()

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
}

var oneCase string // = "Sub config"

func TestGood(t *testing.T) {
	for _, v := range known_variables {
		KnownVariables.NewVariable(v.name, v.class, v.flavor)
	}

	for _, test := range testCases {
		if oneCase != "" && test.desc != oneCase {
			continue
		}
		ss, err := Convert(test.mkname, bytes.NewBufferString(test.in), false)
		if err != nil {
			t.Error(err)
			continue
		}
		got := ss.String()
		if got != test.expected {
			t.Errorf("%q failed\nExpected:\n%s\nActual:\n%s\n", test.desc,
				strings.ReplaceAll(test.expected, "\n", "␤\n"),
				strings.ReplaceAll(got, "\n", "␤\n"))
		}
	}
}
