package ninja

import (
        "fmt"
        "path/filepath"
        "runtime"
        "testing"
)

func assert(t *testing.T, actual, expected interface{}) {
        if fmt.Sprintf("%q", actual) != fmt.Sprintf("%q", expected) {
                _, file, line, ok := runtime.Caller(1)
                if !ok {
                        file = "???"
                        line = 0
                } else {
                        file = filepath.Base(file)
                }
                t.Errorf("%s:%d: Got %q, expected %q", file, line, actual, expected)
        }
}

func TestOpen(t *testing.T) {
        f, err := Open(".", "test.ninja")
        if err != nil {
                t.Fatal(err)
        }
        err = f.Parse()
        if err != nil {
                t.Error(err)
        }
        err = f.Close()
        if err != nil {
                t.Fatal(err)
        }
}

func TestParseRawLines(t *testing.T) {
        input := "a\nb\r\nc \nd"
        output := make(chan string, 100)
        parseRawLines(input, output)
        actual := []string{<-output, <-output, <-output, <-output}
        expected := []string{"a", "b", "c ", "d"}
        assert(t, actual, expected)
        if _, ok := <-output; ok {
                t.Errorf("Channel not closed")
        }
}

func TestParseOutComments(t *testing.T) {
        input := make(chan string, 4)
        input <- "# a"
        input <- " #b"
        input <- ""
        input <- "c #d"
        close(input)
        output := make(chan string, 100)
        parseOutComments(input, output)
        actual := []string{<-output, <-output}
        expected := []string{"", "c"}
        assert(t, actual, expected)
        if _, ok := <-output; ok {
                t.Errorf("Channel not closed")
        }
}

func TestParseLogicalLines(t *testing.T) {
        input := make(chan string, 8)
        input <- "a"
        input <- "b $"
        input <- " c$$$"
        input <- "  d$ "
        input <- "e$$"
        input <- " "
        input <- "$"
        input <- "$"
        close(input)
        output := make(chan logicalLine, 100)
        parseLogicalLines(input, output)
        assert(t, <-output, []string{"a"})
        assert(t, <-output, []string{"b ", "c$$", "d$ "})
        assert(t, <-output, []string{"e$$"})
        assert(t, <-output, []string{})
        assert(t, <-output, []string{""})
        if _, ok := <-output; ok {
                t.Errorf("Channel not closed")
        }
}

func TestParseBlocks(t *testing.T) {
        input := make(chan logicalLine, 5)
        input <- []string{"builddir = ", "a"}
        input <- []string{"rule abc", "d"}
        input <- []string{" test ", "= a"}
        input <- []string{}
        input <- []string{"rule d"}
        close(input)
        output := make(chan block, 100)
        parseBlocks(input, output)
        assert(t, <-output, block{logicalLine{"builddir = ", "a"}})
        assert(t, <-output, block{
                logicalLine{"rule abc", "d"},
                logicalLine{" test ", "= a"},
        })
        assert(t, <-output, block{logicalLine{"rule d"}})
        if _, ok := <-output; ok {
                t.Errorf("Channel not closed")
        }
}

func TestLogicalLine(t *testing.T) {
        assert(t, logicalLine{}.IsIndented(), false)
        assert(t, logicalLine{""}.IsIndented(), false)
        assert(t, logicalLine{"","a"}.IsIndented(), false)
        assert(t, logicalLine{" a"}.IsIndented(), true)
        assert(t, logicalLine{" ","a"}.IsIndented(), true)
        assert(t, logicalLine{""," a"}.IsIndented(), true)

        assert(t, logicalLine{}.FirstWord(" "), "")
        assert(t, logicalLine{""}.FirstWord(" "), "")
        assert(t, logicalLine{"a"}.FirstWord(" "), "a")
        assert(t, logicalLine{"","a"}.FirstWord(" "), "a")
        assert(t, logicalLine{"","a b"}.FirstWord(" "), "a")
        assert(t, logicalLine{"","a ","b"}.FirstWord(" "), "a")
        assert(t, logicalLine{"","a"," b"}.FirstWord(" "), "a")
        assert(t, logicalLine{"","a","b c"}.FirstWord(" "), "ab")
}

func TestParseAssignment(t *testing.T) {
        assert(t,
                parseAssignment(logicalLine{"builddir =  out"}),
                assignment{
                        name: "builddir",
                        value: []string{"out"},
                })
        assert(t,
                parseAssignment(logicalLine{" builddir =  out"}),
                assignment{
                        name: "builddir",
                        value: []string{"out"},
                })
        assert(t,
                parseAssignment(logicalLine{" ", "builddir =  out"}),
                assignment{
                        name: "builddir",
                        value: []string{"out"},
                })
        assert(t,
                parseAssignment(logicalLine{" ", " builddir =  out"}),
                assignment{
                        name: "builddir",
                        value: []string{"out"},
                })
        assert(t,
                parseAssignment(logicalLine{" ", "", " b", "uilddir =  out"}),
                assignment{
                        name: "builddir",
                        value: []string{"out"},
                })
}
