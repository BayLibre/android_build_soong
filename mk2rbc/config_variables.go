package mk2rbc

import (
	mkparser "android/soong/androidmk/parser"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// Extracts the list of product config variables from a file, calling
// given registrar for each variable.
func FindConfigVariables(mkFile string) error {
	mkContents, err := ioutil.ReadFile(mkFile)
	if err != nil {
		return err
	}
	parser := mkparser.NewParser(mkFile, bytes.NewBuffer(mkContents))
	nodes, errs := parser.Parse()
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "ERROR:", e)
		}
		return fmt.Errorf("cannot parse %s", mkFile)
	}
	for _, node := range nodes {
		asgn, ok := node.(*mkparser.Assignment)
		if !ok {
			continue
		}
		// We are looking for a variable called '_product_list_vars'
		// or '_product_single_value_vars'.
		if len(asgn.Name.Strings) > 1 {
			continue
		}
		varName := asgn.Name.Strings[0]
		var starType starType
		if varName == "_product_list_vars" {
			starType = starTypeList
		} else if varName == "_product_single_value_vars" {
			starType = starTypeUnknown
		} else {
			continue
		}
		for _, name := range strings.Fields(asgn.Value.Dump()) {
			KnownVariables.NewVariable(name, VarClassConfig, starType)
		}

	}
	return nil
}
