package mk2rbc

import (
	"io/fs"
	"reflect"
	"testing"
)

func Test_find(t *testing.T) {
	type args struct {
		fsys   fs.FS
		chunks []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "simple",
			args: args{
				fsys: NewFindMockFS([]string{
					"1/1/1", "1/1/2", "1/2", "1/3/1", "1/3/2/1", "1/3/2/2",
				}),
				chunks: []string{"1", "2"},
			},
			want: []string{
				"1/1/2", "1/2", "1/3/2/2",
			},
		},
		{
			name: "hidden dirs",
			args: args{
				fsys: NewFindMockFS([]string{
					"1/1/1", "1/.git/2", "1/2", "1/3/1", "1/3/2/1", "1/3/2/2",
				}),
				chunks: []string{"", "2", "2"},
			},
			want: []string{
				"1/3/2/2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findMatchingPaths(tt.args.fsys, tt.args.chunks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("find() = %v, want %v", got, tt.want)
			}
		})
	}
}
