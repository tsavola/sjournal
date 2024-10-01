// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sjournal

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"time"
)

var defaultSocket = "/run/systemd/journal/socket"

type HandlerOptions struct {
	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelDebug.
	// The handler calls Level.Level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	Level slog.Leveler

	// Prefix is prepended to message strings.
	Prefix string

	// TimeFormat for attribute values.  Defaults to [time.RFC3339Nano].
	TimeFormat string

	Socket string
}

func NewHandler(opts *HandlerOptions) (*Handler, error) {
	sock, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Net: "unixgram"})
	if err != nil {
		return nil, err
	}

	h := &Handler{
		sock: sock,
		addr: net.UnixAddr{
			Net:  "unixgram",
			Name: defaultSocket,
		},
		timeFormat: time.RFC3339Nano,
	}

	if opts != nil {
		h.level = opts.Level
		if opts.Socket != "" {
			h.addr.Name = opts.Socket
		}
		if opts.TimeFormat != "" {
			h.timeFormat = opts.TimeFormat
		}
		h.msgPrefix = opts.Prefix
	}

	return h, nil
}

type Handler struct {
	level             slog.Leveler
	preformattedAttrs []byte
	// groupPrefix is for the text handler only.
	// It holds the prefix for groups that were already pre-formatted.
	// A group will appear here when a call to WithGroup is followed by
	// a call to WithAttrs.
	groupPrefix string
	groups      []string // all groups started from WithGroup
	nOpenGroups int      // the number of groups opened in preformattedAttrs
	sock        *net.UnixConn
	addr        net.UnixAddr
	timeFormat  string
	msgPrefix   string
}

func (h *Handler) ExtendPrefix(s string) slog.Handler {
	h2 := h.clone()
	h2.msgPrefix = h.msgPrefix + s
	return h2
}

func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool {
	minLevel := slog.LevelDebug
	if h.level != nil {
		minLevel = h.level.Level()
	}
	return l >= minLevel
}

func (h *Handler) clone() *Handler {
	h2 := *h
	h2.preformattedAttrs = slices.Clip(h.preformattedAttrs)
	h2.groups = slices.Clip(h.groups)
	return &h2
}

func (h *Handler) WithAttrs(as []slog.Attr) slog.Handler {
	h2 := h.clone()
	// Pre-format the attributes as an optimization.
	state := h2.newHandleState((*buffer)(&h2.preformattedAttrs), false, "")
	defer state.free()
	state.prefix.WriteString(h.groupPrefix)
	if len(h2.preformattedAttrs) > 0 {
		state.sep = " "
	}
	state.openGroups()
	for _, a := range as {
		state.appendAttr(a)
	}
	// Remember the new prefix for later keys.
	h2.groupPrefix = state.prefix.String()
	// Remember how many opened groups are in preformattedAttrs,
	// so we don't open them again when we handle a Record.
	h2.nOpenGroups = len(h2.groups)
	return h2
}

func (h *Handler) WithGroup(name string) slog.Handler {
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

const (
	prefixEmerg   = "PRIORITY=0\nMESSAGE\n\x00\x00\x00\x00\x00\x00\x00\x00"
	prefixAlert   = "PRIORITY=1\nMESSAGE\n\x00\x00\x00\x00\x00\x00\x00\x00"
	prefixCrit    = "PRIORITY=2\nMESSAGE\n\x00\x00\x00\x00\x00\x00\x00\x00"
	prefixErr     = "PRIORITY=3\nMESSAGE\n\x00\x00\x00\x00\x00\x00\x00\x00"
	prefixWarning = "PRIORITY=4\nMESSAGE\n\x00\x00\x00\x00\x00\x00\x00\x00"
	prefixNotice  = "PRIORITY=5\nMESSAGE\n\x00\x00\x00\x00\x00\x00\x00\x00"
	prefixInfo    = "PRIORITY=6\nMESSAGE\n\x00\x00\x00\x00\x00\x00\x00\x00"
	prefixDebug   = "PRIORITY=7\nMESSAGE\n\x00\x00\x00\x00\x00\x00\x00\x00"
)

var priorityPrefixes = [...]string{
	// Lower levels are prefixDebug (if enabled).
	prefixDebug, // LevelDebug
	prefixInfo,
	prefixInfo,
	prefixInfo,
	prefixInfo,    // LevelInfo
	prefixNotice,  // LevelInfo + 1
	prefixNotice,  // LevelNotice
	prefixNotice,  // LevelWarn - 1
	prefixWarning, // LevelWarn
	prefixErr,
	prefixErr,
	prefixErr,
	prefixErr,  // LevelError
	prefixCrit, // LevelError + 1
	prefixCrit,
	prefixCrit,
	prefixCrit, // LevelCrit
	// Higher levels are prefixAlert.
}

var suffixCache sync.Map

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var prefix string
	var suffix string

	switch i := int(r.Level.Level() - slog.LevelDebug); {
	case i < 0:
		prefix = prefixDebug
	case i < len(priorityPrefixes):
		prefix = priorityPrefixes[i]
	default:
		prefix = prefixAlert
	}

	if x, found := suffixCache.Load(r.PC); found {
		suffix = x.(string)
	} else {
		f, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		suffix = fmt.Sprintf("\nCODE_FILE=%s\nCODE_LINE=%d\nCODE_FUNC=%s\n", f.File, f.Line, f.Function)
		suffixCache.Store(r.PC, suffix)
	}

	state := h.newHandleState(newBuffer(), true, "")
	defer state.free()

	state.buf.WriteString(prefix)
	messageOffset := state.buf.Len()
	state.buf.WriteString(h.msgPrefix)
	state.buf.WriteString(r.Message)
	state.sep = ": "
	state.appendNonBuiltIns(r)
	messageLen := state.buf.Len() - messageOffset
	state.buf.WriteString(suffix)
	if !r.Time.IsZero() {
		state.buf.WriteString("SYSLOG_TIMESTAMP=")
		*state.buf = strconv.AppendInt(*state.buf, r.Time.Unix(), 10)
		state.buf.WriteByte('\n')
	}

	b := *state.buf
	binary.LittleEndian.PutUint64(b[messageOffset-8:], uint64(messageLen))

	if _, _, err := h.sock.WriteMsgUnix(b, nil, &h.addr); err != nil {
		return h.sendViaFileIfTooLarge(err, b)
	}
	return nil
}

