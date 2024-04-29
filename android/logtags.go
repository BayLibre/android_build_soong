// Copyright 2024 Google Inc. All rights reserved.
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

package android

import "github.com/google/blueprint"

func init() {
	RegisterParallelSingletonType("logtags", LogtagsSingleton)
	pctx.SourcePathVariable("mergeLogtagsCmd", "build/make/tools/merge-event-log-tags.py")
	pctx.SourcePathVariable("logtagsLib", "build/make/tools/event_log_tags.py")
}

var (
	mergeLogtags = pctx.AndroidStaticRule("mergeLogtags",
		blueprint.RuleParams{
			Command:     "$mergeLogtagsCmd -o $out $in",
			CommandDeps: []string{"$mergeLogtagsCmd", "$logtagsLib"},
		})
)

type LogtagsInfo struct {
	Logtags Paths
}

var LogtagsProviderKey = blueprint.NewProvider[*LogtagsInfo]()

func LogtagsSingleton() Singleton {
	return &logtagsSingleton{}
}

type logtagsSingleton struct{}

func MergedLogtagsPath(ctx PathContext) OutputPath {
	return PathForIntermediates(ctx, "all-event-log-tags.txt")
}

func (l *logtagsSingleton) GenerateBuildActions(ctx SingletonContext) {
	var allLogtags Paths
	ctx.VisitAllModules(func(module Module) {
		if !module.ExportedToMake() {
			return
		}
		if logtagsInfo, ok := SingletonModuleProvider(ctx, module, LogtagsProviderKey); ok {
			allLogtags = append(allLogtags, logtagsInfo.Logtags...)
		}
	})

	ctx.Build(pctx, BuildParams{
		Rule:        mergeLogtags,
		Description: "merge logtags",
		Output:      MergedLogtagsPath(ctx),
		Inputs:      SortedUniquePaths(allLogtags),
	})
}
