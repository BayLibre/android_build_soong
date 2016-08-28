package ninja

import (
        "fmt"
	"os"
        "strings"
        "syscall"
)

type File struct {
        workingDir string
        contents string

        // Kept around so that we can munmap and close the file
        fd *os.File
        mmap []byte
}

func Open(workingDir, filename string) (*File, error) {
        // TODO: currently assumes workingDir == $PWD
        file, err := os.Open(filename)
        if err != nil {
                return nil, err
        }
        st, err := file.Stat()
        if err != nil {
                file.Close()
                return nil, err
        }
        size := st.Size()
        if int64(int(size+4095)) != size+4095 {
                file.Close()
                return nil, fmt.Errorf("Size is too big for mmap")
        }
        if size == 0 {
                file.Close()
                return &File{workingDir: workingDir}, nil
        }
        data, err := syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
        if err != nil {
                file.Close()
                return nil, err
        }
        return &File{
                workingDir: workingDir,
                fd: file,
                mmap: data,
                contents: string(data[:size]),
        }, nil
}

func (f *File) Parse() error {
        rawlines := make(chan string)
        go parseRawLines(f.contents, rawlines)
        noComments := make(chan string)
        go parseOutComments(rawlines, noComments)
        logicalLines := make(chan logicalLine)
        go parseLogicalLines(noComments, logicalLines)
        blocks := make(chan block)
        go parseBlocks(logicalLines, blocks)

        //for l := range logicalLines {
        //        fmt.Printf("line: %q\n", l)
        //}

        return nil
}

// Divide up the file into lines split by \r\n or \n
func parseRawLines(data string, output chan<- string) {
        defer close(output)

        for len(data) > 0 {
                idx := strings.IndexByte(data, '\n')
                if idx == -1 {
                        output <- data
                        data = ""
                } else {
                        line := data[:idx]
                        data = data[idx+1:]
                        if len(line) > 0 && line[len(line)-1] == '\r' {
                                line = line[:len(line)-1]
                        }
                        output <- line
                }
        }
}

// Strip comments and any spaces before the comments
// Lines that had comments, but are now blank are omitted
func parseOutComments(input <-chan string, output chan<- string) {
        defer close(output)

        for l := range input {
                idx := strings.IndexByte(l, '#')
                if idx == -1 {
                        output <- l
                } else if idx != 0 {
                        l = strings.TrimRight(l[:idx], " ")
                        if len(l) > 0 {
                                output <- l
                        }
                }
        }
}

type logicalLine []string
func (l logicalLine) IsIndented() bool {
        for _, s := range l {
                for _, c := range s {
                        return c == ' '
                }
        }
        return false
}
func (l logicalLine) FirstWord(chars string) (word string, rest logicalLine) {
        var result string
        for _, s := range l {
                if len(s) == 0 {
                        continue
                }
                if idx := strings.IndexAny(s, chars); idx != -1 {
                        if idx != 0 {
                                result = result + s[:idx]
                        }
                        return result
                } else {
                        result = result + s
                }
        }
        return result
}
func (l logicalLine) TrimLeftSpaces() logicalLine {

}

// Create slices of logical lines after removing line continuations ($\n)
// These aren't returned as single strings, because that would trigger a copy
func parseLogicalLines(input <-chan string, output chan<- logicalLine) {
        defer close(output)

        var current []string
        for l := range input {
                if current != nil {
                        l = strings.TrimLeft(l, " ")
                }
                var count int
                for i := len(l)-1; i >= 0 && l[i] == '$'; i-- { count++ }
                if count % 2 == 1 {
                        if current == nil || len(l) > 1 {
                                current = append(current, l[:len(l)-1])
                        }
                } else {
                        // If a line only contains whitespace, make it empty
                        if current == nil && strings.TrimLeft(l, " ") == "" {
                                output <- nil
                        } else {
                                output <- append(current, l)
                        }
                        current = nil
                }
        }

        if current != nil {
                output <- current
        }
}

type block []logicalLine

func parseBlocks(input <-chan logicalLine, output chan<- block) {
        defer close(output)

        var current block
        for l := range input {
                if len(current) > 0 {
                        if len(l) == 0 {
                                output <- current
                                current = nil
                        } else if l.IsIndented() {
                                current = append(current, l)
                        } else {
                                output <- current
                                current = []logicalLine{l}
                        }
                } else {
                        if len(l) == 0 {
                                // Nothing
                        } else if l.IsIndented() {
                                // TODO: propagate error
                                fmt.Println("Error: invalid indent")
                        } else {
                                current = []logicalLine{l}
                        }
                }
        }
        if current != nil {
                output <- current
        }
}

type assignment struct {
        name string
        value []string
}

func parseAssignment(line logicalLine) (assignment) {
        var result assignment
        result.name = line.TrimLeftSpaces().FirstWord(" =")
        return result
}

func (f *File) Close() error {
        err := syscall.Munmap(f.mmap)
        if err != nil {
                return err
        }
        return f.fd.Close()
}
