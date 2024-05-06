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
commit=$(git commit-tree $tree -m "Initial commit")
git commit-tree -m "Second commit" -p $commit $tree > $result/ref_commit_tree.txt

rm -rf .git

$mygit init
$mygit write-tree
commit=$($mygit commit-tree -m "Initial commit" $tree)
git commit-tree -m "Second commit" -p $commit $tree > $result/got_commit_tree.txt

diff -u $result/ref_commit_tree.txt $result/got_commit_tree.txt
if [ $? -ne 0 ]; then
    echo "[KO] commit-tree failed"
    exit 1
else
    echo "[OK] good output"
fi
