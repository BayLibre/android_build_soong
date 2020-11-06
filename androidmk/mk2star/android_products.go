package mk2star

import (
	mkparser "android/soong/androidmk/parser"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type localDirEval struct {
	localDir  string
	hasErrors bool
}

func (l *localDirEval) Get(name string) string {
	if name == "LOCAL_DIR" {
		return l.localDir
	}
	l.hasErrors = true
	return fmt.Sprintf("$(%s)", name)
}

func (l *localDirEval) Set(_, _ string) {
}

func (l *localDirEval) Call(_ string, _ []string) []string {
	l.hasErrors = true
	return []string{"$(call ...)"}
}

func (l *localDirEval) SetFunc(_ string, _ func([]string) []string) {
}

// FindProductMakefiles returns the names of the  product makefiles listed in given
// file, which contains assignments to the PRODUCT_MAKEFILES variable. The value
// assigned to this variable is a list of items. Each item is a file path optionally
// preceded by "<PRODUCT>:" (see build/make/target/product/AndroidProducts.mk).
// A file name can reference $(LOCAL_DIR), which is the name of the input file
// directory (i.e., if the source tree root is TOP and the file passed to
// FindProductMakefiles is TOP/device/foo/AndroidProduct.mk,
// LOCAL_DIR will be TOP/device/foo
func FindProductMakefiles(androidProductMk string) ([]string, error) {
	contents, err := ioutil.ReadFile(androidProductMk)
	if err != nil {
		return nil, err
	}
	parser := mkparser.NewParser(androidProductMk, bytes.NewBuffer(contents))
	nodes, errs := parser.Parse()
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "ERROR:", e)
		}
		return nil, fmt.Errorf("cannot parse %s", androidProductMk)
	}

	var res []string
	ldEval := &localDirEval{localDir: filepath.Dir(androidProductMk)}
	for _, node := range nodes {
		asgn, ok := node.(*mkparser.Assignment)
		if !ok {
			continue
		}
		// We are looking for a variable called 'PRODUCT_MAKEFILES'
		if len(asgn.Name.Strings) > 1 || asgn.Name.Strings[0] != "PRODUCT_MAKEFILES" {
			continue
		}

		ldEval.hasErrors = false
		value := asgn.Value.Value(ldEval)
		if ldEval.hasErrors {
			return nil, fmt.Errorf("cannot evaluate %s", asgn.Value.Dump())
		}
		for _, token := range strings.Fields(value) {
			filename := token
			if n := strings.Index(token, ":"); n >= 0 {
				filename = token[n+1:]
			}

			res = append(res, filename)
		}
	}
	return res, nil
}
