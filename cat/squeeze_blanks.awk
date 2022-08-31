# Copyright 2022 Michal Vyskocil. All rights reserved.
# Use of this source code is governed by a MIT
# license that can be found in the LICENSE file.

BEGIN {
    squeeze = 0;
}
{
    if (NF == 0) {
        if (squeeze==0) {print};
        squeeze = 1;
    } else {
        squeeze = 0;
    }
    if (squeeze == 0) {
        print($_);
    }
}
