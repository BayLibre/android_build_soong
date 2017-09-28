package cc

import (
	"android/soong/android"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/google/blueprint/proptools"
)

var buildDir string

func setUp() {
	var err error
	buildDir, err = ioutil.TempDir("", "soong_cc_test")
	if err != nil {
		panic(err)
	}
}

func tearDown() {
	os.RemoveAll(buildDir)
}

func TestMain(m *testing.M) {
	run := func() int {
		setUp()
		defer tearDown()

		return m.Run()
	}

	os.Exit(run())
}

func testCc(t *testing.T, bp string, allowMissingDependencies bool) *android.TestContext {
	config := android.TestArchConfig(buildDir, nil)
	config.ProductVariables.DeviceVndkVersion = proptools.StringPtr("current")

	ctx := android.NewTestArchContext()
	ctx.SetAllowMissingDependencies(allowMissingDependencies)
	ctx.RegisterModuleType("cc_library", android.ModuleFactoryAdaptor(libraryFactory))
	ctx.RegisterModuleType("toolchain_library", android.ModuleFactoryAdaptor(toolchainLibraryFactory))
	ctx.RegisterModuleType("llndk_library", android.ModuleFactoryAdaptor(llndkLibraryFactory))
	ctx.PreDepsMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.BottomUp("image", vendorMutator).Parallel()
		ctx.BottomUp("link", linkageMutator).Parallel()
		ctx.BottomUp("vndk", vndkMutator).Parallel()
	})
	ctx.Register()

	ctx.MockFileSystem(map[string][]byte{
		"Android.bp": []byte(bp),
		"foo.c":      nil,
		"bar.c":      nil,
	})

	_, errs := ctx.ParseBlueprintsFiles("Android.bp")
	failIfErrored(t, errs)
	_, errs = ctx.PrepareBuildActions(config)
	failIfErrored(t, errs)

	return ctx
}

func TestVendorSrc(t *testing.T) {
	ctx := testCc(t, `
		cc_library {
			name: "libTest",
			srcs: ["foo.c"],
			no_libgcc : true,
			nocrt : true,
			system_shared_libs : [],
			vendor_available: true,
			target: {
				vendor: {
					srcs: ["bar.c"],
				},
			},
		}
		toolchain_library {
			name: "libatomic",
			vendor_available: true,
		}
		toolchain_library {
			name: "libcompiler_rt-extras",
			vendor_available: true,
		}
		cc_library {
			name: "libc",
			no_libgcc : true,
			nocrt : true,
			system_shared_libs: [],
		}
		llndk_library {
			name: "libc",
			symbol_file: "",
		}
		cc_library {
			name: "libm",
			no_libgcc : true,
			nocrt : true,
			system_shared_libs: [],
		}
		llndk_library {
			name: "libm",
			symbol_file: "",
		}
		cc_library {
			name: "libdl",
			no_libgcc : true,
			nocrt : true,
			system_shared_libs: [],
		}
		llndk_library {
			name: "libdl",
			symbol_file: "",
		}
	`, false)

	ld := ctx.ModuleForTests("libTest", "android_arm_armv7-a-neon_vendor_shared").Rule("ld")
	var objs []string
	for _, o := range ld.Inputs {
		objs = append(objs, o.Base())
	}
	if len(objs) != 2 {
		t.Errorf("inputs of libTest is expected to 2, but was %d.", len(objs))
	}
	if objs[0] != "foo.o" || objs[1] != "bar.o" {
		t.Errorf("inputs of libTest must be []string{\"foo.o\", \"bar.o\"}, but was %#v.", objs)
	}
}

