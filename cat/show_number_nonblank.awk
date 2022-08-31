# Copyright 2022 Michal Vyskocil. All rights reserved.
# Use of this source code is governed by a MIT
# license that can be found in the LICENSE file.

BEGIN { n = 1; }
{
    if (NF > 0) {
        printf("%6d\t%s\n", n, $_);
        n++;
    } else {
        print;
    }
}
