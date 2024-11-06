package find_input_delta_lib

import (
	"bytes"
	"slices"
	"testing"

	// For Assert*.
	"android/soong/android"
)

func (fl *FileList) Equal(other *FileList) bool {
	if fl.Name != other.Name {
		return false
	}
	if !slices.Equal(fl.Additions, other.Additions) {
		return false
	}
	if !slices.Equal(fl.Deletions, other.Deletions) {
		return false
	}
	if len(fl.Changes) != len(other.Changes) {
		return false
	}
	for idx, ch := range fl.Changes {
		if !ch.Equal(&other.Changes[idx]) {
			return false
		}
	}
	return true
}

func TestFormat(t *testing.T) {
	testCases := []struct {
		Name     string
		Template string
		Input    FileList
		Expected string
		Err      error
	}{
		{
			Name:     "no contents",
			Template: DefaultTemplate,
			Input: FileList{
				Name:      "target",
				Additions: []string{"add1", "add2"},
				Deletions: []string{"del1", "del2"},
				Changes: []FileList{
					FileList{Name: "mod1"},
					FileList{Name: "mod2"},
				},
			},
			Expected: "-del1 -del2 +add1 +add2 +mod1 +mod2 ",
			Err:      nil,
		},
		{
			Name:     "adds",
			Template: DefaultTemplate,
			Input: FileList{
				Name:      "target",
				Additions: []string{"add1", "add2"},
			},
			Expected: "+add1 +add2 ",
			Err:      nil,
		},
		{
			Name:     "deletes",
			Template: DefaultTemplate,
			Input: FileList{
				Name:      "target",
				Deletions: []string{"del1", "del2"},
			},
			Expected: "-del1 -del2 ",
			Err:      nil,
		},
		{
			Name:     "changes",
			Template: DefaultTemplate,
			Input: FileList{
				Name: "target",
				Changes: []FileList{
					FileList{Name: "mod1"},
					FileList{Name: "mod2"},
				},
			},
			Expected: "+mod1 +mod2 ",
			Err:      nil,
		},
		{
			Name:     "with contents",
			Template: DefaultTemplate,
			Input: FileList{
				Name:      "target",
				Additions: []string{"add1", "add2"},
				Deletions: []string{"del1", "del2"},
				Changes: []FileList{
					FileList{
						Name: "mod1",
					},
					FileList{
						Name:      "mod2",
						Additions: []string{"a1"},
						Deletions: []string{"d1"},
					},
				},
			},
			Expected: "-del1 -del2 +add1 +add2 +mod1 +mod2 --file mod2 -d1 +a1 --endfile ",
			Err:      nil,
		},
	}
	for _, tc := range testCases {
		buf := bytes.NewBuffer([]byte{})
		err := tc.Input.Format(buf, tc.Template)
		android.AssertSame(t, tc.Name, tc.Err, err)
		android.AssertSame(t, tc.Name, tc.Expected, buf.String())
	}
}
