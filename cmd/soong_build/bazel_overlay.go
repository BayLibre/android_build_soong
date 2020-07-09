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

	soongModuleBzl = `
SoongModuleInfo = provider(
    fields = {
        "name": "Name of module",
        "variant": "Variant of module",
        "type": "Type of module",
    },
)

def _soong_module_impl(ctx):
    return [
        SoongModuleInfo(
            name = ctx.attrs.module_name,
            type = ctx.attrs.module_type,
            variant = ctx.attrs.module_variant,
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

func createBazelOverlay(ctx *android.Context, bazelOverlayDir string) error {
	var c *blueprint.Context
	c = ctx.Context
	modules := c.LogicModules()
	for _, module := range modules {
		moduleType := c.ModuleType(module)
		// Ignore Blueprint/Bootstrap modules
		if moduleType == "bootstrap_go_package" ||
			moduleType == "blueprint_go_binary" ||
			moduleType == "bootstrap_go_binary" {
			continue
		}
		dirPath := filepath.Join(
			bazelOverlayDir,
			strings.Replace(
				c.BlueprintFile(module), "/Android.bp", "", 1))
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			os.MkdirAll(dirPath, os.ModePerm)
		}
		buildFilePath := filepath.Join(dirPath, "BUILD.bazel")
		f, err := os.OpenFile(
			buildFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		fi, err := f.Stat()
		if err != nil {
			return err
		}
		if fi.Size() == 0 {
			f.Write([]byte("load(\"//:soong_module.bzl\", \"soong_module\")\n"))
		}
		// TODO: DirectDeps can have duplicate modules. Why?
		depLabels := map[string]bool{}
		for _, depModule := range c.DirectDeps(module) {
			depLabels["\""+c.QualifiedTargetLabel(depModule)+"\""] = true
		}
		depLabelList := ""
		for depLabel, _ := range depLabels {
			depLabelList += depLabel + ",\n        "
		}
		f.Write([]byte(
			fmt.Sprintf(
				soongModuleTarget,
				c.TargetNameWithVariant(module),
				c.ModuleName(module),
				c.ModuleType(module),
				c.ModuleSubDir(module),
				depLabelList)))
		f.Close()
	}

	workspaceFile := filepath.Join(bazelOverlayDir, "WORKSPACE")
	err := ioutil.WriteFile(workspaceFile, []byte{}, 0444) // 0444 is read-only
	if err != nil {
		return err
	}

	buildFile := filepath.Join(bazelOverlayDir, "BUILD")
	err = ioutil.WriteFile(buildFile, []byte{}, 0444)
	if err != nil {
		return err
	}

	soongModuleFile := filepath.Join(bazelOverlayDir, "soong_module.bzl")
	err = ioutil.WriteFile(soongModuleFile, []byte(soongModuleBzl), 0444)
	return err
}
