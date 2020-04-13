package remoteexec

import (
	"fmt"
	"testing"
)

func TestTemplate(t *testing.T) {
	tests := []struct {
		name   string
		params *REParams
		want   string
	}{
		{
			name: "basic",
			params: &REParams{
				Labels:      map[string]string{"type": "compile", "lang": "cpp", "compiler": "clang"},
				Inputs:      []string{"$in"},
				OutputFiles: []string{"$out"},
				Platform: map[string]string{
					ContainerImageKey: DefaultImage,
					PoolKey:           "default",
				},
			},
			want: fmt.Sprintf("${config.REWrapper} --labels=compiler=clang,lang=cpp,type=compile --platform=\"Pool=default,container-image=%s\" --exec_strategy=local --inputs=$in --output_files=$out -- ", DefaultImage),
		},
		{
			name: "all params",
			params: &REParams{
				Labels:          map[string]string{"type": "compile", "lang": "cpp", "compiler": "clang"},
				Inputs:          []string{"$in"},
				OutputFiles:     []string{"$out"},
				ExecStrategy:    "remote",
				RSPFile:         "$out.rsp",
				ToolchainInputs: []string{"clang++"},
				Platform: map[string]string{
					ContainerImageKey: DefaultImage,
					PoolKey:           "default",
				},
			},
			want: fmt.Sprintf("${config.REWrapper} --labels=compiler=clang,lang=cpp,type=compile --platform=\"Pool=default,container-image=%s\" --exec_strategy=remote --inputs=$in --input_list_paths=$out.rsp --output_files=$out --toolchain_inputs=clang++ -- ", DefaultImage),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.params.Template(); got != test.want {
				t.Errorf("Template() returned\n%s\nwant\n%s", got, test.want)
			}
		})
	}
}

func TestTemplateDeterminism(t *testing.T) {
	r := &REParams{
		Labels:      map[string]string{"type": "compile", "lang": "cpp", "compiler": "clang"},
		Inputs:      []string{"$in"},
		OutputFiles: []string{"$out"},
		Platform: map[string]string{
			ContainerImageKey: DefaultImage,
			PoolKey:           "default",
		},
	}
	want := fmt.Sprintf("${config.REWrapper} --labels=compiler=clang,lang=cpp,type=compile --platform=\"Pool=default,container-image=%s\" --exec_strategy=local --inputs=$in --output_files=$out -- ", DefaultImage)
	for i := 0; i < 1000; i++ {
		if got := r.Template(); got != want {
			t.Fatalf("Template() returned\n%s\nwant\n%s", got, want)
		}
	}
}
