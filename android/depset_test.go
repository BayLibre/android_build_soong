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

import (
	"fmt"
	"reflect"
	"testing"
)

func ExampleDepSet_ToPostorderedList() {
	a := NewDepSetBuilder().Direct(PathForTesting("a")).Build()
	b := NewDepSetBuilder().Direct(PathForTesting("b")).Transitive(a).Build()
	c := NewDepSetBuilder().Direct(PathForTesting("c")).Transitive(a).Build()
	d := NewDepSetBuilder().Direct(PathForTesting("d")).Transitive(b, c).Build()

	fmt.Println(d.ToPostorderedList().Strings())
	// Output: [a b c d]
}

func ExampleDepSet_ToPreorderedList() {
	a := NewDepSetBuilder().Direct(PathForTesting("a")).Build()
	b := NewDepSetBuilder().Direct(PathForTesting("b")).Transitive(a).Build()
	c := NewDepSetBuilder().Direct(PathForTesting("c")).Transitive(a).Build()
	d := NewDepSetBuilder().Direct(PathForTesting("d")).Transitive(b, c).Build()

	fmt.Println(d.ToPreorderedList().Strings())
	// Output: [d b a c]
}

func ExampleDepSet_ToTopologicalList() {
	a := NewDepSetBuilder().Direct(PathForTesting("a")).Build()
	b := NewDepSetBuilder().Direct(PathForTesting("b")).Transitive(a).Build()
	c := NewDepSetBuilder().Direct(PathForTesting("c")).Transitive(a).Build()
	d := NewDepSetBuilder().Direct(PathForTesting("d")).Transitive(b, c).Build()

	fmt.Println(d.ToTopologicalList().Strings())
	// Output: [d b c a]
}

func ExampleDepSet_ToSortedUniqueList() {
	a := NewDepSetBuilder().Direct(PathForTesting("a")).Build()
	b := NewDepSetBuilder().Direct(PathForTesting("b")).Transitive(a).Build()
	c := NewDepSetBuilder().Direct(PathForTesting("c")).Transitive(a).Build()
	d := NewDepSetBuilder().Direct(PathForTesting("d")).Transitive(b, c).Build()

	fmt.Println(d.ToSortedList().Strings())
	// Output: [a b c d]
}

