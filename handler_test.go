// Copyright 2024 Timo Savola. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sjournal

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/slogtest"
	"time"
)

func TestHandler(t *testing.T) {
	const stopMagic = "MHJKRUECSJ"

	sockPath := path.Join(t.TempDir(), "socket")

	sock, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Net: "unixgram", Name: sockPath})
	if err != nil {
		t.Fatal(err)
	}

	received := make(chan error, 1)
	t.Cleanup(func() { <-received })

	var (
		mu sync.Mutex
		ms []map[string]any
	)

	go func() {
		defer close(received)
		defer sock.Close()

		var (
			buf = make([]byte, 65536)
			err error
		)

		for {
			var n int

			n, _, _, _, err = sock.ReadMsgUnix(buf, nil)
			if err != nil {
				break
			}

			b := buf[:n]

			if strings.Contains(string(b), stopMagic) {
				break
			}

			var m map[string]any

			m, err = parseProtocolMessage(b)
			if err != nil {
				break
			}

			ms = append(ms, m)
		}

		received <- err
	}()

	h, err := NewHandler(&HandlerOptions{
		Level:     slog.LevelInfo,
		Delimiter: ColonDelimiter,
		Socket:    sockPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	results := func() []map[string]any {
		time.Sleep(time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		return slices.Clip(ms)
	}

	if err := slogtest.TestHandler(h, results); err != nil {
		t.Error(err)
	}

	if err := h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, stopMagic, 0)); err != nil {
		t.Error(err)
	}

	if err, ok := <-received; !ok {
		t.Error("receiver panicked")
	} else if err != nil {
		t.Error("receiver error:", err)
	}
}

func parseProtocolMessage(b []byte) (map[string]any, error) {
	r := bytes.NewBuffer(b)

	data := make(map[string]string)

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF && line == "" {
				break
			}
			return nil, err
		}
		line = line[:len(line)-1]

		var (
			key   string
			value string
		)

		if pair := strings.SplitN(line, "=", 2); len(pair) == 2 {
			key = pair[0]
			value = pair[1]
		} else {
			key = line

			b := make([]byte, 8)
			if _, err := io.ReadFull(r, b); err != nil {
				return nil, err
			}
			size := binary.LittleEndian.Uint64(b)

			b = make([]byte, size)
			if _, err := io.ReadFull(r, b); err != nil {
				return nil, err
			}
			value = string(b)

			c, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			if c != '\n' {
				return nil, fmt.Errorf("key %q: newline expected after length-encoded value %q", key, value)
			}
		}

		data[key] = value
	}

	value, found := data["MESSAGE"]
	if !found {
		return nil, errors.New("MESSAGE key not found")
	}
	m, err := parseMessageValue(value)
	if err != nil {
		return nil, err
	}

	value, found = data["PRIORITY"]
	if !found {
		return nil, errors.New("PRIORITY key not found")
	}
	switch value {
	case "3":
		m["level"] = slog.LevelError
	case "6":
		m["level"] = slog.LevelInfo
	default:
		return nil, fmt.Errorf("unexpected PRIORITY value: %q", value)
	}

	if value, found := data["SYSLOG_TIMESTAMP"]; found {
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		m["time"] = time.Unix(n, 0)
	}

	for _, key := range []string{"CODE_FILE", "CODE_FUNC", "CODE_LINE"} {
		if _, found := data[key]; !found {
			return nil, fmt.Errorf("%s key not found", key)
		}
	}

	return m, nil
}

func parseMessageValue(s string) (map[string]any, error) {
	m := make(map[string]any)

	pair := strings.SplitN(s, ": ", 2)
	if len(pair) == 1 {
		m["msg"] = s
		return m, nil
	}

	m["msg"] = pair[0]

	for _, attr := range strings.Fields(pair[1]) {
		pair := strings.SplitN(attr, "=", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("attribute parse error: %q", attr)
		}
		key := pair[0]
		value := pair[1]
		setAttr(m, key, value)
	}

	return m, nil
}

func setAttr(m map[string]any, key, value string) {
	pair := strings.SplitN(key, ".", 2)
	if len(pair) == 1 {
		m[key] = value
		return
	}

	name := pair[0]

	group, found := m[name]
	if !found {
		group = make(map[string]any)
		m[name] = group
	}

	setAttr(group.(map[string]any), pair[1], value)
}
