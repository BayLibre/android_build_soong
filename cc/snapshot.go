package cc

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"android/soong/android"
)

var (
	vendorSnapshotZipKey = android.NewOnceKey("vendorSnapshotZip")
	vndkSnapshotZipKey   = android.NewOnceKey("vndkSnapshotZip")
	modulePathsKey       = android.NewOnceKey("modulePaths")
)

func modulePaths(config android.Config) map[string]string {
	return config.Once(modulePathsKey, func() interface{} {
		return make(map[string]string)
	}).(map[string]string)
}

func vendorSnapshotZip(config android.Config) *android.OutputPath {
	return config.Once(vendorSnapshotZipKey, func() interface{} {
		return &android.OutputPath{}
	}).(*android.OutputPath)
}

func vndkSnapshotZip(config android.Config) *android.OutputPath {
	return config.Once(vndkSnapshotZipKey, func() interface{} {
		return &android.OutputPath{}
	}).(*android.OutputPath)
}

func init() {
	android.RegisterSingletonType("snapshot", SnapshotSingleton)
	android.RegisterMakeVarsProvider(pctx, func(ctx android.MakeVarsContext) {
		ctx.Strict("SOONG_VNDK_SNAPSHOT_ZIP", vndkSnapshotZip(ctx.Config()).String())
		ctx.Strict("SOONG_VENDOR_SNAPSHOT_ZIP", vendorSnapshotZip(ctx.Config()).String())
	})
}

func SnapshotSingleton() android.Singleton {
	return &snapshotSingleton{}
}

type snapshotSingleton struct{}

type snapshotLibraryInterface interface {
	exportedFlagsProducer
	libraryInterface
}

func installSnapshotFileFromPath(ctx android.SingletonContext, path android.Path, snapshotDir, out string) android.OutputPath {
	outPath := android.PathForOutput(ctx, snapshotDir, out)
	ctx.Build(pctx, android.BuildParams{
		Rule:        android.Cp,
		Input:       path,
		Output:      outPath,
		Description: "snapshot " + out,
		Args: map[string]string{
			"cpFlags": "-f -L",
		},
	})
	return outPath
}

func installSnapshotFileFromContent(ctx android.SingletonContext, content, snapshotDir, out string) android.OutputPath {
	outPath := android.PathForOutput(ctx, snapshotDir, out)
	ctx.Build(pctx, android.BuildParams{
		Rule:        android.WriteFile,
		Output:      outPath,
		Description: "snapshot " + out,
		Args: map[string]string{
			"content": content,
		},
	})
	return outPath
}

func writePathsToFile(ctx android.SingletonContext, paths android.Paths, file android.OutputPath) {
	var intermediates android.Paths

	rule := android.NewRuleBuilder()
	rule.Command().
		Text("rm").
		Flag("-rf").
		Text(file.String())

	rule.Command().
		Text("mkdir").
		Flag("-p").
		Text(filepath.Dir(file.String()))

	if len(paths) == 0 {
		rule.Command().
			Text("touch").
			Output(file)
		rule.Build(pctx, ctx, file.String(), "writing paths "+file.String())
		return
	}

	tmp_no := 0
	idx := 0
	for idx < len(paths) {
		tmpFile := android.PathForOutput(ctx, fmt.Sprintf("%s.%d", file.Rel(), tmp_no))
		tmp_no++

		r := idx
		totalLength := 0
		for r < len(paths) && totalLength+len(paths[r].String())+1 < 100000 {
			totalLength += len(paths[r].String()) + 1
			r++
		}
		ctx.Build(pctx, android.BuildParams{
			Rule:        android.WriteFile,
			Output:      tmpFile,
			Implicits:   paths[idx:r],
			Description: "writing paths " + tmpFile.String(),
			Args: map[string]string{
				"content": android.JoinWithSuffix(paths[idx:r].Strings(), "\\n", ""),
			},
		})
		intermediates = append(intermediates, tmpFile)
		idx = r
	}

	rule.Command().
		Text("cat ").
		Inputs(intermediates).
		FlagWithOutput("> ", file)

	rule.Build(pctx, ctx, file.String(), "writing paths "+file.String())
}

