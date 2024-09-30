// Copyright 2023 Timo Savola. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (darwin || unix) && !linux

package sjournal

import (
	"os"
)

func createNonlinkedFile() (*os.File, error) {
	var ok bool

	f, err := os.CreateTemp("", "journal-entry-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if !ok {
			f.Close()
		}
	}()

	os.Remove(f.Name())
	ok = true

	return f, nil
}
