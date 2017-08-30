#!/usr/bin/env bash
#
# Run all CNI tests
#   ./test
#   ./test -v
#
# Run tests for one package
#   PKG=./libcni ./test
#
set -e

source ./build.sh

# test everything that's not in vendor
pushd "$GOPATH/src/$REPO_PATH" >/dev/null
  TESTABLE="$(go list ./... | grep -v vendor | xargs echo)"
popd >/dev/null

FORMATTABLE="$TESTABLE"

# user has not provided PKG override
if [ -z "$PKG" ]; then
	TEST=$TESTABLE
	FMT=$FORMATTABLE

# user has provided PKG override
else
	# strip out slashes and dots from PKG=./foo/
	TEST=${PKG//\//}
	TEST=${TEST//./}

	# only run gofmt on packages provided by user
	FMT="$TEST"
fi

echo -n "Running tests "
function testrun {
    sudo -E bash -c "umask 0; PATH=$GOROOT/bin:$(pwd)/bin:$PATH go test -covermode set $@"
}
if [ ! -z "${COVERALLS}" ]; then
    echo "with coverage profile generation..."
    i=0
    for t in ${TEST}; do
        testrun "-coverprofile ${i}.coverprofile ${t}"
        i=$((i+1))
    done
    gover
    goveralls -service=travis-ci -coverprofile=gover.coverprofile -repotoken=$COVERALLS_TOKEN
else
    echo "without coverage profile generation..."
    testrun "${TEST}"
fi

echo "Checking gofmt..."
fmtRes=$(go fmt $FMT)
if [ -n "${fmtRes}" ]; then
	echo -e "go fmt checking failed:\n${fmtRes}"
	exit 255
fi

echo "Checking govet..."
vetRes=$(go vet $TEST)
if [ -n "${vetRes}" ]; then
	echo -e "go vet checking failed:\n${vetRes}"
	exit 255
fi

echo "Checking license header..."
licRes=$(
       for file in $(find . -type f -iname '*.go' ! -path './vendor/*'); do
               head -n1 "${file}" | grep -Eq "(Copyright|generated)" || echo -e "  ${file}"
       done
)
if [ -n "${licRes}" ]; then
       echo -e "license header checking failed:\n${licRes}"
       exit 255
fi


echo "Success"
