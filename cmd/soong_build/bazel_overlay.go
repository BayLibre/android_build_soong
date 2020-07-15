package main

import (
	"android/soong/android"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
)

const (
	soongModuleLoad = `package(default_visibility = ["//visibility:public"])
load("//:soong_module.bzl", "soong_module")
`

	// A BUILD file target snippet representing a Soong module
	soongModuleTarget = `soong_module(
    name = "%s",
    module_name = "%s",
    module_type = "%s",
    module_variant = "%s",
    deps = [
        %s
    ],
)
`

	// The soong_module rule implementation in a .bzl file
	soongModuleBzl = `
SoongModuleInfo = provider(
    fields = {
        "name": "Name of module",
        "type": "Type of module",
        "variant": "Variant of module",
    },
)

def _soong_module_impl(ctx):
    return [
        SoongModuleInfo(
            name = ctx.attr.module_name,
            type = ctx.attr.module_type,
            variant = ctx.attr.module_variant,
        ),
    ]

soong_module = rule(
    implementation = _soong_module_impl,
    attrs = {
        "module_name": attr.string(mandatory = True),
        "module_type": attr.string(mandatory = True),
        "module_variant": attr.string(),
        "deps": attr.label_list(providers = [SoongModuleInfo]),
    },
)
`
)

func targetNameWithVariant(c *blueprint.Context, logicModule blueprint.Module) string {
	name := ""
	if c.ModuleSubDir(logicModule) != "" {
		name = c.ModuleName(logicModule) + "--" + c.ModuleSubDir(logicModule)
	} else {
		name = c.ModuleName(logicModule)
	}

	return strings.Replace(name, "//", "", 1)
}

func qualifiedTargetLabel(c *blueprint.Context, logicModule blueprint.Module) string {
	return "//" +
		packagePath(c, logicModule) +
		":" +
		targetNameWithVariant(c, logicModule)
}

func packagePath(c *blueprint.Context, logicModule blueprint.Module) string {
	packagePathParts := strings.Split(c.BlueprintFile(logicModule), "/")
	return strings.Join(packagePathParts[:len(packagePathParts)-1], "/")
}

func createBazelOverlay(ctx *android.Context, bazelOverlayDir string) error {
	blueprintCtx := ctx.Context
	blueprintCtx.VisitAllModules(func(module blueprint.Module) {
		buildFile, err := buildFileForModule(blueprintCtx, module)
		if err != nil {
			panic(err)
		}

		// TODO(soong-team): DirectDeps can have duplicate (module, variant)
		// items. Why?
		depLabels := map[string]bool{}
		blueprintCtx.VisitDirectDeps(module, func(depModule blueprint.Module) {
			// For convenience, wrap the target label with quotes to be
			// templated into the string later.
			depLabels[qualifiedTargetLabel(blueprintCtx, depModule)] = true
		})

		depLabelList := ""
		for depLabel, _ := range depLabels {
			depLabelList += "\"" + depLabel + "\",\n        "
		}
		buildFile.Write([]byte(
			fmt.Sprintf(
				soongModuleTarget,
				targetNameWithVariant(blueprintCtx, module),
				blueprintCtx.ModuleName(module),
				blueprintCtx.ModuleType(module),
				// misleading name, this actually returns the variant.
				blueprintCtx.ModuleSubDir(module),
				depLabelList)))
		buildFile.Close()
	})

	if err := writeReadOnlyFile(bazelOverlayDir, "WORKSPACE", ""); err != nil {
		return err
	}

	if err := writeReadOnlyFile(bazelOverlayDir, "BUILD", ""); err != nil {
		return err
	}

	return writeReadOnlyFile(bazelOverlayDir, "soong_module.bzl", soongModuleBzl)
}

func buildFileForModule(ctx *blueprint.Context, module blueprint.Module) (*os.File, error) {
	// Create nested directories for the BUILD file
	dirPath := filepath.Join(bazelOverlayDir, packagePath(ctx, module))
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		os.MkdirAll(dirPath, os.ModePerm)
	}
	// Open the file for appending, and create it if it doesn't exist
	f, err := os.OpenFile(
		filepath.Join(dirPath, "BUILD.bazel"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644)
	if err != nil {
		return nil, err
	}

	// If the file is empty, add the load statement for the `soong_module` rule
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() == 0 {
		f.Write([]byte(soongModuleLoad + "\n"))
	}

	return f, nil
}

// The overlay directory should be read-only, sufficient for bazel query.
func writeReadOnlyFile(dir string, baseName string, content string) error {
	workspaceFile := filepath.Join(bazelOverlayDir, baseName)
	// 0444 is read-only
	return ioutil.WriteFile(workspaceFile, []byte(content), 0444)
}
