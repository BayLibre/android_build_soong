package build

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"android/soong/ui/logger"
)

func TestDumpRBEMetrics(t *testing.T) {
	ctx := testContext()
	tests := []struct {
		description     string
		env             []string
		forceDumbOutput bool
		generated       bool
	}{{
		description: "RBE disabled",
		env: []string{
			"NOSTART_RBE=true",
		},
	}, {
		description: "rbe metrics generated",
		env: []string{
			"USE_RBE=true",
			"RBE_log_path=/some/fake_path",
		},
		generated: true,
	}, {
		description: "rbe metrics generated with force dump output",
		env: []string{
			"USE_RBE=true",
			"RBE_log_path=/some/fake_path",
		},
		forceDumbOutput: true,
		generated:       true,
	}}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			tmpDir := t.TempDir()

			rbeBootstrapCmd := filepath.Join(tmpDir, bootstrapCmd)
			if err := ioutil.WriteFile(rbeBootstrapCmd, []byte(rbeBootstrapProgram), 0755); err != nil {
				t.Fatalf("failed to create a fake bootstrap command file %s: %v", rbeBootstrapCmd, err)
			}

			rbeDumpStatsCmd := filepath.Join(tmpDir, dumpStatsCmd)
			if err := ioutil.WriteFile(rbeDumpStatsCmd, []byte(rbeDumpStatsProgram), 0755); err != nil {
				t.Fatalf("failed to create a fake dumpStats command bash script: %s: %v", rbeDumpStatsCmd, err)
			}

			env := Environment(tt.env)
			env.Set("OUT_DIR", tmpDir)
			env.Set("RBE_DIR", tmpDir)
			config := Config{&configImpl{
				environ: &env,
			}}

			rbeMetricsFilename := filepath.Join(tmpDir, rbeMetricsPBFname)
			DumpRBEMetrics(ctx, config, tt.forceDumbOutput, rbeMetricsFilename)

			// Validate that the rbe metrics file exists if RBE is enabled.
			if _, err := os.Stat(rbeMetricsFilename); err == nil {
				if !tt.generated {
					t.Errorf("got true, want false for rbe metrics file %s to exist.", rbeMetricsFilename)
				}
			} else if os.IsNotExist(err) {
				if tt.generated {
					t.Errorf("got false, want true for rbe metrics file %s to exist.", rbeMetricsFilename)
				}
			} else {
				t.Errorf("unknown error found on checking %s exists: %v", rbeMetricsFilename, err)
			}
		})
	}
}

func TestDumpRBEMetricsErrors(t *testing.T) {
	ctx := testContext()
	tests := []struct {
		description       string
		rbeLogPathDefined bool
		bootstrapProgram  string
		dumpStatsProgram  string
		expectedErr       string
	}{{
		description:      "RBE_log_path not defined",
		bootstrapProgram: rbeBootstrapProgram,
		expectedErr:      "rbe proxy log environment variable not defined",
	}, {
		description:       "stopRBE failed",
		rbeLogPathDefined: true,
		bootstrapProgram:  "#!/bin/bash\nexit 1",
		expectedErr:       "shutdown failed",
	}, {
		description:       "dumpStats failed",
		rbeLogPathDefined: true,
		bootstrapProgram:  rbeBootstrapProgram,
		dumpStatsProgram:  "#!/bin/bash\nexit 1",
		expectedErr:       "dump metrics failed",
	}, {
		description:       "failed to copy metrics file",
		rbeLogPathDefined: true,
		bootstrapProgram:  rbeBootstrapProgram,
		dumpStatsProgram:  "#!/bin/bash",
		expectedErr:       "failed to copy",
	}}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			defer logger.Recover(func(err error) {
				got := err.Error()
				if !strings.Contains(got, tt.expectedErr) {
					t.Errorf("got %q, want %q to be contained in error", got, tt.expectedErr)
				}
			})

			tmpDir := t.TempDir()

			rbeBootstrapCmd := filepath.Join(tmpDir, bootstrapCmd)
			if err := ioutil.WriteFile(rbeBootstrapCmd, []byte(tt.bootstrapProgram), 0755); err != nil {
				t.Fatalf("failed to create a fake bootstrap command file %s: %v", rbeBootstrapCmd, err)
			}

			rbeDumpStatsCmd := filepath.Join(tmpDir, dumpStatsCmd)
			if err := ioutil.WriteFile(rbeDumpStatsCmd, []byte(tt.dumpStatsProgram), 0755); err != nil {
				t.Fatalf("failed to create a fake dumpStats command bash script: %s: %v", rbeDumpStatsCmd, err)
			}

			env := &Environment{}
			env.Set("USE_RBE", "true")
			env.Set("OUT_DIR", tmpDir)
			env.Set("RBE_DIR", tmpDir)
			if tt.rbeLogPathDefined {
				env.Set("RBE_log_path", "/some/fake_path")
			}

			config := Config{&configImpl{
				environ: env,
			}}

			rbeMetricsFilename := filepath.Join(tmpDir, rbeMetricsPBFname)
			DumpRBEMetrics(ctx, config, false, rbeMetricsFilename)
			t.Errorf("got nil, expecting %q as a failure", tt.expectedErr)
		})
	}
}

var rbeBootstrapProgram = "#!/bin/bash"

var rbeDumpStatsProgram = fmt.Sprintf(`#!/bin/bash
while [[ $# -gt 0 ]]; do
  case "$1" in
    "-output_dir")
      echo 1 > "$2/%s"
      shift
      ;;
    *)
      shift
      ;;
  esac
done`, rbeMetricsPBFname)
