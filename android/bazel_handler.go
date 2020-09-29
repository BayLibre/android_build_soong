package android

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type cqueryKey struct {
	label        string
	starlarkExpr string
}

type BazelContext struct {
	requests map[cqueryKey]bool   // cquery requests that have not yet been issued to Bazel
	results  map[cqueryKey]string // Results of cquery requests after Bazel invocations
}

func (context *BazelContext) Cquery(label string, starlarkExpr string) (string, bool) {
	key := cqueryKey{label, starlarkExpr}
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

func issueBazelCommand(vars BazelEnvVars, command string, labels []string,
	extraFlags ...string) (string, error) {

	cmdFlags := []string{"--output_base=" + vars.outputBase, command}
	cmdFlags = append(cmdFlags, labels...)
	cmdFlags = append(cmdFlags, extraFlags...)

	bazelCmd := exec.Command(vars.bazelPath, cmdFlags...)
	bazelCmd.Dir = vars.workspaceDir
	bazelCmd.Env = append(os.Environ(), "HOME="+vars.homeDir, pwdPrefix())

	var stderr bytes.Buffer
	bazelCmd.Stderr = &stderr

	if output, err := bazelCmd.Output(); err != nil {
		return "", fmt.Errorf("bazel command failed. command: [%s], error [%s]", bazelCmd, stderr)
	} else {
		return string(output), nil
	}
}

func (context *BazelContext) InvokeBazel(vars BazelEnvVars) error {
	context.results = map[cqueryKey]string{}

	var labels []string
	for val, _ := range context.requests {
		labels = append(labels, val.label)

		// TODO(cparsons): Combine requests into a batch cquery request.
		cqueryOutput, cqueryErr := issueBazelCommand(vars, "cquery", []string{val.label},
			"--output=starlark",
			"--starlark:expr="+val.starlarkExpr)

		if cqueryErr != nil {
			return cqueryErr
		} else {
			context.results[val] = string(cqueryOutput)
		}
	}

	// Issue a build command.
	// TODO(cparsons): Invoking bazel execution during soong_build should be avoided;
	// bazel actions should either be added to the Ninja file and executed later,
	// or bazel should handle execution.
	_, buildErr := issueBazelCommand(vars, "build", labels)

	if buildErr != nil {
		return buildErr
	}

	// Clear requests.
	context.requests = map[cqueryKey]bool{}
	return nil
}
