# ci-linux.dockerfile - the linux/amd64 image `make ci-linux` runs the heavy
# tier in.
#
# Every runtime identity here mirrors .github/actions/assurance-runtimes, the
# single owner of the hosted runtime pins: Go 1.27.1, Node 26.8.1 with
# TypeScript 7.0.2, CPython 3.14.7, Elixir 1.20.4 on OTP 29.0.6. The required
# lane re-verifies each identity against testdata/integration-lanes and fails
# closed, so a drifted stage here fails loudly instead of producing evidence
# that does not match hosted CI.
#
# Each stage is an override-able ARG so a moved upstream tag can be repinned
# without editing this file: scripts/ci-linux.sh forwards
# MACHINERY_CI_LINUX_<STAGE>_IMAGE for every stage.

ARG GO_IMAGE=golang:1.27.1-trixie
ARG NODE_IMAGE=node:26.8.1-trixie-slim
ARG PYTHON_IMAGE=python:3.14.7-slim-trixie
ARG ELIXIR_IMAGE=hexpm/elixir:1.20.4-erlang-29.0.6-debian-trixie-20260824-slim
ARG DOCKER_CLI_IMAGE=docker:29.7.2-cli

FROM ${NODE_IMAGE} AS node
FROM ${PYTHON_IMAGE} AS python
FROM ${ELIXIR_IMAGE} AS elixir
FROM ${DOCKER_CLI_IMAGE} AS dockercli

FROM ${GO_IMAGE}

# Shared library surface for the copied CPython and OTP runtimes, plus the
# tools the lane and the sweep shell out to.
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates curl git xz-utils procps \
      libexpat1 libffi8 libsqlite3-0 libbz2-1.0 liblzma5 libncursesw6 \
      libreadline8 libssl3 zlib1g \
 && rm -rf /var/lib/apt/lists/*

# Node 26.8.1 and its npm, from the pinned upstream image.
COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -sf ../lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
 && ln -sf ../lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

# CPython 3.14.7, from the pinned upstream image.
COPY --from=python /usr/local/bin/python3.14 /usr/local/bin/python3.14
COPY --from=python /usr/local/lib/python3.14 /usr/local/lib/python3.14
COPY --from=python /usr/local/lib/libpython3.14.so.1.0 /usr/local/lib/libpython3.14.so.1.0
RUN ln -sf python3.14 /usr/local/bin/python3 && ldconfig

# Elixir 1.20.4 on OTP 29.0.6, from the pinned upstream image.
COPY --from=elixir /usr/local/lib/erlang /usr/local/lib/erlang
COPY --from=elixir /usr/local/lib/elixir /usr/local/lib/elixir
RUN for tool in erl erlc escript elixir elixirc mix iex; do \
      for prefix in /usr/local/lib/erlang/bin /usr/local/lib/elixir/bin; do \
        if [ -x "$prefix/$tool" ]; then ln -sf "$prefix/$tool" "/usr/local/bin/$tool"; fi; \
      done; \
    done

# TypeScript 7.0.2, the compiler identity the assurance catalog pins.
RUN npm install -g typescript@7.0.2

# The Docker client only. The lane talks to the host daemon over the socket
# the runner mounts; no daemon runs inside this image.
COPY --from=dockercli /usr/local/bin/docker /usr/local/bin/docker

# The lane and the sweep both write under the module cache; keep it inside the
# image so a cold container does not re-download on every run. The BEAM reads
# its filename encoding from the locale: without a UTF-8 locale `elixir
# --version` prints a latin1 warning on stderr, and the lane's runtime probe
# treats any unexpected diagnostic as a failed identity (evidence run on the
# Linux VM, 2026-09-09).
ENV GOFLAGS=-buildvcs=false LANG=C.UTF-8 LC_ALL=C.UTF-8
WORKDIR /src
