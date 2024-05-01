#!/bin/sh

tmpFolder=$(mktemp -d)

build() {
    go build -o "$tmpFolder/mygit"
}

build
if [ -f "$tmpFolder/mygit" ]; then
    echo "Build successful"
else
    echo "Build failed"
    exit 1
fi

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