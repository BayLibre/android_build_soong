// Copyright 2017 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package python

import (
	"android/soong/android"
	"strings"

	"github.com/google/blueprint"
)

func init() {
	pctx.HostBinToolVariable("protocCmd", "aprotoc")
}

var (
	proto = pctx.AndroidStaticRule("protoc",
		blueprint.RuleParams{
			Command: `rm -rf $outDir && mkdir -p $outDir && ` +
				`$protocCmd --python_out=$outDir $protoFlags $in && ` +
				`$parCmd -o $out -P $pkgPath -C $outDir -D $outDir`,
			CommandDeps: []string{
				"$protocCmd",
				"$parCmd",
			},
		}, "protoFlags", "outDir", "pkgPath")
)

func genProto(ctx android.ModuleContext, outputSrcJar android.WritablePath,
	protoFiles android.Paths, protoFlags []string, pkgPath string) {

	ctx.Build(pctx, android.BuildParams{
		Rule:        proto,
		Description: "protoc " + protoFiles[0].Rel(),
		Output:      outputSrcJar,
		Inputs:      protoFiles,
		Args: map[string]string{
			"outDir":     android.ProtoDir(ctx).String(),
			"protoFlags": strings.Join(protoFlags, " "),
			"pkgPath":    pkgPath,
		},
	})
}
