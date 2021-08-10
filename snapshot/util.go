// Copyright 2021 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package snapshot

import "android/soong/android"

func WriteStringToFileRule(ctx android.SingletonContext, content, out string) android.OutputPath {
	outPath := android.PathForOutput(ctx, out)
	android.WriteFileRule(ctx, outPath, content)
	return outPath
}

func CopyFileRule(pctx android.PackageContext, ctx android.SingletonContext, path android.Path, out string) android.OutputPath {
	outPath := android.PathForOutput(ctx, out)
	ctx.Build(pctx, android.BuildParams{
		Rule:        android.Cp,
		Input:       path,
		Output:      outPath,
		Description: "copy " + path.String() + " -> " + out,
		Args: map[string]string{
			"cpFlags": "-f -L",
		},
	})
	return outPath
}

// This is to be saved as .json files, which is for development/vendor_snapshot/update.py.
// These flags become Android.bp snapshot module properties.
type SnapshotJsonFlags struct {
	ModuleName          string `json:",omitempty"`
	RelativeInstallPath string `json:",omitempty"`
	Filename            string `json:",omitempty"`
	ModuleStemName      string `json:",omitempty"`

	// library flags
	ExportedDirs       []string `json:",omitempty"`
	ExportedSystemDirs []string `json:",omitempty"`
	ExportedFlags      []string `json:",omitempty"`
	Sanitize           string   `json:",omitempty"`
	SanitizeMinimalDep bool     `json:",omitempty"`
	SanitizeUbsanDep   bool     `json:",omitempty"`

	// binary flags
	Symlinks         []string `json:",omitempty"`
	StaticExecutable bool     `json:",omitempty"`

	// dependencies
	SharedLibs  []string `json:",omitempty"`
	StaticLibs  []string `json:",omitempty"`
	RuntimeLibs []string `json:",omitempty"`
	Required    []string `json:",omitempty"`

	// extra config files
	InitRc         []string `json:",omitempty"`
	VintfFragments []string `json:",omitempty"`
}

// zip snapshot
func zipSnapshot(ctx android.SingletonContext, dir string, baseName string, snapshotOutputs android.Paths) android.OptionalPath {
	zipPath := android.PathForOutput(
		ctx, dir, baseName+".zip")

	zipRule := android.NewRuleBuilder(pctx, ctx)
	rspFile := android.PathForOutput(
		ctx, dir, baseName+"_list.rsp")

	zipRule.Command().
		BuiltTool("soong_zip").
		FlagWithOutput("-o ", zipPath).
		FlagWithArg("-C ", android.PathForOutput(ctx, dir).String()).
		FlagWithRspFileInputList("-r ", rspFile, snapshotOutputs)

	zipRule.Build(zipPath.String(), baseName+" snapshot "+zipPath.String())
	return android.OptionalPathForPath(zipPath)
}

// Relative installation path
type RelativeInstallPath interface {
	RelativeInstallPath() string
}
