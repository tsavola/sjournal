// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sjournal

import (
	"log/slog"
)

const (
	LevelDebug  = slog.LevelDebug
	LevelInfo   = slog.LevelInfo
	LevelNotice = slog.LevelInfo + 2
	LevelWarn   = slog.LevelWarn
	LevelError  = slog.LevelError
	LevelCrit   = slog.LevelError + 4
	LevelAlert  = slog.LevelError + 8
)