func (s *handleState) appendNonBuiltIns(r slog.Record) {
	// preformatted Attrs
	if len(s.h.preformattedAttrs) > 0 {
		s.buf.WriteString(s.sep)
		s.buf.Write(s.h.preformattedAttrs)
		s.sep = " "
	}
	// Attrs in Record -- unlike the built-in ones, they are in groups started
	// from WithGroup.
	s.prefix.WriteString(s.h.groupPrefix)
	s.openGroups()
	r.Attrs(func(a slog.Attr) bool {
		s.appendAttr(a)
		return true
	})
}

// handleState holds state for a single call to commonHandler.handle.
// The initial value of sep determines whether to emit a separator
// before the next key, after which it stays non-empty.
type handleState struct {
	h       *Handler
	buf     *buffer
	freeBuf bool    // should buf be freed?
	sep     string  // separator to write before next key
	prefix  *buffer // for text: key prefix
}

func (h *Handler) newHandleState(buf *buffer, freeBuf bool, sep string) handleState {
	return handleState{
		h:       h,
		buf:     buf,
		freeBuf: freeBuf,
		sep:     sep,
		prefix:  newBuffer(),
	}
}

func (s *handleState) free() {
	if s.freeBuf {
		s.buf.Free()
	}
	s.prefix.Free()
}

func (s *handleState) openGroups() {
	for _, n := range s.h.groups[s.h.nOpenGroups:] {
		s.openGroup(n)
	}
}

// Separator for group names and keys.
const keyComponentSep = '.'

// openGroup starts a new group of attributes
// with the given name.
func (s *handleState) openGroup(name string) {
	s.prefix.WriteString(name)
	s.prefix.WriteByte(keyComponentSep)
}

// closeGroup ends the group with the given name.
func (s *handleState) closeGroup(name string) {
	(*s.prefix) = (*s.prefix)[:len(*s.prefix)-len(name)-1 /* for keyComponentSep */]
	s.sep = " "
}

// appendAttr appends the Attr's key and value using app.
// It handles replacement and checking for an empty key.
// after replacement).
func (s *handleState) appendAttr(a slog.Attr) {
	a.Value = a.Value.Resolve()
	// Elide empty Attrs.
	if a.Equal(slog.Attr{}) {
		return
	}
	// Special cases.
	switch v := a.Value; v.Kind() {
	case slog.KindAny:
		if src, ok := v.Any().(*slog.Source); ok {
			a.Value = slog.StringValue(fmt.Sprintf("%s:%d", src.File, src.Line))
		}
	case slog.KindTime:
		a.Value = slog.StringValue(a.Value.Time().Format(s.h.timeFormat))
	}
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		// Output only non-empty groups.
		if len(attrs) > 0 {
			// Inline a group with an empty key.
			if a.Key != "" {
				s.openGroup(a.Key)
			}
			for _, aa := range attrs {
				s.appendAttr(aa)
			}
			if a.Key != "" {
				s.closeGroup(a.Key)
			}
		}
	} else {
		s.appendKey(a.Key)
		s.appendString(a.Value.String())
	}
}

func (s *handleState) appendKey(key string) {
	s.buf.WriteString(s.sep)
	if s.prefix != nil && len(*s.prefix) > 0 {
		// TODO: optimize by avoiding allocation.
		s.appendString(string(*s.prefix) + key)
	} else {
		s.appendString(key)
	}
	s.buf.WriteByte('=')
	s.sep = " "
}

func (s *handleState) appendString(str string) {
	if needsQuoting(str) {
		*s.buf = strconv.AppendQuote(*s.buf, str)
	} else {
		s.buf.WriteString(str)
	}
}
