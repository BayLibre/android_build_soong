package main

import (
    "fmt"
    "os"
    "path"
    "bufio"
    "strings"

    bpparser "github.com/google/blueprint/parser"
)

type androidMkWriter struct {
    commentsPos int
    defsPos int
    file *bpparser.File
    path string
    writer *bufio.Writer
}


func (w *androidMkWriter) errorf(format string, values ...interface{}) {
    s := fmt.Sprintf(format, values)
    w.writer.WriteString("# ANDROIDBP ERROR:\n" )
    for _, line := range(strings.Split(s, "\n")) {
        w.writer.WriteString(fmt.Sprintf("# %s\n", line))
    }
}

func (w *androidMkWriter) handleComment(comment *bpparser.Comment) {
    for _, c := range(comment.Comment) {
        mkComment := strings.Replace(c, "//", "#", 1)
        // TODO: handle /* comments?
        w.writer.WriteString(fmt.Sprintf("%s\n", mkComment))
    }
}

func (w *androidMkWriter) handleModule(module *bpparser.Module) {
    if moduleName, ok := moduleTypes[module.Type.Name]; ok {
        w.writer.WriteString("include $(CLEAR_VARS)\n")
        standardProps := make([]string, len(module.Properties))
        standardPropsIdx := 0
        //condProps := make([]string, len(module.Properties))
        //condPropsIdx := 0
        for _, prop := range(module.Properties) {

            switch prop.Value.Type {
            case bpparser.Bool:
                if mkProp, ok := boolProperties[prop.Name.Name]; ok {
                    if prop.Value.Variable == "" {
                        standardProps[standardPropsIdx] =
                                fmt.Sprintf("%s := %t\n", mkProp, prop.Value.BoolValue)
                    } else {
                        standardProps[standardPropsIdx] =
                                fmt.Sprintf("%s := $(%s)\n", mkProp, prop.Value.Variable)
                    }
                    standardPropsIdx++
                } else {
                    w.errorf("Unknown boolean property %s", prop.Name.Name)
                }
            case bpparser.String:
                if mkProp, ok := stringProperties[prop.Name.Name]; ok {
                    if prop.Value.Variable == "" {
                        standardProps[standardPropsIdx] =
                                fmt.Sprintf("%s := %s\n", mkProp, prop.Value.StringValue)
                    } else {
                        standardProps[standardPropsIdx] =
                                fmt.Sprintf("%s := $(%s)\n", mkProp, prop.Value.Variable)
                    }
                    standardPropsIdx++
                } else {
                    w.errorf("Unknown string property %s", prop.Name.Name)
                }
            case bpparser.List:
                if mkProp, ok := listProperties[prop.Name.Name]; ok {
                    if prop.Value.Variable == "" {
                        s := fmt.Sprintf("%s := \\\n", mkProp)
                        for _, tok := range(prop.Value.ListValue) {
                            switch tok.Type {
                            case bpparser.Bool:
                                s += fmt.Sprintf("\t\"%t\" \\\n", tok.BoolValue)
                            case bpparser.String:
                               s += fmt.Sprintf("\t\"%s\" \\\n", tok.StringValue)
                            default:
                                // TODO: support this recursively?
                                w.errorf("Unsupported type %s within list", tok.Type.String())
                            }
                        }
                        standardProps[standardPropsIdx] = s
                    } else {
                        standardProps[standardPropsIdx] =
                                fmt.Sprintf("%s := $(%s)\n", mkProp, prop.Value.Variable)
                    }
                    standardPropsIdx++
                } else {
                    w.errorf("Unknown string property %s", prop.Name.Name)
                }
            case bpparser.Map:
                w.errorf("Skipped map")
            }
        }

        for _, s:= range(standardProps) {
            w.writer.WriteString(s)
        }

        w.writer.WriteString(fmt.Sprintf("include $(%s)\n\n", moduleName))

    } else {
        w.errorf("Unsupported module %s", module.Type.Name)
    }
}

func (w *androidMkWriter) handleAssignment(assignment *bpparser.Assignment) {
    w.writer.WriteString(assignment.Name.Name)
    w.writer.WriteString(" := ")
    switch assignment.Value.Type {
    case bpparser.Bool:
        w.writer.WriteString(fmt.Sprintf("\"%t\"", assignment.Value.BoolValue))
    case bpparser.String:
        w.writer.WriteString(fmt.Sprintf("\"%s\"", assignment.Value.StringValue))
    case bpparser.List:
        w.writer.WriteString("\\\n")
        for _, tok := range(assignment.Value.ListValue) {
            switch tok.Type {
            case bpparser.Bool:
                w.writer.WriteString(fmt.Sprintf("\t\"%t\" \\\n", tok.BoolValue))
            case bpparser.String:
                w.writer.WriteString(fmt.Sprintf("\t\"%s\" \\\n", tok.StringValue))
            default:
                // TODO: support this recursively?
                w.errorf("Unsupported type %s within list", tok.Type.String())
            }
        }
    case bpparser.Map:
        w.errorf("maps not supported in assignment")
    }

    w.writer.WriteString("\n")
}

func (w *androidMkWriter) next() (definition interface{}) {
    if w.defsPos >= len(w.file.Defs) {
        if w.commentsPos >= len(w.file.Comments) {
            return nil
        }
        w.commentsPos++
        return w.file.Comments[w.commentsPos - 1]
    } else if w.commentsPos >= len(w.file.Comments) {
        w.defsPos++
        return w.file.Defs[w.defsPos - 1]
    }

    commentsPos := 0
    defsPos := 0

    def := w.file.Defs[w.defsPos]
    switch def := def.(type) {
    case *bpparser.Module:
        defsPos = def.LbracePos.Line
    case *bpparser.Assignment:
        defsPos = def.Pos.Line
    }

    comment := w.file.Comments[w.commentsPos]
    commentsPos = comment.Pos.Line

    if commentsPos < defsPos {
        w.commentsPos++
        return comment
    } else {
        w.defsPos++
        return def
    }
}

func (w *androidMkWriter) write() {
    outFilePath := fmt.Sprintf("%s/Android.mk.out", w.path)
    fmt.Printf("Writing %s\n", outFilePath)

    f, err := os.Create(outFilePath)
    if err != nil {
        panic(err)
    }

    defer func() {
        if err := f.Close(); err != nil {
            panic(err)
        }
    }()

    w.writer = bufio.NewWriter(f)

    block := w.next()
    for block != nil {
        switch block := block.(type) {
        case *bpparser.Module:
            w.handleModule(block)
        case *bpparser.Assignment:
            w.handleAssignment(block)
        case bpparser.Comment:
            w.handleComment(&block)
        }
        block = w.next()
    }

    if err = w.writer.Flush(); err != nil {
        panic(err)
    }
}

func main() {
    if (len(os.Args) < 2) {
        fmt.Println("No filename supplied");
        return;
    }

    reader, err := os.Open(os.Args[1])
    if err != nil {
        fmt.Println(err.Error())
        return
    }

    scope := bpparser.NewScope(nil)
    file, errs := bpparser.ParseAndEval(os.Args[1], reader, scope)
    if len(errs) > 0 {
        fmt.Println("%d errors parsing %s", len(errs), os.Args[1])
        fmt.Println(errs)
        return
    }

    writer := &androidMkWriter{
        file: file,
        path: path.Dir(os.Args[1]),
    }

    writer.write()
}

