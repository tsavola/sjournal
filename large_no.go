// Copyright 2023 Timo Savola. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !(darwin || unix)

package sjournal

const LargeMessageSupport = false

func (h *Handler) sendViaFileIfTooLarge(err error, b []byte) error {
	return err
}
