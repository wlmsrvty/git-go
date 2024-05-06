mygit=mygit

print_object() {
    for object in .git/objects/**/*; do
        [ -f $object ] && echo $object && cat $object | python3 -c "import zlib;import sys; a = sys.stdin.buffer.read();print(zlib.decompress(a))" && echo
    done
}

config() {
    export GIT_AUTHOR_EMAIL=wlmsrvty@william.ovh
    export GIT_AUTHOR_NAME=wlmsrvty

    export GIT_COMMITTER_EMAIL=wlmsrvty@william.ovh
    export GIT_COMMITTER_NAME=wlmsrvty

    export GIT_AUTHOR_DATE="1715028250 +0200"
    export GIT_COMMITTER_DATE="1715028250 +0200"
}

prepare() {
    echo "hello world" > hello.txt
}

config

prepare

result=$(mktemp -d)

git init > /dev/null 2>&1
git add hello.txt
tree=$(git write-tree)
git commit-tree $tree -m "Initial commit" > $result/ref_commit_tree.txt

print_object > $result/ref_objects.txt

rm -rf .git

$mygit init
$mygit write-tree
$mygit commit-tree -m "Initial commit" $tree  > $result/got_commit_tree.txt

print_object > $result/got_objects.txt

diff -u $result/ref_objects.txt $result/got_objects.txt
if [ $? -ne 0 ]; then
    echo "[KO] commit-tree failed"
    exit 1
else
    echo "[OK] good objects"
fi


diff -u $result/ref_commit_tree.txt $result/got_commit_tree.txt
if [ $? -ne 0 ]; then
    echo "[KO] commit-tree failed"
    exit 1
else
    echo "[OK] good output"
fi
