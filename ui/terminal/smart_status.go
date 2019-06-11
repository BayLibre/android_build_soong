// Copyright 2019 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package terminal

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"android/soong/ui/status"
)

type statusTableEntry struct {
	action    *status.Action
	startTime time.Time
}

type smartStatusOutput struct {
	writer    io.Writer
	formatter formatter

	lock sync.Mutex

	haveBlankLine bool

	tableHeight int

	runningActions []statusTableEntry
}

// NewSmartStatusOutput returns a StatusOutput that represents the
// current build status similarly to Ninja's built-in terminal
// output.
func NewSmartStatusOutput(w io.Writer, formatter formatter) status.StatusOutput {
	tableHeight, _ := strconv.Atoi(os.Getenv("SOONG_UI_TABLE_HEIGHT"))

	s := &smartStatusOutput{
		writer:    w,
		formatter: formatter,

		haveBlankLine: true,
		tableHeight:   tableHeight,
	}

	if tableHeight > 0 {
		if _, height, ok := termSize(s.writer); ok {
			if tableHeight > height-1 {
				tableHeight = height - 1
			}

			// Add empty lines at the bottom of the screen to scroll back the existing history
			// and make room for the status table.
			// TODO: read the cursor position to see if the empty lines are necessary?
			for i := 0; i < tableHeight; i++ {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(s.writer, "\x1b[?25l") // Hide cursor
			s.statusTable()
			ticker := time.NewTicker(time.Second)
			go func() {
				for {
					<-ticker.C
					s.lock.Lock()
					s.statusTable()
					s.lock.Unlock()
				}
			}()
		}
	}

	return s
}

func (s *smartStatusOutput) Message(level status.MsgLevel, message string) {
	if level < status.StatusLvl {
		return
	}

	str := s.formatter.message(level, message)

	s.lock.Lock()
	defer s.lock.Unlock()

	if level > status.StatusLvl {
		s.print(str)
	} else {
		s.statusLine(str)
	}
}

func (s *smartStatusOutput) StartAction(action *status.Action, counts status.Counts) {
	startTime := time.Now()

	str := action.Description
	if str == "" {
		str = action.Command
	}

	progress := s.formatter.progress(counts)

	s.lock.Lock()
	defer s.lock.Unlock()

	s.runningActions = append(s.runningActions, statusTableEntry{
		action:    action,
		startTime: startTime,
	})

	s.statusLine(progress + str)
}

func (s *smartStatusOutput) FinishAction(result status.ActionResult, counts status.Counts) {
	str := result.Description
	if str == "" {
		str = result.Command
	}

	progress := s.formatter.progress(counts) + str

	output := s.formatter.result(result)

	s.lock.Lock()
	defer s.lock.Unlock()

	for i, runningAction := range s.runningActions {
		if runningAction.action == result.Action {
			s.runningActions = append(s.runningActions[:i], s.runningActions[i+1:]...)
			break
		}
	}

	if output != "" {
		s.statusLine(progress)
		s.requestLine()
		s.print(output)
	} else {
		s.statusLine(progress)
	}
}

func (s *smartStatusOutput) Flush() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.requestLine()

	s.runningActions = nil

	if s.tableHeight > 0 {
		s.statusTable()

		fmt.Fprintf(s.writer, "\x1b[r") // Set Top and Bottom Margins DECSTBM
		_, height, _ := termSize(s.writer)
		fmt.Fprintf(s.writer, "\x1b[%d;0H", height-s.tableHeight) // Direct cursor addressing
		fmt.Fprintf(s.writer, "\x1b[?25h")                        // Show cursor
	}
}

func (s *smartStatusOutput) Write(p []byte) (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.print(string(p))
	return len(p), nil
}

func (s *smartStatusOutput) requestLine() {
	if !s.haveBlankLine {
		fmt.Fprintln(s.writer)
		s.haveBlankLine = true
	}
}

func (s *smartStatusOutput) print(str string) {
	if !s.haveBlankLine {
		fmt.Fprint(s.writer, "\r", "\x1b[K")
		s.haveBlankLine = true
	}
	fmt.Fprint(s.writer, str)
	if len(str) == 0 || str[len(str)-1] != '\n' {
		fmt.Fprint(s.writer, "\n")
	}
}

func (s *smartStatusOutput) statusLine(str string) {
	idx := strings.IndexRune(str, '\n')
	if idx != -1 {
		str = str[0:idx]
	}

	// Limit line width to the terminal width, otherwise we'll wrap onto
	// another line and we won't delete the previous line.
	//
	// Run this on every line in case the window has been resized while
	// we're printing. This could be optimized to only re-run when we get
	// SIGWINCH if it ever becomes too time consuming.
	if width, _, ok := termSize(s.writer); ok {
		str = s.elide(str, width)
	}

	// Move to the beginning on the line, turn on bold, print the output,
	// turn off bold, then clear the rest of the line.
	start := "\r\x1b[1m"
	end := "\x1b[0m\x1b[K"
	fmt.Fprint(s.writer, start, str, end)
	s.haveBlankLine = false
}

func (s *smartStatusOutput) elide(str string, width int) string {
	if len(str) > width {
		// TODO: Just do a max. Ninja elides the middle, but that's
		// more complicated and these lines aren't that important.
		str = str[:width]
	}

	return str
}

func (s *smartStatusOutput) statusTable() {
	width, height, ok := termSize(s.writer)
	if !ok {
		return
	}

	tableHeight := s.tableHeight
	if tableHeight > height-1 {
		tableHeight = height - 1
	}

	scrollingHeight := height - tableHeight

	fmt.Fprintf(s.writer, "\x1b[0;%dr", scrollingHeight)   // Set Top and Bottom Margins DECSTBM
	fmt.Fprintf(s.writer, "\x1b[%d;0H", scrollingHeight+1) // Direct cursor addressing

	var tableLine int
	var runningAction statusTableEntry
	for tableLine, runningAction = range s.runningActions {
		if tableLine > tableHeight {
			break
		}

		runningTime := time.Since(runningAction.startTime).Round(time.Second)

		str := fmt.Sprintf("   %2d:%02d %s",
			int(runningTime.Minutes()), int(runningTime.Seconds())%60, runningAction.action.Description)
		str = s.elide(str, width)
		fmt.Fprint(s.writer, str, "\x1b[K\n")
	}

	for ; tableLine < tableHeight; tableLine++ {
		fmt.Fprintln(s.writer, "\x1b[K")
	}

	fmt.Fprintf(s.writer, "\x1b[%d;0H", scrollingHeight) // Direct cursor addressing
}
