#!/bin/sh
# run-suite.sh is how this project runs its Go suite: it stages the test sources in stage/suite
# and starts go test itself, twice, once plain and once for the tests behind the build tag
# faults. When MACHINERY_GO_TEST_JSON names a directory, each run leaves its event stream there.
# SUITE_FLAGS replaces the default flags (-count=1).
set -u
flags="${SUITE_FLAGS:--count=1}"
rm -rf stage
mkdir -p stage/suite
cp e2e/suite/*_test.go stage/suite/
status=0
if [ -n "${MACHINERY_GO_TEST_JSON:-}" ]; then
  go test -json $flags ./stage/suite > "$MACHINERY_GO_TEST_JSON/1-plain.json" || status=$?
  go test -json $flags -tags faults -run TestProviderDown ./stage/suite > "$MACHINERY_GO_TEST_JSON/2-faults.json" || status=$?
else
  go test $flags ./stage/suite || status=$?
  go test $flags -tags faults -run TestProviderDown ./stage/suite || status=$?
fi
rm -rf stage
echo "run-suite: 2 go test run(s), exit $status"
exit $status
