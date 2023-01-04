#!/bin/bash
#
# Build sbase utilities via

if [[ ! -f sbase/cat.1 ]]; then
    echo "[ERROR]: sbase/ git submodule is missing or empty"
    exit 1
fi

if ! type zig 1>/dev/null; then
    echo "[ERROR]: zig command not found"
    exit 1
fi

set -e
cd sbase

export CC="zig cc"
export AR="zig ar"
export RANLIB="zig ranlib"
export CFLAGS="-target wasm32-wasi"
export LDFLAGS="-target wasm32-wasi"

make \
    CC="${CC}" \
    CFLAGS="${CFLAGS}" \
    LDFLAGS="${LDFLAGS}" \
    AR="${AR}" \
    RANLIB="${RANLIB}" \
    libutf.a libutil.a

# Standalone binaries - each built into own wasm module
standalone() {
    local BIN=${1}
    make \
        CC='zig cc' \
        CFLAGS="${CFLAGS}" \
        LDFLAGS="${LDFLAGS}" \
        AR='zig ar' \
        RANLIB='zig ranlib' \
        BIN="${BIN}"
    chmod a-x "${BIN}"
    if [[ -d ../../${BIN} ]]; then
        mv "${BIN}" ../../"${BIN}"/"${BIN}".wasm
    fi
}

BINS=(basename cal cat chgrp chmod chown chroot cksum cmp comm cp cut date dd dirname du echo env expand expr false fold getconf grep head hostname join link ln logger logname ls md5sum mkdir mkfifo mknod mktemp mv nl od paste pathchk printenv printf pwd readlink rev rm rmdir sed seq setsid sha1sum sha224sum sha256sum sha384sum sha512sum sha512-224sum sha512-256sum sleep sort split sponge strings sync tail tar test touch tr true tsort uname unexpand uniq unlink uudecode uuencode wc which whoami yes)

for BIN in "${BINS[@]}"; do
    standalone "${BIN}"
done

# TODO:
# 1. test generated cat wasm: PASSED
# 2. test the sbase-box target, maybe it's better: has 2.5MB - unless result can be cached, no!
# 3. figure out why one needs to call only one BIN
# 4. collect warnings and implicit funcs like mknod and so - done for sbase-box
# 5. figure out which sbase commands are we going to implement
#
# grep/sed/sort/tr/tsort

# TODO: build a sbase-box - generates 2,5 MB big binary for all bins, but can be worth effort for smaller binary with fewer utilities
#make \
#    CC="${CC}" \
#    CFLAGS="${CFLAGS}" \
#    LDFLAGS="${LDFLAGS}" \
#    AR="${AR}" \
#    RANLIB="${RANLIB}"
#    BIN="${BINS[*]}" \
#    sbase-box
