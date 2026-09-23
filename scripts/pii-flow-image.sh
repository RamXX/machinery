#!/usr/bin/env bash
# Provision the digest-pinned pii-flow checker image (CPython plus Souffle).
#
# No upstream registry publishes a Souffle image, so the image is built from
# examples/pii-flow/souffle-image/Dockerfile, whose every input is pinned by
# content. The build is reproducible: a digest-pinned BuildKit, a fixed
# SOURCE_DATE_EPOCH, and rewritten layer timestamps make the manifest digest a
# pure function of the Dockerfile. The result is pushed to a throwaway loopback
# registry and pulled back, because only a registry round trip gives the local
# engine the RepoDigests entry `machinery verify-checkers` requires. A build
# whose digest differs from the pin in the example registry fails; nothing is
# ever re-pinned implicitly.
#
# Usage: scripts/pii-flow-image.sh [docker executable]
# Silent apart from one confirmation line on success.
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)
cd "$repo_root"

docker_bin=${1:-docker}
registry_file=examples/pii-flow/checkers.local.example.yaml
context_dir=examples/pii-flow/souffle-image
platform=linux/amd64
# Souffle 2.5's upstream release time; any fixed value works, this one is
# traceable.
source_date_epoch=1742825974
registry_image=registry@sha256:325b4b29b041e82803abeb703e201655e4e23ab83264ec1a7c9ddb0a5b14a6e0
buildkit_image=moby/buildkit@sha256:6c2fa84a6b61ccd72899dde4239f8d5717f05f9a8ca6f3cad185fb1a95a94de3
registry_name=machinery-pii-flow-registry
builder_name=machinery-pii-flow-builder

fail() { echo "pii-flow-image: $*" >&2; exit 1; }

image=$(sed -n 's/^ *image: *//p' "$registry_file")
[[ "$image" =~ ^(localhost:[0-9]+)/[a-z0-9/-]+@sha256:[0-9a-f]{64}$ ]] ||
  fail "$registry_file must pin exactly one loopback image by digest, got: $image"
registry_host=${BASH_REMATCH[1]}
registry_port=${registry_host#localhost:}
name=${image%@*}
digest=${image#*@}

provisioned() {
  local inspect
  inspect=$("$docker_bin" image inspect --format '{{json .RepoDigests}} {{.Os}}/{{.Architecture}}' "$image" 2>/dev/null) || return 1
  [[ "$inspect" == *\""$image"\"*" $platform" ]]
}

if ! provisioned; then
  work=$(mktemp -d)
  started_registry=no
  created_builder=no
  cleanup() {
    if [[ $created_builder == yes ]]; then "$docker_bin" buildx rm "$builder_name" >/dev/null 2>&1 || true; fi
    if [[ $started_registry == yes ]]; then "$docker_bin" rm -f "$registry_name" >/dev/null 2>&1 || true; fi
    rm -rf -- "$work"
  }
  trap cleanup EXIT

  if [[ -z "$("$docker_bin" ps -q --filter "name=^${registry_name}$")" ]]; then
    "$docker_bin" rm -f "$registry_name" >/dev/null 2>&1 || true
    "$docker_bin" pull --quiet "$registry_image" >/dev/null ||
      fail "could not pull the pinned registry image $registry_image"
    "$docker_bin" run -d --name "$registry_name" -p "127.0.0.1:${registry_port}:5000" "$registry_image" >/dev/null ||
      fail "could not start the loopback registry on port $registry_port"
    started_registry=yes
  fi
  if ! "$docker_bin" buildx inspect "$builder_name" >/dev/null 2>&1; then
    "$docker_bin" buildx create --name "$builder_name" --driver docker-container \
      --driver-opt "image=$buildkit_image" --driver-opt network=host >/dev/null ||
      fail "could not create the pinned BuildKit builder"
    created_builder=yes
  fi

  SOURCE_DATE_EPOCH=$source_date_epoch "$docker_bin" buildx build --builder "$builder_name" \
    --platform "$platform" --provenance=false --sbom=false --no-cache --quiet \
    --metadata-file "$work/metadata.json" \
    --output "type=image,name=$name,push=true,registry.insecure=true,oci-mediatypes=true,rewrite-timestamp=true,compression=gzip,force-compression=true" \
    "$context_dir" >"$work/build.out" 2>"$work/build.err" || {
      cat "$work/build.err" >&2
      fail "image build failed"
    }
  built=$(sed -n 's/.*"containerimage.digest": *"\(sha256:[0-9a-f]\{64\}\)".*/\1/p' "$work/metadata.json")
  [[ -n "$built" ]] || fail "build metadata carries no image digest"
  [[ "$built" == "$digest" ]] ||
    fail "rebuilt image digest $built does not match the pinned $digest; the build is not reproducible here, or an input changed"

  "$docker_bin" pull --quiet --platform "$platform" "$image" >/dev/null ||
    fail "could not pull $image back from the loopback registry"
  provisioned || fail "local engine does not report $image for $platform after the pull"
fi

version=$("$docker_bin" run --rm --pull=never --platform "$platform" --network=none --read-only "$image" \
  souffle --version 2>&1) || fail "pinned image cannot run souffle offline on $platform"
case "$version" in
  *$'\nVersion: 2.5 '*) ;;
  *) fail "pinned image does not carry Souffle 2.5: $version" ;;
esac
echo "pii-flow checker image provisioned: $image ($platform)"