func generateSnapshotFiles(ctx android.SingletonContext, snapshotDir string, includeBuildArtifacts bool,
	pred func(ctx android.SingletonContext, m *Module) (i snapshotLibraryInterface, libDir string, isSnapshotLib bool)) []android.OutputPath {
	var outputs []android.OutputPath

	archLibDir := make(map[android.ArchType]string)

	snapshotVariantDir := ctx.DeviceConfig().DeviceArch()
	for _, target := range ctx.Config().Targets[android.Android] {
		dir := snapshotVariantDir
		if ctx.DeviceConfig().BinderBitness() == "32" {
			dir = filepath.Join(dir, "binder32")
		}
		arch := "arch-" + target.Arch.ArchType.String()
		if target.Arch.ArchVariant != "" {
			arch += "-" + target.Arch.ArchVariant
		}
		dir = filepath.Join(dir, arch)
		archLibDir[target.Arch.ArchType] = dir
	}
	configsDir := filepath.Join(snapshotVariantDir, "configs")
	noticeDir := filepath.Join(snapshotVariantDir, "NOTICE_FILES")
	includeDir := filepath.Join(snapshotVariantDir, "include")
	noticeBuilt := make(map[string]bool)

	tryBuildNotice := func(m *Module, l snapshotLibraryInterface) {
		name := ctx.ModuleName(m) + ".txt"
		if l.shared() || l.static() {
			name = m.outputFile.Path().Base() + ".txt"
		}

		if _, ok := noticeBuilt[name]; ok {
			return
		}

		noticeBuilt[name] = true

		if m.NoticeFile().Valid() {
			outputs = append(outputs,
				installSnapshotFileFromPath(ctx, m.NoticeFile().Path(), snapshotDir, filepath.Join(noticeDir, name)))
		}
	}

	var generatedHeaders android.Paths
	includeDirs := make(map[string]bool)
	installedModulePaths := make(map[string]string)
	modulePaths := modulePaths(ctx.Config())

	var _ snapshotLibraryInterface = (*prebuiltLibraryLinker)(nil)
	var _ snapshotLibraryInterface = (*libraryDecorator)(nil)

	installSnapshotLib := func(m *Module, l snapshotLibraryInterface, dir string) bool {
		name := ctx.ModuleName(m)

		if l.shared() || l.static() {
			name = m.outputFile.Path().Base()
			libOut := filepath.Join(dir, name)
			outputs = append(outputs, installSnapshotFileFromPath(ctx, m.outputFile.Path(), snapshotDir, libOut))
		}

		jsonOut := filepath.Join(dir, name+".json")

		if includeBuildArtifacts {
			prop := struct {
				ExportedDirs        []string `json:",omitempty"`
				ExportedSystemDirs  []string `json:",omitempty"`
				ExportedFlags       []string `json:",omitempty"`
				RelativeInstallPath string   `json:",omitempty"`
			}{}
			prop.ExportedFlags = l.exportedFlags()
			prop.ExportedDirs = l.exportedDirs()
			prop.ExportedSystemDirs = l.exportedSystemDirs()
			prop.RelativeInstallPath = m.RelativeInstallPath()

			j, err := json.Marshal(prop)
			if err != nil {
				ctx.Errorf("json marshal to %q failed: %#v", jsonOut, err)
				return false
			}

			outputs = append(outputs, installSnapshotFileFromContent(ctx, string(j), snapshotDir, jsonOut))
		}

		tryBuildNotice(m, l)

		installedModulePaths[name] = modulePaths[ctx.ModuleName(m)]

		return true
	}

	ctx.VisitAllModules(func(module android.Module) {
		m, ok := module.(*Module)
		if !ok || !m.Enabled() {
			return
		}

		baseDir, ok := archLibDir[m.Target().Arch.ArchType]
		if !ok {
			return
		}

		l, libDir, ok := pred(ctx, m)
		if !ok {
			return
		}

		if !installSnapshotLib(m, l, filepath.Join(baseDir, libDir)) {
			return
		}

		generatedHeaders = append(generatedHeaders, l.exportedDeps()...)
		for _, dir := range append(l.exportedDirs(), l.exportedSystemDirs()...) {
			includeDirs[dir] = true
		}
	})

	if includeBuildArtifacts {
		headers := make(map[string]bool)

		for _, dir := range android.SortedStringKeys(includeDirs) {
			// workaround to determine if dir is under output directory
			if strings.HasPrefix(dir, android.PathForOutput(ctx).String()) {
				continue
			}
			exts := headerExts
			// Glob all files under this special directory, because of C++ headers.
			if strings.HasPrefix(dir, "external/libcxx/include") {
				exts = []string{""}
			}
			for _, ext := range exts {
				glob, err := ctx.GlobWithDeps(dir+"/**/*"+ext, nil)
				if err != nil {
					ctx.Errorf("%#v\n", err)
					return nil
				}
				for _, header := range glob {
					if strings.HasSuffix(header, "/") {
						continue
					}
					headers[header] = true
				}
			}
		}

		for _, header := range android.SortedStringKeys(headers) {
			outputs = append(outputs, installSnapshotFileFromPath(ctx, android.PathForSource(ctx, header), snapshotDir,
				filepath.Join(includeDir, header)))
		}

		isHeader := func(path string) bool {
			for _, ext := range headerExts {
				if strings.HasSuffix(path, ext) {
					return true
				}
			}
			return false
		}

		for _, path := range android.PathsToDirectorySortedPaths(android.FirstUniquePaths(generatedHeaders)) {
			header := path.String()

			if !isHeader(header) {
				continue
			}

			outputs = append(outputs, installSnapshotFileFromPath(ctx, path, snapshotDir, filepath.Join(includeDir, header)))
		}
	}

	var modulePathTxtBuilder strings.Builder

	first := true
	for _, lib := range android.SortedStringKeys(installedModulePaths) {
		if first {
			first = false
		} else {
			modulePathTxtBuilder.WriteString("\\n")
		}
		modulePathTxtBuilder.WriteString(lib)
		modulePathTxtBuilder.WriteString(" ")
		modulePathTxtBuilder.WriteString(installedModulePaths[lib])
	}

	outputs = append(outputs, installSnapshotFileFromContent(ctx, modulePathTxtBuilder.String(), snapshotDir,
		filepath.Join(configsDir, "module_paths.txt")))

	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].String() < outputs[j].String()
	})

	return outputs
}

