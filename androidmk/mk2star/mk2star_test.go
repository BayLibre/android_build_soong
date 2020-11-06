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

def init(globals):
  cfg = propset()
  g = propset(globals)
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  return (cfg, g)
`,
	},
	{
		desc:   "Item variable",
		mkname: "pixel3.mk",
		in: `
PRODUCT_NAME := Pixel 3
PRODUCT_MODEL :=
local_var = foo
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")

def init(globals):
  cfg = propset()
  g = propset(globals)
  cfg.PRODUCT_NAME = "Pixel 3"
  cfg.PRODUCT_MODEL = ""
  _local_var = "foo"
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  cfg.PRODUCT_PACKAGES = [
      "package1",
      "package2",
  ]
  if not hasattr(cfg, "PRODUCT_COPY_FILES"):
    cfg.PRODUCT_COPY_FILES = []
  cfg.PRODUCT_COPY_FILES += ["file2:target"]
  cfg.PRODUCT_PACKAGES += ["package3"]
  cfg.PRODUCT_COPY_FILES = []
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  rc = (cfg, g)
  rblf.warning("product.mk", "partially successful conversion")
  return rc
`,
	},
	{
		desc:   "Inherit configuration always",
		mkname: "product.mk",
		in: `
ifdef PRODUCT_NAME
$(call inherit-product, part.mk)
else
$(call inherit-product, $(LOCAL_PATH)/part.mk)
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
load(":part.star", _part_init = "init")

def init(globals):
  cfg = propset()
  g = propset(globals)
  if hasattr(g, "PRODUCT_NAME"):
    rblf.merge(cfg, g, _part_init(g))
  else:
    rblf.merge(cfg, g, _part_init(g))
  return (cfg, g)
`,
	},
	{
		desc:   "Inherit configuration if it exists",
		mkname: "product.mk",
		in: `
$(call inherit-product-if-exists, $(SRC_TARGET_DIR)/part.mk)
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
load(":part.star|init", _part_init = "init")

def init(globals):
  cfg = propset()
  g = propset(globals)
  if _part_init != None:
    rblf.merge(cfg, g, _part_init(g))
  return (cfg, g)
`,
	},
	{
		desc:   "Inherit configuration with include",
		mkname: "product.mk",
		in: `
ifdef PRODUCT_NAME
include part.mk)
else
-include $(LOCAL_PATH)/part.mk)
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")
load(":part.star", _part_init = "init")

def init(globals):
  cfg = propset()
  g = propset(globals)
  if hasattr(g, "PRODUCT_NAME"):
    rblf.merge(cfg, g, _part_init(g))
  else:
    if _part_init != None:
      rblf.merge(cfg, g, _part_init(g))
  return (cfg, g)
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
load("//foo:font.star", _font_init = "init")
load("//bar:font.star", _font1_init = "init")

def init(globals):
  cfg = propset()
  g = propset(globals)
  rblf.merge(cfg, g, _font_init(g))
  rblf.merge(cfg, g, _font1_init(g))
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  rc = (cfg, g)
  rblf.warning("product.mk", "partially successful conversion")
  return rc
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if hasattr(g, "PRODUCT_NAME"):
    cfg.PRODUCT_NAME = "gizmo"
  else:
    pass
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  rblf.mkwarning("product.mk", "this is the warning")
  rblf.mkwarning("product.mk", "")
  rblf.mkinfo("product.mk", "this is the info")
  rblf.mkerror("product.mk", "this is the error")
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if hasattr(g, "PRODUCT_NAME"):
    # Comment
    pass
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if not hasattr(g, "PRODUCT_NAME"):
    cfg.PRODUCT_NAME = "gizmo1"
  else:
    cfg.PRODUCT_NAME = "gizmo2"
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if hasattr(g, "PRODUCT_NAME"):
    cfg.PRODUCT_NAME = "gizmo"
  elif not hasattr(g, "PRODUCT_PACKAGES"):
    pass
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if "aosp_arm" == getattr(g, "TARGET_PRODUCT", ""):
    cfg.PRODUCT_MODEL = "pix2"
  else:
    cfg.PRODUCT_MODEL = "pix21"
  if "aosp_x86" != getattr(g, "TARGET_PRODUCT", ""):
    cfg.PRODUCT_MODEL = "pix3"
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if getattr(g, "TARGET_BUILD_VARIANT", "") not in ["userdebug", "eng"]:
    pass
  if getattr(g, "TARGET_BUILD_VARIANT", "") in ["userdebug"]:
    pass
  if getattr(g, "TARGET_BUILD_VARIANT", "") in ["userdebug", "eng"]:
    pass
  if getattr(g, "TARGET_BUILD_VARIANT", "") in ["userdebug", "eng"]:
    pass
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if getattr(g, "TARGET_PRODUCT", "") not in ["yukawa_gms", "yukawa_sei510_gms"]:
    if getattr(g, "TARGET_PRODUCT", "") in ["yukawa_gms"]:
      pass
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if "aosp" == getattr(g, "TARGET_PRODUCT", ""):
    pass
  elif hasattr(g, "TARGET_PRODUCT"):
    pass
  return (cfg, g)
