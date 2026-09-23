# The pii-flow checker userspace: CPython 3.14.7 (the adapter) plus the
# upstream Souffle 2.5 Datalog engine (the rules), for linux/amd64.
#
# Every input is pinned by content: the base by manifest digest, each package
# by the sha256 Debian's signed trixie index (or Souffle's published
# sha512sum.txt) vouches for. Nothing is resolved from a package index at build
# time, so the build is reproducible: scripts/pii-flow-image.sh builds it with
# a digest-pinned BuildKit, SOURCE_DATE_EPOCH, and rewritten timestamps, and
# refuses any result whose manifest digest differs from the one the example
# registry pins.
#
# Only the souffle binary is taken from the upstream .deb. Its package also
# depends on g++ and development headers for Souffle's compile mode; the
# checker runs the interpreter only, so the image carries the runtime
# libraries the binary links (libgomp1 and libncurses6 are the two the base
# lacks) and nothing else. The adapter invokes souffle with --no-preprocessor,
# so no C preprocessor is needed either.

FROM --platform=linux/amd64 python:3.14.7-slim-trixie@sha256:7bf6c3111fe094f8ee1a1cbcdc63c4cfb345b0e3df42d5aa9a90b3b4b022ab6d AS fetch
ADD --checksum=sha256:c7e9dd1349506bbb23c4dcf89e87396198006235f79b1cc516c0a2b67ac067bc \
    https://github.com/souffle-lang/souffle/releases/download/2.5/x86_64-ubuntu-2404-souffle-2.5-Linux.deb \
    /pkgs/souffle.deb
ADD --checksum=sha256:72b7a1a33c48f3195efeb044111b34ae684b1e640bb6495c832443b798601cd5 \
    https://snapshot.debian.org/archive/debian/20250315T204837Z/pool/main/g/gcc-14/libgomp1_14.2.0-19_amd64.deb \
    /pkgs/libgomp1.deb
ADD --checksum=sha256:e00cbcc8c0826993f881f492c05f33ba8d79b5375dc5a2ab59c82e0c7b9179a9 \
    https://snapshot.debian.org/archive/debian/20250306T211336Z/pool/main/n/ncurses/libncurses6_6.5%2B20250216-2_amd64.deb \
    /pkgs/libncurses6.deb
RUN dpkg-deb --fsys-tarfile /pkgs/souffle.deb | tar -xO ./usr/bin/souffle > /pkgs/souffle \
 && chmod 0755 /pkgs/souffle \
 && rm /pkgs/souffle.deb

FROM --platform=linux/amd64 python:3.14.7-slim-trixie@sha256:7bf6c3111fe094f8ee1a1cbcdc63c4cfb345b0e3df42d5aa9a90b3b4b022ab6d
RUN --mount=type=bind,from=fetch,source=/pkgs,target=/pkgs \
    dpkg -i /pkgs/libgomp1.deb /pkgs/libncurses6.deb >/dev/null \
 && install -m 0755 /pkgs/souffle /usr/local/bin/souffle \
 && rm -f /var/log/dpkg.log /var/cache/ldconfig/aux-cache \
 && souffle --version >/dev/null
