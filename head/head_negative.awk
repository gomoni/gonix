# Copyright 2022 Michal Vyskocil. All rights reserved.
# Use of this source code is governed by a MIT
# license that can be found in the LICENSE file.
function alen(a, i, k) {
    k = 0
    for(i in a) k++
    return k
}
BEGIN {
	delete ring_buf[0]
	buf_idx = 0
	buf_head = 0
}
{
	if (alen(ring_buf) < lines) {
		ring_buf[buf_idx]=$0
		buf_idx ++
		next
	}

	print(ring_buf[buf_head])
	ring_buf[buf_head]=$0
	buf_head ++
	if (buf_head == lines)
		buf_head = 0
}
