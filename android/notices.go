package android

import (
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
)

func init() {
	pctx.SourcePathVariable("mergeNotices", "build/soong/scripts/mergenotice.py")
	pctx.SourcePathVariable("generateNotice", "build/make/tools/generate-notice-files.py")
}

var (
	mergeNoticesRule = pctx.AndroidStaticRule("mergeNoticesRule", blueprint.RuleParams{
		Command:     `${mergeNotices} --output $out $inputs`,
		CommandDeps: []string{"${mergeNotices}"},
		Description: "merge notice files into $out",
	}, "inputs")

	generateNoticeRule = pctx.AndroidStaticRule("generateNoticeRule", blueprint.RuleParams{
		Command: `rm -rf $$(dirname $noticeHtml) $$(dirname $noticeText) && ` +
			`mkdir -p $$(dirname $noticeHtml) $$(dirname $noticeText) && ` +
			`${generateNotice} --text-output $noticeText --html-output $noticeHtml -t "$title" -s $srcDir`,
		CommandDeps: []string{"${generateNotice}"},
		Description: "generate HTML notice file $noticeHtml",
	}, "noticeHtml", "noticeText", "title", "srcDir")
)

func MergeNotices(ctx ModuleContext, mergedNotice WritablePath, noticePaths []Path) {
	noticePathStrings := make([]string, len(noticePaths))
	for i := 0; i < len(noticePaths); i++ {
		noticePathStrings[i] = noticePaths[i].String()
	}
	ctx.Build(pctx, BuildParams{
		Rule:   mergeNoticesRule,
		Inputs: noticePaths,
		Output: mergedNotice,
		Args: map[string]string{
			"inputs": strings.Join(noticePathStrings, " "),
		},
	})
}

func BuildNoticeHtml(
	ctx ModuleContext, installPath OutputPath, installFilename string, noticePaths []Path) ModuleOutPath {
	// Merge all NOTICE files into one.
	// TODO(jungjw): We should just produce a well-formatted NOTICE.html file in a single pass.
	//
	// generate-notice-files.py, which processes the merged NOTICE file, has somewhat strict rules
	// about input NOTICE file paths.
	// 1. Their relative paths to the src root become their NOTICE index titles. We want to use
	// on-device paths as titles, and so output the merged NOTICE file the corresponding location.
	// 2. They must end with .txt extension. Otherwise, they're ignored.
	noticeRelPath := InstallPathToOnDevicePath(ctx, installPath.Join(ctx, installFilename+".txt"))
	mergedNotice := PathForModuleOut(ctx, filepath.Join("NOTICE_FILES/src", noticeRelPath))
	MergeNotices(ctx, mergedNotice, noticePaths)

	// Transform the merged NOTICE file into an HTML file.
	noticeHtml := PathForModuleOut(ctx, "NOTICE", "NOTICE.html")
	// We don't use the text output, but generate-notice-files.py mandates it.
	noticeText := PathForModuleOut(ctx, "NOTICE_tmp", "NOTICE.txt")
	// Title is used only for the text output, and so doesn't really matter.
	noticeTitle := "Notices for " + ctx.ModuleName()
	ctx.Build(pctx, BuildParams{
		Rule:   generateNoticeRule,
		Input:  mergedNotice,
		Output: noticeHtml,
		Args: map[string]string{
			"noticeHtml": noticeHtml.String(),
			"noticeText": noticeText.String(),
			"title":      noticeTitle,
			"srcDir":     PathForModuleOut(ctx, "NOTICE_FILES/src").String(),
		},
	})

	return noticeHtml
}
