package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"android/soong/mk2rbc"
)

var (
	rootDir = flag.String("root", ".", "the value of // for load paths")
)

type variable struct {
	Name      string
	Value     string
	Class     mk2rbc.VarClass
	ValueType mk2rbc.StarlarkType
}

func printVars(vars []variable) {
	pat := regexp.MustCompile(`(\S+)`)
	for _, newVar := range vars {
		if newVar.ValueType == mk2rbc.StarlarkTypeList {
			fmt.Printf("  \"%s\": [", newVar.Name)
			for _, match := range pat.FindAllStringSubmatch(newVar.Value, -1) {
				fmt.Printf("\"%s\", ", match[1])
			}
			fmt.Printf("],\n")
		} else {
			fmt.Printf("  \"%s\": \"%s\",\n", newVar.Name, newVar.Value)
		}
	}
}

func main() {
	flag.Usage = func() {
		cmd := filepath.Base(os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(),
			"Usage: %[1]s [-root]\n", cmd)
		fmt.Fprintf(flag.CommandLine.Output(), `
mkvars2rbc is a simpler version of mk2rbc. It only accepts a list of
make variables on its stdin, and produces a file containing two starlark
dictionaries (cfg and globals) on its stdout.

The input must consist of only lines of the form MAKE_VARIABLE := value,
empty lines, or comments starting with #.

`)
		flag.PrintDefaults()
	}
	flag.Parse()

	knownVariables, err := mk2rbc.CreateKnownVariables(*rootDir)
	if err != nil {
		quit("%s\n", err.Error())
	}

	scanner := bufio.NewScanner(os.Stdin)

	globals := make([]variable, 0)
	configs := make([]variable, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":=", 2)

		if len(parts) < 2 {
			quit("Unknown syntax: %s\n", line)
		}

		name := strings.TrimSpace(parts[0])
		newVar := variable{
			Name:      name,
			Value:     strings.TrimSpace(parts[1]),
			Class:     mk2rbc.VarClassSoong,
			ValueType: mk2rbc.StarlarkTypeUnknown,
		}

		if t, ok := knownVariables[name]; ok {
			newVar.Class = t.Class
			newVar.ValueType = t.ValueType
		}

		if newVar.Class == mk2rbc.VarClassConfig {
			configs = append(configs, newVar)
		} else {
			globals = append(globals, newVar)
		}
	}

	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Name < configs[j].Name
	})
	sort.Slice(globals, func(i, j int) bool {
		return globals[i].Name < globals[j].Name
	})

	fmt.Println("cfg = {")
	printVars(configs)
	fmt.Println("}\n\nglobals = {")
	printVars(globals)
	fmt.Println("}")
}

func quit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
