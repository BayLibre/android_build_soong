package cc

import (
	"android/soong/android"
	"strings"
)

var (
	headerExts = []string{".h", ".hh", ".hpp", ".hxx", ".h++", ".inl", ".inc", ".ipp", ".h.generic"}
)

type snapshotLibraryInterface interface {
	exportedFlagsProducer
	libraryInterface
}

var _ snapshotLibraryInterface = (*prebuiltLibraryLinker)(nil)
var _ snapshotLibraryInterface = (*libraryDecorator)(nil)

func isHeader(path string) bool {
	for _, ext := range headerExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func exportedHeaders(ctx android.SingletonContext, l exportedFlagsProducer) android.Paths {
	var ret android.Paths

	for _, path := range append(l.exportedDirs(), l.exportedSystemDirs()...) {
		dir := path.String()
		// Skip if dir is for generated headers
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
				ret = append(ret, android.PathForSource(ctx, header))
			}
		}
	}

	for _, dep := range l.exportedDeps() {
		if !isHeader(dep.String()) {
			continue
		}
		ret = append(ret, dep)
	}

	return ret
}

func targetArchMap(config android.Config) map[android.ArchType]string {
	ret := make(map[android.ArchType]string)
	for _, target := range config.Targets[android.Android] {
		arch := "arch-" + target.Arch.ArchType.String()
		if target.Arch.ArchVariant != "" {
			arch += "-" + target.Arch.ArchVariant
		}
		ret[target.Arch.ArchType] = arch
	}
	return ret
}

func copyFile(ctx android.SingletonContext, path android.Path, out string) android.OutputPath {
	outPath := android.PathForOutput(ctx, out)
	ctx.Build(pctx, android.BuildParams{
		Rule:        android.Cp,
		Input:       path,
		Output:      outPath,
		Description: "Cp " + out,
		Args: map[string]string{
			"cpFlags": "-f -L",
		},
	})
	return outPath
}

func writeStringToFile(ctx android.SingletonContext, content, out string) android.OutputPath {
	outPath := android.PathForOutput(ctx, out)
	ctx.Build(pctx, android.BuildParams{
		Rule:        android.WriteFile,
		Output:      outPath,
		Description: "WriteFile " + out,
		Args: map[string]string{
			"content": content,
		},
	})
	return outPath
}
