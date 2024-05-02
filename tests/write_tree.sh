mygit=mygit

print_object() {
    for object in .git/objects/**/*; do
        [ -f $object ] && echo $object && cat $object | python3 -c "import zlib;import sys; a = sys.stdin.buffer.read();print(zlib.decompress(a))" && echo
    done
}

prepare() {
    echo "hello world" > hello.txt
    mkdir merci
    echo "test merci" > merci/test.txt
}

ref_result=$(mktemp -d)

git init > /dev/null 2>&1
git add .
git write-tree > $ref_result/ref_command.txt
print_object > $ref_result/ref_print_object.txt

rm -rf .git

got_result=$(mktemp -d)

$mygit init
$mygit write-tree > $got_result/got_command.txt
print_object > $got_result/got_print_object.txt

diff -u $ref_result/ref_command.txt $got_result/got_command.txt
if [ $? -ne 0 ]; then
    echo "[KO] write-tree failed"
    exit 1
else
    echo "[OK] good output"
fi

diff -u $ref_result/ref_print_object.txt $got_result/got_print_object.txt
if [ $? -ne 0 ]; then
    echo "[KO] print-object failed"
    exit 1
else
    echo "[OK] good output"
fi

$mygit init

