// Copyright 2023 Timo Savola. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package sjournal

import (
	"os"

	"golang.org/x/sys/unix"
)

func createNonlinkedFile() (*os.File, error) {
	fd, err := unix.MemfdCreate("journal-entry", unix.MFD_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "journal-entry"), nil
}
