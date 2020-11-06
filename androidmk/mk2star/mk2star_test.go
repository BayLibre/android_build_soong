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
`,
		expected: `#  Comment
load("//build/make/target/product:product_config.star", "prodconf")
product = prodconf(
    "product",
    [],
)
`,
	},
	{
		desc:   "Item variable",
		mkname: "pixel3.mk",
		in: `
PRODUCT_NAME := Pixel 3
`,
		expected: `_vars = dict()
_vars["PRODUCT_NAME"] = "Pixel 3"
load("//build/make/target/product:product_config.star", "prodconf")
pixel3 = prodconf(
    "pixel3",
    [],
    PRODUCT_NAME = _vars["PRODUCT_NAME"],
)
`,
	},
	{
		desc:   "List variable",
		mkname: "pixel4.mk",
		in: `
PRODUCT_PACKAGES = package1 package2
PRODUCT_COPY_FILES += file2:target 
PRODUCT_PACKAGES += package3
`,
		expected: `_vars = dict()
_vars["PRODUCT_PACKAGES"] = [
    "package1",
    "package2",
]
_vars["PRODUCT_COPY_FILES"] = ["file2:target"]
_vars["PRODUCT_PACKAGES"] = _vars["PRODUCT_PACKAGES"] + ["package3"]
load("//build/make/target/product:product_config.star", "prodconf")
pixel4 = prodconf(
    "pixel4",
    [],
    PRODUCT_COPY_FILES = _vars["PRODUCT_COPY_FILES"],
    PRODUCT_PACKAGES = _vars["PRODUCT_PACKAGES"],
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
		desc:   "sub config",
		mkname: "product.mk",
		in: `
$(call inherit-product, part.mk)
`,
		expected: `load("//build/make/target/product:product_config.star", "prodconf")
load(":part.star", "part")
product = prodconf(
    "product",
    [part],
)
`,
	},
	{
		desc:   "conditional subconfig",
		mkname: "product.mk",
		in: `
$(call inherit-product-if-exists, part.mk)
`,
		expected: `load("//build/make/target/product:product_config.star", "prodconf")
# MK2STAR TRANSLATION ERROR: conditional load not supported
# conditional_load(":part.star", "part")
product = prodconf(
    "product",
    [],
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
load("//build/make/target/product:product_config.star", "prodconf")
product = prodconf(
    "product",
    [],
)

print("The conversion of product.mk to this script was only partially successful. Please fix the problems by inspecting commented out lines and then remove this print statement")
`,
	},
}

var known_variables = []struct {
	name   string
	flavor string
}{
	{"PRODUCT_NAME", "item"},
	{"PRODUCT_MODEL", "item"},
	{"PRODUCT_PACKAGES", "list"},
	{"PRODUCT_COPY_FILES", "list"},
}

func TestGood(t *testing.T) {
	for _, v := range known_variables {
		_ = ConfigVariables.newVariable(v.name, v.flavor)
	}

	for _, test := range testCases {
		ss, err := Convert(test.mkname, bytes.NewBufferString(test.in))
		if err != nil {
			t.Error(err)
		}
		got := ss.String()
		if got != test.expected {
			t.Errorf("test %s failed\nExpected:\n%s\nActual:\n%s\n", test.desc,
				strings.ReplaceAll(test.expected, "\n", "␤\n"),
				strings.ReplaceAll(got, "\n", "␤\n"))
		}
	}
}
