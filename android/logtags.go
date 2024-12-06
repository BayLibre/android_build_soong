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

import (
	"github.com/google/blueprint"
	"github.com/google/blueprint/depset"
)

type LogtagsInfo struct {
	Logtags depset.DepSet[Path]
}

var LogtagsProviderKey = blueprint.NewProvider[*LogtagsInfo]()

// CreateLogtagsDepset creates a depset of the given logtags + the logtags of any direct deps
// whose dependency tag matches the predicate.
func CreateLogtagsDepset(ctx ModuleContext, logtags Paths, tagPredicate func(blueprint.DependencyTag) bool) depset.DepSet[Path] {
	var allDepLogtags []depset.DepSet[Path]
	ctx.VisitDirectDeps(func(m Module) {
		tag := ctx.OtherModuleDependencyTag(m)
		if tagPredicate(tag) {
			depLogTags, ok := OtherModuleProvider(ctx, m, LogtagsProviderKey)
			if ok {
				allDepLogtags = append(allDepLogtags, depLogTags.Logtags)
			}
		}
	})

	return depset.New(
		depset.PREORDER,
		logtags,
		allDepLogtags,
	)
}
