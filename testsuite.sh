#!/bin/sh

tmpFolder=$(mktemp -d)

build() {
    ( cd $(dirname "$0") &&
	    go build -buildvcs="false" -o "$tmpFolder/mygit" ./cmd/mygit )
}

build

for test in tests/*.sh
do
    echo "=== Running $(basename $test) ==="
    test_tmp_folder=$(mktemp -d)
    test_path=$(realpath $test)
    ( PATH=$PATH:$tmpFolder; cd $test_tmp_folder; $test_path )
    if [ $? -ne 0 ]; then
        echo "=> $test failed"
    else
        echo "=> $test passed"
    fi
    echo
done