func generateVndkSnapshot(ctx android.SingletonContext) {
	// BOARD_VNDK_VERSION must be set to 'current' in order to generate a VNDK snapshot.
	if ctx.DeviceConfig().VndkVersion() != "current" {
		return
	}

	if ctx.DeviceConfig().PlatformVndkVersion() == "" {
		return
	}

	if ctx.DeviceConfig().BoardVndkRuntimeDisable() {
		return
	}

	vndkCoreLibraries := vndkCoreLibraries(ctx.Config())
	vndkSpLibraries := vndkSpLibraries(ctx.Config())
	vndkPrivateLibraries := vndkPrivateLibraries(ctx.Config())
	llndkLibraries := llndkLibraries(ctx.Config())

	isVndkSnapshotLibrary := func(ctx android.SingletonContext, m *Module) (i snapshotLibraryInterface, libDir string, isSnapshotLib bool) {
		if m.Target().NativeBridge == android.NativeBridgeEnabled {
			return nil, "", false
		}
		if !m.useVndk() || !m.IsForPlatform() || !m.installable() {
			return nil, "", false
		}
		l, ok := m.linker.(snapshotLibraryInterface)
		if !ok || !l.shared() {
			return nil, "", false
		}
		name := ctx.ModuleName(m)
		if inList(name, *vndkCoreLibraries) {
			return l, filepath.Join("shared", "vndk-core"), true
		} else if inList(name, *vndkSpLibraries) {
			return l, filepath.Join("shared", "vndk-sp"), true
		} else {
			return nil, "", false
		}
	}

	outputs := android.Paths{
		installSnapshotFileFromContent(ctx, android.JoinWithSuffix(*vndkCoreLibraries, ".so", "\\n"),
			filepath.Join("vndk-snapshot", ctx.DeviceConfig().DeviceArch(), "configs"), "vndkcore.libraries.txt"),
		installSnapshotFileFromContent(ctx, android.JoinWithSuffix(*vndkPrivateLibraries, ".so", "\\n"),
			filepath.Join("vndk-snapshot", ctx.DeviceConfig().DeviceArch(), "configs"), "vndkprivate.libraries.txt"),
		installSnapshotFileFromContent(ctx, android.JoinWithSuffix(*vndkSpLibraries, ".so", "\\n"),
			filepath.Join("vndk-snapshot", ctx.DeviceConfig().DeviceArch(), "configs"), "vndksp.libraries.txt"),
		installSnapshotFileFromContent(ctx, android.JoinWithSuffix(*llndkLibraries, ".so", "\\n"),
			filepath.Join("vndk-snapshot", ctx.DeviceConfig().DeviceArch(), "configs"), "llndk.libraries.txt"),
	}
	for _, output := range generateSnapshotFiles(
		ctx,
		"vndk-snapshot",
		ctx.Config().VndkSnapshotBuildArtifacts(),
		isVndkSnapshotLibrary) {
		outputs = append(outputs, output)
	}
	outputList := android.PathForOutput(ctx, "snapshot", "vndk-"+ctx.Config().Getenv("TARGET_PRODUCT")+"_list")
	writePathsToFile(ctx, outputs, outputList)

	zip := android.PathForOutput(ctx, "snapshot", "vndk-"+ctx.Config().Getenv("TARGET_PRODUCT")+".zip")
	rule := android.NewRuleBuilder()
	rule.Command().
		BuiltTool(ctx, "soong_zip").
		FlagWithOutput("-o ", zip).
		FlagWithArg("-C ", android.PathForOutput(ctx, "vndk-snapshot").String()).
		FlagWithInput("-l ", outputList)

	rule.Build(pctx, ctx, zip.String(), "vndk snapshot "+zip.String())
	*vndkSnapshotZip(ctx.Config()) = zip
}

