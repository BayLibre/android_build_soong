package find_input_delta_lib

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	// For Assert*.
	"android/soong/android"

	fid_proto "android/soong/cmd/find_input_delta/find_input_delta_proto_internal"
	"google.golang.org/protobuf/proto"
)

// Various state files

func marshalProto(t *testing.T, message proto.Message) []byte {
	data, err := proto.Marshal(message)
	if err != nil {
		t.Errorf("%v", err)
	}
	return data
}

func protoFile(t *testing.T, name string, mtime_nsec int64, hash string, contents []*FileState) (pci *fid_proto.PartialCompileInput) {
	pci = &fid_proto.PartialCompileInput{
		Name: proto.String(name),
	}
	if mtime_nsec != 0 {
		pci.MtimeNsec = proto.Int64(mtime_nsec)
	}
	if len(hash) > 0 {
		pci.Hash = proto.String(hash)
	}
	if contents != nil {
		for _, c := range contents {
			cp := protoFile(t, c.Name, c.MtimeNsec, c.Hash, c.Contents)
			pci.Contents = append(pci.Contents, cp)
		}
	}
	return
}

func (s *InputState) Equal(other *InputState) bool {
	if len(s.Files) != len(other.Files) {
		return false
	}
	for idx, val := range s.Files {
		if !val.Equal(other.Files[idx]) {
			return false
		}
	}
	return true
}

func TestLoadState(t *testing.T) {
	testCases := []struct {
		Name     string
		Filename string
		Mapfs    fs.ReadFileFS
		Expected InputState
		Err      error
	}{
		{
			Name:     "missing file",
			Filename: "missing",
			Mapfs:    fstest.MapFS{},
			Expected: InputState{},
			Err:      nil,
		},
		{
			Name:     "bad file",
			Filename: ".",
			Mapfs:    OsFs,
			Expected: InputState{},
			Err:      errors.New("read failed"),
		},
		{
			Name:     "file with mtime",
			Filename: "state.old",
			Mapfs: fstest.MapFS{
				"state.old": &fstest.MapFile{
					Data: marshalProto(t, &fid_proto.PartialCompileInputs{
						InputFiles: []*fid_proto.PartialCompileInput{
							protoFile(t, "input1", 100, "", nil),
						},
					}),
				},
			},
			Expected: InputState{
				Files: []*FileState{
					&FileState{Name: "input1", MtimeNsec: 100},
				},
			},
			Err: nil,
		},
		{
			Name:     "file with mtime and hash",
			Filename: "state.old",
			Mapfs: fstest.MapFS{
				"state.old": &fstest.MapFile{
					Data: marshalProto(t, &fid_proto.PartialCompileInputs{
						InputFiles: []*fid_proto.PartialCompileInput{
							protoFile(t, "input1", 100, "crc:crc_value", nil),
						},
					}),
				},
			},
			Expected: InputState{
				Files: []*FileState{
					&FileState{
						Name:      "input1",
						MtimeNsec: 100, Hash: "crc:crc_value",
					},
				},
			},
			Err: nil,
		},
	}
	for _, tc := range testCases {
		actual, err := LoadState(tc.Filename, tc.Mapfs)
		if tc.Err == nil {
			android.AssertSame(t, tc.Name, tc.Err, err)
		} else if err == nil {
			t.Errorf("%s: expected error, did not get one", tc.Name)
		}
		if !tc.Expected.Equal(&actual) {
			t.Errorf("%s: expected %v, actual %v", tc.Name, tc.Expected, actual)
		}
	}
}

func TestCreateState(t *testing.T) {
	testCases := []struct {
		Name     string
		Inputs   []string
		Inspect  bool
		Mapfs    StatReadFileFS
		Expected InputState
		Err      error
	}{
		{
			Name:     "no inputs",
			Inputs:   []string{},
			Mapfs:    fstest.MapFS{},
			Expected: InputState{},
			Err:      nil,
		},
		{
			Name:   "files found",
			Inputs: []string{"baz", "foo", "bar"},
			Mapfs: fstest.MapFS{
				"foo": &fstest.MapFile{ModTime: time.Unix(0, 100).UTC()},
				"baz": &fstest.MapFile{ModTime: time.Unix(0, 300).UTC()},
				"bar": &fstest.MapFile{ModTime: time.Unix(0, 200).UTC()},
			},
			Expected: InputState{
				Files: []*FileState{
					// Files are always sorted.
					&FileState{Name: "bar", MtimeNsec: 200},
					&FileState{Name: "baz", MtimeNsec: 300},
					&FileState{Name: "foo", MtimeNsec: 100},
				},
			},
			Err: nil,
		},
	}
	for _, tc := range testCases {
		actual, err := CreateState(tc.Inputs, tc.Inspect, tc.Mapfs)
		if tc.Err == nil {
			android.AssertSame(t, tc.Name, tc.Err, err)
		} else if err == nil {
			t.Errorf("%s: expected error, did not get one", tc.Name)
		}
		if !tc.Expected.Equal(&actual) {
			t.Errorf("%s: expected %v, actual %v", tc.Name, tc.Expected, actual)
		}
	}
}

func TestInputStateCompare(t *testing.T) {
	testCases := []struct {
		Name     string
		Target   string
		Prior    InputState
		New      InputState
		Expected FileList
	}{
		{
			Name:   "prior is empty",
			Target: "foo",
			Prior:  InputState{},
			New: InputState{
				Files: []*FileState{
					&FileState{Name: "file1", MtimeNsec: 100},
				},
			},
			Expected: FileList{
				Name:      "foo",
				Additions: []string{"file1"},
			},
		},
		{
			Name:   "one of each",
			Target: "foo",
			Prior: InputState{
				Files: []*FileState{
					&FileState{Name: "file0", MtimeNsec: 100},
					&FileState{Name: "file1", MtimeNsec: 100},
					&FileState{Name: "file2", MtimeNsec: 200},
				},
			},
			New: InputState{
				Files: []*FileState{
					&FileState{Name: "file0", MtimeNsec: 100},
					&FileState{Name: "file1", MtimeNsec: 200},
					&FileState{Name: "file3", MtimeNsec: 300},
				},
			},
			Expected: FileList{
				Name:      "foo",
				Additions: []string{"file3"},
				Changes:   []FileList{FileList{Name: "file1"}},
				Deletions: []string{"file2"},
			},
		},
	}
	for _, tc := range testCases {
		actual := tc.Prior.Compare(tc.New, tc.Target)
		if !tc.Expected.Equal(&actual) {
			t.Errorf("%s: expected %q, actual %q", tc.Name, tc.Expected, actual)
		}
	}
}