`,
	},
	{
		desc:   "Nested if",
		mkname: "product.mk",
		in: `
ifdef PRODUCT_NAME
  PRODUCT_PACKAGES = pack-if0
  ifdef PRODUCT_MODEL
    PRODUCT_PACKAGES = pack-if-if
  else ifdef PRODUCT_NAME
    PRODUCT_PACKAGES = pack-if-elif
  else
    PRODUCT_PACKAGES = pack-if-else
  endif
  PRODUCT_PACKAGES = pack-if
else ifneq (,$(TARGET_PRODUCT))
  PRODUCT_PACKAGES = pack-elif
else
  PRODUCT_PACKAGES = pack-else
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")

def init(globals):
  cfg = propset()
  g = propset(globals)
  if hasattr(g, "PRODUCT_NAME"):
    cfg.PRODUCT_PACKAGES = ["pack-if0"]
    if hasattr(g, "PRODUCT_MODEL"):
      cfg.PRODUCT_PACKAGES = ["pack-if-if"]
    elif hasattr(g, "PRODUCT_NAME"):
      cfg.PRODUCT_PACKAGES = ["pack-if-elif"]
    else:
      cfg.PRODUCT_PACKAGES = ["pack-if-else"]
    cfg.PRODUCT_PACKAGES = ["pack-if"]
  elif hasattr(g, "TARGET_PRODUCT"):
    cfg.PRODUCT_PACKAGES = ["pack-elif"]
  else:
    cfg.PRODUCT_PACKAGES = ["pack-else"]
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if not rblf.file_exists("foo.mk"):
    pass
  if rblf.file_wildcard_exists("foo*.mk"):
    pass
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if getattr(g, "VARIABLE", "") != "true":
    pass
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if "1" != "0":
    pass
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  if g.TARGET_BOARD_PLATFORM in ["msm8998"]:
    pass
  elif g.TARGET_BOARD_PLATFORM != "copper":
    pass
  elif g.TARGET_BOARD_PLATFORM not in g.QCOM_BOARD_PLATFORMS:
    pass
  elif g.TARGET_PRODUCT in getattr(g, "PLATFORM_LIST", []):
    pass
  return (cfg, g)
`,
	},
	{
		desc:   "findstring call",
		mkname: "product.mk",
		in: `
ifneq ($(findstring foo,$(PRODUCT_PACKAGES)),)
endif
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")

def init(globals):
  cfg = propset()
  g = propset(globals)
  if getattr(cfg, "PRODUCT_PACKAGES", []).find("foo") != -1:
    pass
  return (cfg, g)
`,
	},
	{
		desc:   "rhs call",
		mkname: "product.mk",
		in: `
PRODUCT_COPY_FILES = $(call add-to-product-copy-files-if-exists, path:distpath) \
 $(call find-copy-subdir-files, *, fromdir, todir)
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")

def init(globals):
  cfg = propset()
  g = propset(globals)
  cfg.PRODUCT_COPY_FILES = (rblf.copy_if_exists("path:distpath") +
      rblf.find_and_copy("*", "fromdir", "todir"))
  return (cfg, g)
`,
	},
	{
		desc:   "list with vars",
		mkname: "product.mk",
		in: `
PRODUCT_COPY_FILES += path1:$(TARGET_PRODUCT)/path1 $(PRODUCT_MODEL)/path2:$(TARGET_PRODUCT)/path2 
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")

def init(globals):
  cfg = propset()
  g = propset(globals)
  if not hasattr(cfg, "PRODUCT_COPY_FILES"):
    cfg.PRODUCT_COPY_FILES = []
  cfg.PRODUCT_COPY_FILES += [
      "path1:%s/path1" % getattr(g, "TARGET_PRODUCT", ""),
      "%s/path2:%s/path2" % (getattr(cfg, "PRODUCT_MODEL", ""), getattr(g, "TARGET_PRODUCT", "")),
  ]
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  rblf.enforce_product_packages_exist("")
  rblf.enforce_product_packages_exist("foo")
  rblf.require_artifacts_in_path("foo", "bar")
  return (cfg, g)
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