func generateVendorSnapshot(ctx android.SingletonContext) {
	isVendorSnapshotLibrary := func(ctx android.SingletonContext, m *Module) (i snapshotLibraryInterface, libDir string, isSnapshotLib bool) {
		dir := ctx.ModuleDir(m)
		for _, p := range []string{"vendor", "device", "cts", "external/llvm", "external/clang"} {
			if strings.HasPrefix(dir, p) {
				return nil, "", false
			}
		}
		if strings.HasPrefix(dir, "hardware/") {
			aosp := false
			for _, p := range []string{"interfaces", "libhardware", "libhardware_legacy", "ril"} {
				if strings.HasPrefix(dir, p) {
					aosp = true
					break
				}
			}
			if !aosp {
				return nil, "", false
			}
		}

		if m.Target().NativeBridge == android.NativeBridgeEnabled {
			return nil, "", false
		}
		if !m.useVndk() || !m.IsForPlatform() || !m.installable() {
			return nil, "", false
		}
		if m.sanitize != nil && !m.sanitize.isUnsanitizedVariant() {
			return nil, "", false
		}
		if _, ok := m.linker.(*vndkPrebuiltLibraryDecorator); ok {
			return nil, "", false
		}
		l, ok := m.linker.(snapshotLibraryInterface)
		if !ok {
			return nil, "", false
		}
		if l.static() {
			if m.VendorProperties.Vendor_available != nil && !*m.VendorProperties.Vendor_available {
				return nil, "", false
			}
			return l, "static", true
		}
		if l.shared() {
			if m.isVndk() {
				return nil, "", false
			}
			return l, "shared", true
		}
		return l, "header", true
	}

	var outputs android.Paths

	for _, output := range generateSnapshotFiles(ctx, "vendor-snapshot", true, isVendorSnapshotLibrary) {
		outputs = append(outputs, output)
	}

	outputList := android.PathForOutput(ctx, "snapshot", "vendor-"+ctx.Config().Getenv("TARGET_PRODUCT")+"_list")
	writePathsToFile(ctx, outputs, outputList)

	zip := android.PathForOutput(ctx, "snapshot", "vendor-"+ctx.Config().Getenv("TARGET_PRODUCT")+".zip")

	rule := android.NewRuleBuilder()
	rule.Command().
		BuiltTool(ctx, "soong_zip").
		FlagWithOutput("-o ", zip).
		FlagWithArg("-C ", android.PathForOutput(ctx, "vendor-snapshot").String()).
		FlagWithInput("-l ", outputList)

	rule.Build(pctx, ctx, zip.String(), "vendor snapshot "+zip.String())
	*vendorSnapshotZip(ctx.Config()) = zip
}

func (c *snapshotSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	generateVndkSnapshot(ctx)
	generateVendorSnapshot(ctx)
}
