package common

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/bootstrap"
)

var (
	androidmkCmd  = filepath.Join(bootstrap.BinDir, "soong_build")
	androidmkRule = pctx.StaticRule("AndroidMk",
		blueprint.RuleParams{
			Command:     androidmkCmd + " -androidmk $srcDir/Android.bp $bpFiles",
			Description: "androidmk",
			Restat:      true,
		}, "bpFiles")

	translatedDir = "androidmk"
)

type AndroidMkDataProvider interface {
	AndroidMk() AndroidMkData
}

type AndroidMkData struct {
	Class      string
	OutputFile string

	Custom func(w io.Writer, name, prefix string)

	Extra func(name, prefix string) []string
}

func AndroidMkSingleton() blueprint.Singleton {
	return &androidMkSingleton{}
}

type androidMkSingleton struct{}

func (c *androidMkSingleton) GenerateBuildActions(ctx blueprint.SingletonContext) {
	fileModules := make(map[string][]blueprint.Module)
	hasBPFile := make(map[string]bool)
	bpFiles := []string{}

	ctx.VisitAllModules(func(module blueprint.Module) {
		if _, ok := module.(AndroidModule); ok {
			bpFile := ctx.BlueprintFile(module)

			if !hasBPFile[bpFile] {
				hasBPFile[bpFile] = true
				bpFiles = append(bpFiles, bpFile)
			}

			fileModules[bpFile] = append(fileModules[bpFile], module)
		}
	})

	// Gather list of eligible Android modules for translation
	androidMkModules := make(map[blueprint.Module]bool)
	var transMks []string
	var validBpFiles []string
	srcDir := ctx.Config().(Config).SrcDir()
	intermediatesDir := filepath.Join(ctx.Config().(Config).IntermediatesDir(), "androidmk")
	sort.Strings(bpFiles)
	for _, origBp := range bpFiles {
		mkFile := filepath.Join(srcDir, filepath.Dir(origBp), "Android.mk")

		files, err := Glob(ctx, intermediatesDir, mkFile, nil)
		if err != nil {
			ctx.Errorf("glob: %s", err.Error())
			continue
		}

		// Existing Android.mk file, use that instead
		if len(files) > 0 {
			for _, file := range files {
				ctx.AddNinjaFileDeps(file)
			}
			continue
		}

		validBpFiles = append(validBpFiles, origBp)

		for _, mod := range fileModules[origBp] {
			androidMkModules[mod] = true
		}

		transMk := filepath.Join(translatedDir, "Android_"+strings.Replace(filepath.Dir(origBp), "/", "_", -1)+".mk")
		transMks = append(transMks, transMk)
	}

	// Validate that all modules have proper dependencies, collect outputs
	var androidMkDeps []string
	for mod := range androidMkModules {
		ctx.VisitDepsDepthFirstIf(mod, isAndroidModule, func(module blueprint.Module) {
			if !androidMkModules[module] {
				ctx.Errorf("Module %q missing dependency for Android.mk: %q", ctx.ModuleName(mod), ctx.ModuleName(module))
			}
		})

		if provider, ok := mod.(AndroidMkDataProvider); ok {
			androidMkDeps = append(androidMkDeps, provider.AndroidMk().OutputFile)
		}
	}

	sort.Strings(androidMkDeps)

	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      blueprint.Phony,
		Outputs:   []string{"all_mk_modules"},
		Implicits: androidMkDeps,
		Optional:  true,
	})

	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      androidmkRule,
		Outputs:   transMks,
		Implicits: []string{"build.ninja"},
		Optional:  true,
		Args: map[string]string{
			"bpFiles": strings.Join(validBpFiles, " "),
		},
	})

	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      blueprint.Phony,
		Outputs:   []string{"androidmk"},
		Implicits: transMks,
		Optional:  true,
	})
}

