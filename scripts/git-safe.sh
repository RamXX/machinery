#!/usr/bin/env bash
# Shared shell adapter for scripts/git-safe. Source this file, call
# git_safe_prepare once, then use git_safe in place of correctness-critical Git.

git_safe_prepare() {
  local trusted_root=$1 destination=$2 build_stderr
  build_stderr=$destination.stderr
  if ! go build -o "$destination" "$trusted_root/scripts/git-safe" 2>"$build_stderr"; then
    printf 'build bounded Git helper failed\n' >&2
    cat "$build_stderr" >&2
    return 1
  fi
  if [[ -s "$build_stderr" ]]; then
    printf 'building bounded Git helper emitted stderr; warnings are forbidden\n' >&2
    cat "$build_stderr" >&2
    return 1
  fi
  GIT_SAFE_ROOT=$trusted_root
  GIT_SAFE_BIN=$destination
}

git_safe() {
  "${GIT_SAFE_BIN:?git_safe_prepare was not called}" -root "${GIT_SAFE_ROOT:?git_safe_prepare was not called}" -- "$@"
}
