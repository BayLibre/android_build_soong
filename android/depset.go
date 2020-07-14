// Copyright 2020 Google Inc. All rights reserved.
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

// DepSet is designed to be conceptually compatible with Bazel's depsets:
// https://docs.bazel.build/versions/master/skylark/depsets.html

// A DepSet efficiently stores Paths from transitive dependencies without copying. It is stored
// as a DAG of DepSet nodes, each of which has some direct contents and a list of dependency
// DepSet nodes.
//
// A DepSet is created by NewDepSet from the Paths for direct contents and the *DepSets of
// dependencies. A DepSet is immutable once created. It can be converted to a list using
// DepSet.ToPostorderedList, DepSet.ToPreorderedList, DepSet.ToTopographicalList, or
// DepSet.ToSortedList.
type DepSet struct {
	direct     Paths
	transitive []*DepSet
}

// DepSetBuilder is used to create an immutable DepSet.
type DepSetBuilder struct {
	direct     Paths
	transitive []*DepSet
}

// NewDepSet returns an immutable DepSet with the given direct and transitive contents.
func NewDepSet(direct Paths, transitive []*DepSet) *DepSet {
	directCopy := make(Paths, len(direct))
	copy(directCopy, direct)

	transitiveCopy := make([]*DepSet, len(transitive))
	copy(transitiveCopy, transitive)

	return &DepSet{
		direct:     direct,
		transitive: transitive,
	}
}

// NewDepSetBuilder returns a DepSetBuilder to create an immutable DepSet.
func NewDepSetBuilder() *DepSetBuilder {
	return &DepSetBuilder{}
}

// Direct adds direct contents to the DepSet being built by a DepSetBuilder. Newly added direct
// contents are to the right of any existing direct contents.
func (b *DepSetBuilder) Direct(direct ...Path) *DepSetBuilder {
	b.direct = append(b.direct, direct...)
	return b
}

// Transitive adds transitive contents to the DepSet being built by a DepSetBuilder. Newly added
// transitive contents are to the right of any existing transitive contents.
func (b *DepSetBuilder) Transitive(transitive ...*DepSet) *DepSetBuilder {
	b.transitive = append(b.transitive, transitive...)
	return b
}

// Returns the DepSet being built by this DepSetBuilder.  The DepSetBuilder retains its contents
// for creating more DepSets.
func (b *DepSetBuilder) Build() *DepSet {
	return NewDepSet(b.direct, b.transitive)
}

// dfs calls the preOrder and postOrder methods if they are not nil in depth-first order on a
// DepSet. If leftToRight is set the dependencies are visited in left to right order, otherwise
// they are visited in right to left order.
func (d *DepSet) dfs(preOrder, postOrder func(Paths), leftToRight bool) {
	visited := make(map[*DepSet]bool)

	var dfs func(d *DepSet)
	dfs = func(d *DepSet) {
		visited[d] = true
		if preOrder != nil {
			preOrder(d.direct)
		}
		lenTransitive := len(d.transitive)
		for i := range d.transitive {
			var dep *DepSet
			if leftToRight {
				dep = d.transitive[i]
			} else {
				dep = d.transitive[lenTransitive-i-1]
			}
			if !visited[dep] {
				dfs(dep)
			}
		}

		if postOrder != nil {
			postOrder(d.direct)
		}
	}

	dfs(d)
}

// ToPostorderedList returns the direct and transitive contents of a DepSet in left to right
// postordered order. The list will contain children first, starting from the left, and then
// direct contents starting from the left.  Duplicate elements are removed, leaving only the first
// occurrence.
func (d *DepSet) ToPostorderedList() Paths {
	var list Paths
	d.dfs(nil,
		func(paths Paths) {
			list = append(list, paths...)
		},
		true)
	return FirstUniquePaths(list)
}

// ToPreorderedList returns the direct and transitive contents of a DepSet in left to right
// preordered order. The list will contain direct contents first, starting from the left, and
// then children starting from the left.  Duplicate elements are removed, leaving only the first
// occurrence.
func (d *DepSet) ToPreorderedList() Paths {
	var list Paths
	d.dfs(func(paths Paths) {
		list = append(list, paths...)
	}, nil,
		true)
	return FirstUniquePaths(list)
}

// ToTopologicalList returns the direct and transitive contents of a DepSet in topological order
// from the root down to the leaves. This is similar to ToPreorderedList, except that it
// guarantees that elements of children are listed after all of their parents.  Duplicate elements
// are removed, leaving only the first, but the existence of duplicates breaks the above ordering
// guarantee.
func (d *DepSet) ToTopologicalList() Paths {
	var list Paths
	d.dfs(nil,
		func(paths Paths) {
			list = append(list, ReversePaths(paths)...)
		},
		false)
	return ReversePaths(FirstUniquePaths(list))
}

// ToSortedList returns the direct and transitive contents of a DepSet in lexically sorted order
// with duplicates removed.
func (d *DepSet) ToSortedList() Paths {
	list := d.ToPostorderedList()
	return SortedUniquePaths(list)
}
