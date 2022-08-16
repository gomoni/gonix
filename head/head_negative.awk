# Copyright 2022 Michal Vyskocil. All rights reserved.
# Use of this source code is governed by a MIT
# license that can be found in the LICENSE file.
BEGIN {
	delete ring_buf[0]
	buf_idx = 0
	buf_head = 0
    buf_full = 0
}
{
	if (! buf_full) {
		ring_buf[buf_idx]=$0
		buf_idx ++
        if (buf_idx == lines) {
            buf_full = 1
        }
        next
	}

	print(ring_buf[buf_head])
	ring_buf[buf_head]=$0
	buf_head ++
	if (buf_head == lines)
		buf_head = 0
}
