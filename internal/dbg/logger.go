// Copyright 2022 Michal Vyskocil. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package dbg

import (
	"io"
	"log"
)

func Logger(enabled bool, command string, stderr io.Writer) *log.Logger {
	if !enabled {
		stderr = io.Discard
	}
	return log.New(stderr, "[DBG]"+command+": ", log.Flags())
}
