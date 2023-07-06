// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin || unix

package sjournal

import (
	"errors"
	"syscall"
)

const LargeMessageSupport = true

func (h *Handler) sendViaFileIfTooLarge(err error, b []byte) error {
	if !(errors.Is(err, syscall.EMSGSIZE) || errors.Is(err, syscall.ENOBUFS)) {
		return err
	}

	f, err := createNonlinkedFile()
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(b); err != nil {
		return err
	}

	if _, _, err := h.sock.WriteMsgUnix(nil, syscall.UnixRights(int(f.Fd())), &h.addr); err != nil {
		return err
	}
	return nil
}