func translateAndroidMkModule(ctx *blueprint.Context, w io.Writer, mod blueprint.Module) error {
	if mod != ctx.PrimaryModule(mod) {
		// These will be handled by the primary module
		return nil
	}

	name := ctx.ModuleName(mod)

	type hostClass struct {
		host     bool
		class    string
		multilib string
	}

	type archSrc struct {
		arch  string
		src   string
		extra []string
	}

	srcs := make(map[hostClass][]archSrc)
	var modules []hostClass

	ctx.VisitAllModuleVariants(mod, func(m blueprint.Module) {
		provider, ok := m.(AndroidMkDataProvider)
		if !ok {
			return
		}

		amod := m.(AndroidModule).base()
		data := provider.AndroidMk()

		arch := amod.commonProperties.CompileArch.ArchType.String()

		prefix := ""
		// TODO: make this dynamic
		if amod.HostOrDevice() == Host {
			if arch == "x86" {
				prefix = "2ND_"
			}
		} else {
			if arch == "arm" {
				prefix = "2ND_"
			}
		}

		if data.Custom != nil {
			data.Custom(w, name, prefix)
			return
		}

		hC := hostClass{
			host:     amod.HostOrDevice() == Host,
			class:    data.Class,
			multilib: amod.commonProperties.Compile_multilib,
		}

		src := archSrc{
			arch: arch,
			src:  data.OutputFile,
		}

		if data.Extra != nil {
			src.extra = data.Extra(name, prefix)
		}

		if srcs[hC] == nil {
			modules = append(modules, hC)
		}
		srcs[hC] = append(srcs[hC], src)
	})

	for _, hC := range modules {
		archSrcs := srcs[hC]

		io.WriteString(w, "\ninclude $(CLEAR_VARS)\n")
		io.WriteString(w, "LOCAL_MODULE := "+name+"\n")
		io.WriteString(w, "LOCAL_MODULE_CLASS := "+hC.class+"\n")
		io.WriteString(w, "LOCAL_MULTILIB := "+hC.multilib+"\n")
		io.WriteString(w, "LOCAL_CXX_STL := none\n")

		for _, src := range archSrcs {
			io.WriteString(w, "$(SOONG_OUT_DIR)/"+src.src+": build-soong ;\n")
			io.WriteString(w, "LOCAL_SRC_FILES_"+src.arch+" := "+src.src+"\n")

			for _, extra := range src.extra {
				io.WriteString(w, extra+"\n")
			}
		}

		// TODO: relative_install_path -> LOCAL_MODULE_RELATIVE_PATH
		// TODO: LOCAL_REQUIRED_MODULES

		if hC.host {
			// TODO: this isn't true for every module
			io.WriteString(w, "LOCAL_ACP_UNAVAILABLE := true\n")

			io.WriteString(w, "LOCAL_IS_HOST_MODULE := true\n")
		}
		io.WriteString(w, "include $(BUILD_PREBUILT)\n")
	}

	return nil
}

func translateAndroidMk(ctx *blueprint.Context, dir, mkFile string, mods []blueprint.Module) error {
	buf := &bytes.Buffer{}

	io.WriteString(buf, "LOCAL_PATH := $(SOONG_OUT_DIR)\n")
	io.WriteString(buf, "LOCAL_MODULE_MAKEFILE := $(lastword $(MAKEFILE_LIST))\n")
	io.WriteString(buf, "LOCAL_SRC_PATH := "+dir+"\n")

	for _, mod := range mods {
		err := translateAndroidMkModule(ctx, buf, mod)
		if err != nil {
			os.Remove(mkFile)
			return err
		}
	}

	io.WriteString(buf, "\nLOCAL_SRC_PATH :=\n")

	f, err := os.Create(mkFile)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.Write(buf.Bytes())
	return err
}

func CreateAndroidMkFiles(ctx *blueprint.Context, bpFiles []string) error {
	fileModules := make(map[string][]blueprint.Module)
	ctx.VisitAllModules(func(module blueprint.Module) {
		if _, ok := module.(AndroidModule); ok {
			bpFile := ctx.BlueprintFile(module)

			fileModules[bpFile] = append(fileModules[bpFile], module)
		}
	})

	for _, origBp := range bpFiles {
		transMk := filepath.Join(translatedDir, "Android_"+strings.Replace(filepath.Dir(origBp), "/", "_", -1)+".mk")

		translateAndroidMk(ctx, filepath.Dir(origBp), transMk, fileModules[origBp])
		fmt.Printf("Writing %s...\n", transMk)
	}

	return nil
}