// Tests based on Bazel's ExpanderTestBase.java to ensure compatibility
// https://github.com/bazelbuild/bazel/blob/master/src/test/java/com/google/devtools/build/lib/collect/nestedset/ExpanderTestBase.java
func TestDepSet(t *testing.T) {
	a := PathForTesting("a")
	b := PathForTesting("b")
	c := PathForTesting("c")
	c2 := PathForTesting("c2")
	d := PathForTesting("d")
	e := PathForTesting("e")

	tests := []struct {
		name                             string
		depSet                           func(t *testing.T) *DepSet
		postorder, preorder, topological []string
	}{
		{
			name: "simple",
			depSet: func(t *testing.T) *DepSet {
				return NewDepSet(Paths{c, a, b}, nil)
			},
			postorder:   []string{"c", "a", "b"},
			preorder:    []string{"c", "a", "b"},
			topological: []string{"c", "a", "b"},
		},
		{
			name: "simpleNoDuplicates",
			depSet: func(t *testing.T) *DepSet {
				return NewDepSet(Paths{c, a, a, a, b}, nil)
			},
			postorder:   []string{"c", "a", "b"},
			preorder:    []string{"c", "a", "b"},
			topological: []string{"c", "a", "b"},
		},
		{
			name: "nesting",
			depSet: func(t *testing.T) *DepSet {
				subset := NewDepSet(Paths{c, a, e}, nil)
				return NewDepSet(Paths{b, d}, []*DepSet{subset})
			},
			postorder:   []string{"c", "a", "e", "b", "d"},
			preorder:    []string{"b", "d", "c", "a", "e"},
			topological: []string{"b", "d", "c", "a", "e"},
		},
		{
			name: "builderReuse",
			depSet: func(t *testing.T) *DepSet {
				assertEquals := func(t *testing.T, w, g Paths) {
					if !reflect.DeepEqual(w, g) {
						t.Errorf("want %q, got %q", w, g)
					}
				}
				builder := NewDepSetBuilder()
				assertEquals(t, nil, builder.Build().ToPostorderedList())

				builder.Direct(b)
				assertEquals(t, Paths{b}, builder.Build().ToPostorderedList())

				builder.Direct(d)
				assertEquals(t, Paths{b, d}, builder.Build().ToPostorderedList())

				child := NewDepSetBuilder().Direct(c, a, e).Build()
				builder.Transitive(child)
				return builder.Build()
			},
			postorder:   []string{"c", "a", "e", "b", "d"},
			preorder:    []string{"b", "d", "c", "a", "e"},
			topological: []string{"b", "d", "c", "a", "e"},
		},
		{
			name: "builderChaining",
			depSet: func(t *testing.T) *DepSet {
				return NewDepSetBuilder().Direct(b).Direct(d).
					Transitive(NewDepSetBuilder().Direct(c, a, e).Build()).Build()
			},
			postorder:   []string{"c", "a", "e", "b", "d"},
			preorder:    []string{"b", "d", "c", "a", "e"},
			topological: []string{"b", "d", "c", "a", "e"},
		},
		{
			name: "transitiveDepsHandledSeparately",
			depSet: func(t *testing.T) *DepSet {
				subset := NewDepSetBuilder().Direct(c, a, e).Build()
				builder := NewDepSetBuilder()
				// The fact that we add the transitive subset between the Direct(b) and Direct(d)
				// calls should not change the result.
				builder.Direct(b)
				builder.Transitive(subset)
				builder.Direct(d)
				return builder.Build()
			},
			postorder:   []string{"c", "a", "e", "b", "d"},
			preorder:    []string{"b", "d", "c", "a", "e"},
			topological: []string{"b", "d", "c", "a", "e"},
		},
		{
			name: "nestingNoDuplicates",
			depSet: func(t *testing.T) *DepSet {
				subset := NewDepSetBuilder().Direct(c, a, e).Build()
				return NewDepSetBuilder().Direct(b, d, e).Transitive(subset).Build()
			},
			postorder:   []string{"c", "a", "e", "b", "d"},
			preorder:    []string{"b", "d", "e", "c", "a"},
			topological: []string{"b", "d", "c", "a", "e"},
		},
		{
			name: "chain",
			depSet: func(t *testing.T) *DepSet {
				c := NewDepSetBuilder().Direct(c).Build()
				b := NewDepSetBuilder().Direct(b).Transitive(c).Build()
				a := NewDepSetBuilder().Direct(a).Transitive(b).Build()

				return a
			},
			postorder:   []string{"c", "b", "a"},
			preorder:    []string{"a", "b", "c"},
			topological: []string{"a", "b", "c"},
		},
		{
			name: "diamond",
			depSet: func(t *testing.T) *DepSet {
				d := NewDepSetBuilder().Direct(d).Build()
				c := NewDepSetBuilder().Direct(c).Transitive(d).Build()
				b := NewDepSetBuilder().Direct(b).Transitive(d).Build()
				a := NewDepSetBuilder().Direct(a).Transitive(b).Transitive(c).Build()

				return a
			},
			postorder:   []string{"d", "b", "c", "a"},
			preorder:    []string{"a", "b", "d", "c"},
			topological: []string{"a", "b", "c", "d"},
		},
		{
			name: "extendedDiamond",
			depSet: func(t *testing.T) *DepSet {
				d := NewDepSetBuilder().Direct(d).Build()
				e := NewDepSetBuilder().Direct(e).Build()
				b := NewDepSetBuilder().Direct(b).Transitive(d).Transitive(e).Build()
				c := NewDepSetBuilder().Direct(c).Transitive(e).Transitive(d).Build()
				a := NewDepSetBuilder().Direct(a).Transitive(b).Transitive(c).Build()
				return a
			},
			postorder:   []string{"d", "e", "b", "c", "a"},
			preorder:    []string{"a", "b", "d", "e", "c"},
			topological: []string{"a", "b", "c", "e", "d"},
		},
		{
			name: "extendedDiamondRightArm",
			depSet: func(t *testing.T) *DepSet {
				d := NewDepSetBuilder().Direct(d).Build()
				e := NewDepSetBuilder().Direct(e).Build()
				b := NewDepSetBuilder().Direct(b).Transitive(d).Transitive(e).Build()
				c2 := NewDepSetBuilder().Direct(c2).Transitive(e).Transitive(d).Build()
				c := NewDepSetBuilder().Direct(c).Transitive(c2).Build()
				a := NewDepSetBuilder().Direct(a).Transitive(b).Transitive(c).Build()
				return a
			},
			postorder:   []string{"d", "e", "b", "c2", "c", "a"},
			preorder:    []string{"a", "b", "d", "e", "c", "c2"},
			topological: []string{"a", "b", "c", "c2", "e", "d"},
		},
		{
			name: "orderConflict",
			depSet: func(t *testing.T) *DepSet {
				child1 := NewDepSetBuilder().Direct(a, b).Build()
				child2 := NewDepSetBuilder().Direct(b, a).Build()
				parent := NewDepSetBuilder().Transitive(child1).Transitive(child2).Build()
				return parent
			},
			postorder:   []string{"a", "b"},
			preorder:    []string{"a", "b"},
			topological: []string{"b", "a"},
		},
		{
			name: "orderConflictNested",
			depSet: func(t *testing.T) *DepSet {
				a := NewDepSetBuilder().Direct(a).Build()
				b := NewDepSetBuilder().Direct(b).Build()
				child1 := NewDepSetBuilder().Transitive(a).Transitive(b).Build()
				child2 := NewDepSetBuilder().Transitive(b).Transitive(a).Build()
				parent := NewDepSetBuilder().Transitive(child1).Transitive(child2).Build()
				return parent
			},
			postorder:   []string{"a", "b"},
			preorder:    []string{"a", "b"},
			topological: []string{"b", "a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depSet := tt.depSet(t)

			t.Run("postorder", func(t *testing.T) {
				if g, w := depSet.ToPostorderedList().Strings(), tt.postorder; !reflect.DeepEqual(g, w) {
					t.Errorf("expected ToPostorderedList() = %q, got %q", w, g)
				}
			})
			t.Run("preorder", func(t *testing.T) {
				if g, w := depSet.ToPreorderedList().Strings(), tt.preorder; !reflect.DeepEqual(g, w) {
					t.Errorf("expected ToPreorderedList() = %q, got %q", w, g)
				}
			})
			t.Run("topological", func(t *testing.T) {
				if g, w := depSet.ToTopologicalList().Strings(), tt.topological; !reflect.DeepEqual(g, w) {
					t.Errorf("expected ToTopologicalList() = %q, got %q", w, g)
				}
			})
		})
	}
}
