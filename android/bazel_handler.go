package android

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type cqueryKey struct {
	label string
	starlarkExpr string
	arch ArchType
}

type BazelContext struct {
	requests map[cqueryKey]bool // cquery requests that have not yet been issued to Bazel
	results map[cqueryKey]string // Results of cquery requests after Bazel invocations
}

func (context *BazelContext) Cquery(label string, starlarkExpr string, arch ArchType) (string, bool) {
	key := cqueryKey{label, starlarkExpr, arch}
	if result, ok := context.results[key]; ok {
		return result, true
	} else {
		if context.requests == nil {
			context.requests = map[cqueryKey]bool{}
		}
		context.requests[key] = true
		return "", false
	}
}

func pwdPrefix() string {
	// Darwin doesn't have /proc
	if runtime.GOOS != "darwin" {
		return "PWD=/proc/self/cwd"
	}
	return ""
}

func platformLabel(arch ArchType) string {
	switch arch {
	case X86_64:
		return "//build/soong/bazel/devices:generic_x86"
	case Arm:
		return "//build/soong/bazel/devices:generic_arm"
	case Arm64:
		return "//build/soong/bazel/devices:generic_arm64"
	default:
		// TODO: Better error handling
		return "//build/soong/bazel/devices:generic_x86"
	}
}

func issueBazelCommand(vars BazelEnvVars, command string, label string,
		extraFlags ...string) (string, error) {

	cmdFlags := []string{"--output_base="+vars.outputBase, command, label}

	cmdFlags = append(cmdFlags, extraFlags...)

	bazelCmd := exec.Command(vars.bazelPath, cmdFlags...)
	bazelCmd.Dir = vars.workspaceDir
	bazelCmd.Env = append(os.Environ(), "HOME="+vars.homeDir, pwdPrefix())

	var stderr bytes.Buffer
	bazelCmd.Stderr = &stderr

	if output, err := bazelCmd.Output(); err != nil {
		return bazelCmd.String() + " " + stderr.String(), err
	} else {
		fmt.Println(bazelCmd.String())
		return string(output), nil
	}
}

func (context *BazelContext) InvokeBazel(vars BazelEnvVars) {
	context.results = map[cqueryKey]string{}

	for val, _ := range context.requests {
		platformFlag := "--platforms=" + platformLabel(val.arch)

		// Issue a build command for now.
		// TODO(cparsons): Invoking bazel execution during soong_build should be avoided;
		// bazel actions should either be added to the Ninja file and executed later,
		// or bazel should handle execution.
		buildOutput, buildErr := issueBazelCommand(vars,"build", val.label, platformFlag)

		if buildErr != nil {
			fmt.Println("error invoking bazel build: ", buildOutput)
			return
		}

		cqueryOutput, cqueryErr := issueBazelCommand(vars, "cquery", val.label,
			platformFlag,
			"--output=starlark",
			"--starlark:expr=" + val.starlarkExpr)

		if cqueryErr != nil {
			// TODO: Better error handling.
			fmt.Println("error invoking bazel cquery: ", cqueryOutput)
			return
		} else {
			context.results[val] = string(cqueryOutput)
		}
		// Clear requests.
		context.requests = map[cqueryKey]bool{}
	}
}