var firstUniqueElementsTestCases = []struct {
	in  []string
	out []string
}{
	{
		in:  []string{"a"},
		out: []string{"a"},
	},
	{
		in:  []string{"a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"a", "a"},
		out: []string{"a"},
	},
	{
		in:  []string{"a", "b", "a"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"b", "a", "a"},
		out: []string{"b", "a"},
	},
	{
		in:  []string{"a", "a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"a", "b", "a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"liblog", "libdl", "libc++", "libdl", "libc", "libm"},
		out: []string{"liblog", "libdl", "libc++", "libc", "libm"},
	},
}

func TestFirstUniqueElements(t *testing.T) {
	for _, testCase := range firstUniqueElementsTestCases {
		out := firstUniqueElements(testCase.in)
		if !reflect.DeepEqual(out, testCase.out) {
			t.Errorf("incorrect output:")
			t.Errorf("     input: %#v", testCase.in)
			t.Errorf("  expected: %#v", testCase.out)
			t.Errorf("       got: %#v", out)
		}
	}
}

var lastUniqueElementsTestCases = []struct {
	in  []string
	out []string
}{
	{
		in:  []string{"a"},
		out: []string{"a"},
	},
	{
		in:  []string{"a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"a", "a"},
		out: []string{"a"},
	},
	{
		in:  []string{"a", "b", "a"},
		out: []string{"b", "a"},
	},
	{
		in:  []string{"b", "a", "a"},
		out: []string{"b", "a"},
	},
	{
		in:  []string{"a", "a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"a", "b", "a", "b"},
		out: []string{"a", "b"},
	},
	{
		in:  []string{"liblog", "libdl", "libc++", "libdl", "libc", "libm"},
		out: []string{"liblog", "libc++", "libdl", "libc", "libm"},
	},
}

func TestLastUniqueElements(t *testing.T) {
	for _, testCase := range lastUniqueElementsTestCases {
		out := lastUniqueElements(testCase.in)
		if !reflect.DeepEqual(out, testCase.out) {
			t.Errorf("incorrect output:")
			t.Errorf("     input: %#v", testCase.in)
			t.Errorf("  expected: %#v", testCase.out)
			t.Errorf("       got: %#v", out)
		}
	}
}

var (
	str11 = "01234567891"
	str10 = str11[:10]
	str9  = str11[:9]
	str5  = str11[:5]
	str4  = str11[:4]
)

var splitListForSizeTestCases = []struct {
	in   []string
	out  [][]string
	size int
}{
	{
		in:   []string{str10},
		out:  [][]string{{str10}},
		size: 10,
	},
	{
		in:   []string{str9},
		out:  [][]string{{str9}},
		size: 10,
	},
	{
		in:   []string{str5},
		out:  [][]string{{str5}},
		size: 10,
	},
	{
		in:   []string{str11},
		out:  nil,
		size: 10,
	},
	{
		in:   []string{str10, str10},
		out:  [][]string{{str10}, {str10}},
		size: 10,
	},
	{
		in:   []string{str9, str10},
		out:  [][]string{{str9}, {str10}},
		size: 10,
	},
	{
		in:   []string{str10, str9},
		out:  [][]string{{str10}, {str9}},
		size: 10,
	},
	{
		in:   []string{str5, str4},
		out:  [][]string{{str5, str4}},
		size: 10,
	},
	{
		in:   []string{str5, str4, str5},
		out:  [][]string{{str5, str4}, {str5}},
		size: 10,
	},
	{
		in:   []string{str5, str4, str5, str4},
		out:  [][]string{{str5, str4}, {str5, str4}},
		size: 10,
	},
	{
		in:   []string{str5, str4, str5, str5},
		out:  [][]string{{str5, str4}, {str5}, {str5}},
		size: 10,
	},
	{
		in:   []string{str5, str5, str5, str4},
		out:  [][]string{{str5}, {str5}, {str5, str4}},
		size: 10,
	},
	{
		in:   []string{str9, str11},
		out:  nil,
		size: 10,
	},
	{
		in:   []string{str11, str9},
		out:  nil,
		size: 10,
	},
}

func TestSplitListForSize(t *testing.T) {
	for _, testCase := range splitListForSizeTestCases {
		out, _ := splitListForSize(android.PathsForTesting(testCase.in), testCase.size)

		var outStrings [][]string

		if len(out) > 0 {
			outStrings = make([][]string, len(out))
			for i, o := range out {
				outStrings[i] = o.Strings()
			}
		}

		if !reflect.DeepEqual(outStrings, testCase.out) {
			t.Errorf("incorrect output:")
			t.Errorf("     input: %#v", testCase.in)
			t.Errorf("      size: %d", testCase.size)
			t.Errorf("  expected: %#v", testCase.out)
			t.Errorf("       got: %#v", outStrings)
		}
	}
}

var staticLinkDepOrderTestCases = []struct {
	// inDeps is a string representation of a map[moduleName][]moduleDependency
	// inDeps models the dependencies declared in an Android.bp file
	inDeps string

	// outDeps is a string representation of a map[moduleName][]moduleDependency
	// The keys of outDeps specify which modules we would like to execute the test against.
	// The values of outDeps specify the expected result for each module to test.
	outDeps string
}{
	// Simple tests
	{
		inDeps:  "",
		outDeps: "",
	},
	{
		inDeps:  "a:",
		outDeps: "a:",
	},
	{
		inDeps:  "a:b; b:",
		outDeps: "a:b; b:",
	},
	// Tests of reordering
	{
		// diamond example
		inDeps:  "a:d,b,c; b:d; c:d; d:",
		outDeps: "a:b,c,d; b:d; c:d; d:",
	},
	{
		// somewhat real example
		inDeps:  "bsdiff_unittest:b,c,d,e,f,g,h,i; e:b",
		outDeps: "bsdiff_unittest:c,d,e,b,f,g,h,i; e:b",
	},
	{
		// multiple reorderings
		inDeps:  "a:b,c,d,e; d:b; e:c",
		outDeps: "a:d,b,e,c; d:b; e:c",
	},
	{
		// should reorder without adding new transitive dependencies
		inDeps:  "bin:lib1,lib2; lib2:lib1,liboptional",
		outDeps: "bin:lib2,lib1; lib2:lib1,liboptional",
	},
	{
		// Doesn't need to check more than one level of transitive dependencies
		// (though f->b and b->c are declared, f->c is not declared, so f need not precede c,
		// (and even b does not need to precede c), so c should remain in its original position).
		// It's unclear what the ideal behavior would be in this case, but this test
		// serves to enforce that if this behavior is changed then the change is intentional.
		inDeps:  "a:b,c,d,e,f,g,h; b:c; c:d; f:b",
		outDeps: "a:c,d,e,f,b,g,h; b:c; c:d; f:b",
	},
	{
		// if transitive dependencies are declared, they should be respected
		inDeps:  "a:b,c,d,e,f,g,h; b:c,d; c:d; f:b,c,d",
		outDeps: "a:e,f,b,c,d,g,h; b:c,d; c:d; f:b,c,d",
	},
	{
		// When computing the link order of the dependencies of module M, only modules that
		// are explicitly declared as being in the transitive dependency set of M should
		// be used for calculating dependency ordering.
		//
		// Although bin->ifaces->impl1->lib1, bin doesn't depend on impl1 so ifaces need not precede
		// lib1 .
		//
		// (although this requirement is implied by the
		// not-having-to-reevaluate-dependencies-recursively requirement, it is still independent
		// and could still be true even if we remove the transitivity requirement)
		inDeps:  "lib1:; impl1:lib1; ifaces:impl1,impl2; bin:lib1,ifaces,impl2",
		outDeps: "lib1:; impl1:lib1; ifaces:impl1,impl2; bin:lib1,ifaces,impl2",
	},
	// Tests involving duplicate dependencies
	{
		// simple duplicate
		inDeps:  "a:b,c,c,b",
		outDeps: "a:c,b",
	},
	{
		// duplicates with reordering
		inDeps:  "a:b,c,d,c; c:b",
		outDeps: "a:d,c,b",
	},
	// Tests to confirm the nonexistence of infinite loops.
	// These cases should never happen, so as long as the test terminates and the
	// result is deterministic then that should be fine.
	{
		inDeps:  "a:a",
		outDeps: "a:a",
	},
	{
		inDeps:  "a:b; b:c; c:a",
		outDeps: "a:b; b:c; c:a",
	},
	{
		inDeps:  "a:b,c; b:c,a; c:a,b",
		outDeps: "a:c,b",
	},
}

// converts from a string like "a:b,c; d:e" to (["a","b"], {"a":["b","c"], "d":["e"]}, [{"a", "a.o"}, {"b", "b.o"}])
func parseModuleDeps(text string) (modulesInOrder []android.Path, allDeps map[android.Path][]android.Path) {
	// convert from "a:b,c; d:e" to "a:b,c;d:e"
	strippedText := strings.Replace(text, " ", "", -1)
	if len(strippedText) < 1 {
		return []android.Path{}, make(map[android.Path][]android.Path, 0)
	}
	allDeps = make(map[android.Path][]android.Path, 0)

	// convert from "a:b,c;d:e" to ["a:b,c", "d:e"]
	moduleTexts := strings.Split(strippedText, ";")

	outputForModuleName := func(moduleName string) android.Path {
		return android.PathForTesting(moduleName)
	}

	for _, moduleText := range moduleTexts {
		// convert from "a:b,c" to ["a", "b,c"]
		components := strings.Split(moduleText, ":")
		if len(components) != 2 {
			panic(fmt.Sprintf("illegal module dep string %q from larger string %q; must contain one ':', not %v", moduleText, text, len(components)-1))
		}
		moduleName := components[0]
		moduleOutput := outputForModuleName(moduleName)
		modulesInOrder = append(modulesInOrder, moduleOutput)

		depString := components[1]
		// convert from "b,c" to ["b", "c"]
		depNames := strings.Split(depString, ",")
		if len(depString) < 1 {
			depNames = []string{}
		}
		var deps []android.Path
		for _, depName := range depNames {
			deps = append(deps, outputForModuleName(depName))
		}
		allDeps[moduleOutput] = deps
	}
	return modulesInOrder, allDeps
}

func TestStaticLinkDependencyOrdering(t *testing.T) {
	for _, testCase := range staticLinkDepOrderTestCases {
		errs := []string{}

		// confirm that no more than one module's dependencies will need reordering
		_, givenTransitiveDeps := parseModuleDeps(testCase.inDeps)
		expectedModuleNames, expectedTransitiveDeps := parseModuleDeps(testCase.outDeps)

		// For each module whose post-reordered dependencies were specified, validate that
		// reordering the inputs produces the expected outputs. The main reason that we support
		// validating multiple modules is to guard against typos within the tests themselves
		for _, moduleName := range expectedModuleNames {
			moduleDeps := givenTransitiveDeps[moduleName]
			reordered := orderDeps(moduleDeps, givenTransitiveDeps)
			correctDeps := expectedTransitiveDeps[moduleName]
			if !reflect.DeepEqual(correctDeps, reordered) {
				errs = append(errs, fmt.Sprintf("orderDeps failed to correctly reorder dependencies."+
					"\nInput:    %q"+
					"\nmodule:   %v"+
					"\nexpected: %#v"+
					"\nactual:   %#v",
					testCase.inDeps, moduleName, correctDeps, reordered))
			}
		}

		if len(errs) > 0 {
			sort.Strings(errs)
			for _, err := range errs {
				t.Error(err)
			}
		}
	}
}
func failIfErrored(t *testing.T, errs []error) {
	if len(errs) > 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.FailNow()
	}
}

func getOutputPaths(ctx *android.TestContext, variant string, moduleNames []string) (paths android.Paths) {
	for _, moduleName := range moduleNames {
		module := ctx.ModuleForTests(moduleName, variant).Module().(*Module)
		output := module.outputFile.Path()
		paths = append(paths, output)
	}
	return paths
}

func TestLibDeps(t *testing.T) {
	ctx := testCc(t, `
	cc_library {
		name: "a",
		static_libs: ["b", "c", "d"],
	}
	cc_library {
		name: "b",
	}
	cc_library {
		name: "c",
		static_libs: ["b"],
	}
	cc_library {
		name: "d",
	}

	`, true)

	variant := "android_arm64_armv8-a_core_static"
	moduleA := ctx.ModuleForTests("a", variant).Module().(*Module)
	actual := moduleA.staticDepsInLinkOrder
	expected := getOutputPaths(ctx, variant, []string{"c", "b", "d"})

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("staticDeps orderings were not propagated correctly"+
			"\nactual:   %v"+
			"\nexpected: %v",
			actual,
			expected,
		)
	}

}
