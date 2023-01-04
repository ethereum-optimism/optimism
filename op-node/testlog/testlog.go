// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package testlog provides a log handler for unit tests.
package testlog

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

var useColorInTestLog bool = true

func init() {
	if os.Getenv("OP_TESTLOG_DISABLE_COLOR") == "true" {
		useColorInTestLog = false
	}
}

// Testing interface to log to. Some functions are marked as Helper function to log the call site accurately.
// Standard Go testing.TB implements this, as well as Hive and other Go-like test frameworks.
type Testing interface {
	Logf(format string, args ...any)
	Helper()
}

// Handler returns a log handler which logs to the unit test log of t.
func Handler(t Testing, level log.Lvl) log.Handler {
	return log.LvlFilterHandler(level, &handler{t, log.TerminalFormat(false)})
}

type handler struct {
	t   Testing
	fmt log.Format
}

func (h *handler) Log(r *log.Record) error {
	h.t.Logf("%s", h.fmt.Format(r))
	return nil
}

// logger implements log.Logger such that all output goes to the unit test log via
// t.Logf(). All methods in between logger.Trace, logger.Debug, etc. are marked as test
// helpers, so the file and line number in unit test output correspond to the call site
// which emitted the log message.
type logger struct {
	t  Testing
	l  log.Logger
	mu *sync.Mutex
	h  *bufHandler
}

type bufHandler struct {
	buf []*log.Record
	fmt log.Format
}

func (h *bufHandler) Log(r *log.Record) error {
	h.buf = append(h.buf, r)
	return nil
}

// Logger returns a logger which logs to the unit test log of t.
func Logger(t Testing, level log.Lvl) log.Logger {
	l := &logger{
		t:  t,
		l:  log.New(),
		mu: new(sync.Mutex),
		h:  &bufHandler{fmt: log.TerminalFormat(useColorInTestLog)},
	}
	l.l.SetHandler(log.LvlFilterHandler(level, l.h))
	return l
}

func (l *logger) Trace(msg string, ctx ...any) {
	l.t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.Trace(msg, ctx...)
	l.flush()
}

func (l *logger) Debug(msg string, ctx ...any) {
	l.t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.Debug(msg, ctx...)
	l.flush()
}

func (l *logger) Info(msg string, ctx ...any) {
	l.t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.Info(msg, ctx...)
	l.flush()
}

func (l *logger) Warn(msg string, ctx ...any) {
	l.t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.Warn(msg, ctx...)
	l.flush()
}

func (l *logger) Error(msg string, ctx ...any) {
	l.t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.Error(msg, ctx...)
	l.flush()
}

func (l *logger) Crit(msg string, ctx ...any) {
	l.t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.Crit(msg, ctx...)
	l.flush()
}

func (l *logger) New(ctx ...any) log.Logger {
	return &logger{l.t, l.l.New(ctx...), l.mu, l.h}
}

func (l *logger) GetHandler() log.Handler {
	return l.l.GetHandler()
}

func (l *logger) SetHandler(h log.Handler) {
	l.l.SetHandler(h)
}

// flush writes all buffered messages and clears the buffer.
func (l *logger) flush() {
	l.t.Helper()
	// 2 frame skip for flush() + public logger fn
	decorationLen := estimateInfoLen(2)
	padding := 0
	padLength := 30
	if decorationLen <= padLength {
		padding = padLength - decorationLen
	}
	for _, r := range l.h.buf {
		l.t.Logf("%*s%s", padding, "", l.h.fmt.Format(r))
	}
	l.h.buf = nil
}

// The Go testing lib uses the runtime package to get info about the calling site, and then decorates the line.
// We can't disable this decoration, but we can adjust the contents to align by padding after the info.
// To pad the right amount, we estimate how long the info is.
func estimateInfoLen(frameSkip int) int {
	var pc [50]uintptr
	// Skip two extra frames to account for this function
	// and runtime.Callers itself.
	n := runtime.Callers(frameSkip+2, pc[:])
	if n == 0 {
		return 8
	}
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	file := frame.File
	line := frame.Line
	if file != "" {
		// Truncate file name at last file name separator.
		if index := strings.LastIndex(file, "/"); index >= 0 {
			file = file[index+1:]
		} else if index = strings.LastIndex(file, "\\"); index >= 0 {
			file = file[index+1:]
		}
		return 4 + len(file) + 1 + len(strconv.FormatInt(int64(line), 10))
	} else {
		return 8
	}
}
