# souffle.dockerfile - the linux/amd64 image the datalog-parity lane runs in:
# the Soufflé reference engine plus the Go toolchain, so the unchanged parity
# tests (go test ./internal/datalog/ ./internal/gates/ -run Parity) find
# souffle on PATH and compare both engines.
#
# Soufflé publishes no OCI image, so this file builds one from pinned inputs
# only (the same discipline docs/external-checkers.md holds checker images to):
#
#   - the base image by digest (ubuntu 24.04, the distribution the upstream
#     .deb is built for);
#   - the Soufflé 2.5 release asset by sha256, checked before it is
#     installed (its sha512 is the one upstream publishes in sha512sum.txt);
#   - its runtime dependencies from the Ubuntu snapshot archive at a fixed
#     timestamp, so a rebuild resolves the same package versions;
#   - the Go toolchain copied from the golang image by digest, the Go 1.27.1
#     identity scripts/ci-linux.dockerfile pins.
#
# To repin: change a digest, the sha256, or the snapshot below, rebuild with
# `docker build --platform linux/amd64 -f scripts/souffle.dockerfile .`, and
# check `souffle --version` still reports the version the host runs.

ARG GO_IMAGE=golang:1.27.1-trixie@sha256:433790e515d27dc6003e847e644cc0af956985cf315c1c58a3b73ee2dd305183
ARG BASE_IMAGE=ubuntu:24.04@sha256:008173c23f95b170204355c12626cb5a965d779a7e1283b09e9cffbb1bf33ca3

FROM ${GO_IMAGE} AS go

# The release asset is fetched and checked in the go stage, which already has
# curl and CA roots; only the verified file crosses into the final image.
FROM go AS asset
ARG SOUFFLE_URL=https://github.com/souffle-lang/souffle/releases/download/2.5/x86_64-ubuntu-2404-souffle-2.5-Linux.deb
ARG SOUFFLE_SHA256=c7e9dd1349506bbb23c4dcf89e87396198006235f79b1cc516c0a2b67ac067bc
RUN curl -fsSL -o /souffle.deb "${SOUFFLE_URL}" \
 && echo "${SOUFFLE_SHA256}  /souffle.deb" | sha256sum -c -

FROM ${BASE_IMAGE}

ARG APT_SNAPSHOT=20260920T000000Z

# The snapshot archive is HTTPS only and the base image carries no CA roots,
# so ca-certificates alone comes from the base image's own archive; every
# Soufflé dependency then resolves from the snapshot.
COPY --from=asset /souffle.deb /tmp/souffle.deb
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && apt-get update --snapshot "${APT_SNAPSHOT}" \
 && apt-get install -y --no-install-recommends --snapshot "${APT_SNAPSHOT}" /tmp/souffle.deb \
 && rm -f /tmp/souffle.deb \
 && rm -rf /var/lib/apt/lists/* \
 && souffle --version

COPY --from=go /usr/local/go /usr/local/go
ENV PATH=/usr/local/go/bin:${PATH} \
    GOTOOLCHAIN=local \
    CGO_ENABLED=0
