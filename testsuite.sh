#!/bin/sh

tmpFolder=$(mktemp -d)
echo Testsuite using $tmpFolder

build() {
    ( cd $(dirname "$0") &&
	    go build -buildvcs="false" -o "$tmpFolder/mygit" ./cmd/mygit )
}

test_init() {
    echo "=== Running test_init ==="
    (
        cd $tmpFolder
        ./mygit init > /dev/null
        if [ ! -d ".git" ]; then
            echo "[KO] no .git folder"
            exit 1
        fi
        if [ ! -d ".git/objects" ]; then
            echo "[KO] no .git/objects folder"
            exit 1
        fi
        if [ ! -d ".git/refs" ]; then
            echo "[KO] no .git/refs folder"
            exit 1
        fi
    ) && echo "[OK]"
}

test_cat_file() {
    echo "=== Running test_cat_file ==="
    (
        cd $tmpFolder
        test_text="test content"
        hash=$(echo "$test_text" | git hash-object -w --stdin)

        echo $test_text > ref.txt
        ./mygit cat-file -p $hash > got.txt
        diff -u ref.txt got.txt
        if [ $? -ne 0 ]; then
            echo "[KO] cat-file failed"
            exit 1
        fi
    ) && echo "[OK]"
}

test_hash_object() {
    echo "=== Running test_hash_object ==="
    (
        cd $tmpFolder
        test_text="test content"
        echo $test_text > content.md

        echo "d670460b4b4aece5915caf5c68d12f560a9fe3e4" > ref.txt
        ./mygit hash-object -w content.md > got.txt
        diff -u ref.txt got.txt
        if [ $? -ne 0 ]; then
            echo "[KO] hash-object failed"
            exit 1
        fi

        echo "eAFLyslPUjA0ZihJLS5RSM7PK0nNK+ECAEvfBwk=" | base64 -d > ref.txt
        diff -u ref.txt .git/objects/d6/70460b4b4aece5915caf5c68d12f560a9fe3e4
        if [ $? -ne 0 ]; then
            echo "[KO] hash-object failed"
            exit 1
        fi
    ) && echo "[OK]"
}

build
test_init
test_cat_file
test_hash_object