def init(globals):
  cfg = propset()
  g = propset(globals)
  cfg.PRODUCT_COPY_FILES = (rblf.find_and_copy("*.kl", "from1", "to1") +
      rblf.find_and_copy("*.kc", "from2", "to2") +
      [
          "foo",
          "bar",
      ])
  return (cfg, g)
`,
	},
	{
		desc:   "Text functions",
		mkname: "product.mk",
		in: `
PRODUCT_COPY_FILES := $(addprefix pfx-,a b c)
PRODUCT_COPY_FILES := $(addsuffix .sff, a b c)
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")

def init(globals):
  cfg = propset()
  g = propset(globals)
  cfg.PRODUCT_COPY_FILES = rblf.addprefix("pfx-", "a b c")
  cfg.PRODUCT_COPY_FILES = rblf.addsuffix(".sff", "a b c")
  return (cfg, g)
`,
	},
	{
		desc:   "assignment flavors",
		mkname: "product.mk",
		in: `
PRODUCT_LIST1 := a
PRODUCT_LIST2 += a
PRODUCT_LIST1 += b
PRODUCT_LIST2 += b
PRODUCT_LIST3 ?= a
PRODUCT_LIST1 = c
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")

def init(globals):
  cfg = propset()
  g = propset(globals)
  cfg.PRODUCT_LIST1 = ["a"]
  if not hasattr(cfg, "PRODUCT_LIST2"):
    cfg.PRODUCT_LIST2 = []
  cfg.PRODUCT_LIST2 += ["a"]
  cfg.PRODUCT_LIST1 += ["b"]
  cfg.PRODUCT_LIST2 += ["b"]
  if not hasattr(cfg, "PRODUCT_LIST3"):
    cfg.PRODUCT_LIST3 = ["a"]
  cfg.PRODUCT_LIST1 = ["c"]
  return (cfg, g)
`,
	},
	{
		desc:   "assigment flavors2",
		mkname: "product.mk",
		in: `
PRODUCT_LIST1 = a
ifeq (0,1)
  PRODUCT_LIST1 += b
  PRODUCT_LIST2 += b
endif
PRODUCT_LIST1 += c
PRODUCT_LIST2 += c
`,
		expected: `load("//build/make/target/product:product_config.star", "rblf")

def init(globals):
  cfg = propset()
  g = propset(globals)
  cfg.PRODUCT_LIST1 = ["a"]
  if "0" == "1":
    cfg.PRODUCT_LIST1 += ["b"]
    if not hasattr(cfg, "PRODUCT_LIST2"):
      cfg.PRODUCT_LIST2 = []
    cfg.PRODUCT_LIST2 += ["b"]
  cfg.PRODUCT_LIST1 += ["c"]
  if not hasattr(cfg, "PRODUCT_LIST2"):
    cfg.PRODUCT_LIST2 = []
  cfg.PRODUCT_LIST2 += ["c"]
  return (cfg, g)
`,
	},
}

var known_variables = []struct {
	name   string
	class  string
	flavor string
}{
	{"PRODUCT_NAME", "product", "string"},
	{"PRODUCT_MODEL", "product", "string"},
	{"PRODUCT_PACKAGES", "product", "list"},
	{"PRODUCT_COPY_FILES", "product", "list"},
	{"PRODUCT_IS_64BIT", "product", "string"},
	{"PRODUCT_LIST1", "product", "list"},
	{"PRODUCT_LIST2", "product", "list"},
	{"PRODUCT_LIST3", "product", "list"},
	{"TARGET_PRODUCT", "soong", "string"},
	{"TARGET_BUILD_VARIANT", "soong", "string"},
	{"TARGET_BOARD_PLATFORM", "soong", "string"},
	{"QCOM_BOARD_PLATFORMS", "soong", "string"},
	{"PLATFORM_LIST", "soong", "list"}, // TODO(asmundak): make it local instead of soong
}

func TestGood(t *testing.T) {
	for _, v := range known_variables {
		KnownVariables.NewVariable(v.name, v.class, v.flavor)
	}
	for _, test := range testCases {
		t.Run(test.desc,
			func(t *testing.T) {
				ss, err := Convert(Request{
					MkFile:       test.mkname,
					Reader:       bytes.NewBufferString(test.in),
					RootDir:      ".",
					OutputSuffix: ".star",
				})
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
