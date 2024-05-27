mygit=mygit

repo='https://github.com/codecrafters-io/git-sample-1'

# git reference uses packfile with index rather than
# unpacking to objects as done in this implementation
# so we test only files and directories and not .git/

$mygit clone $repo got_repo
rm -rf got_repo/.git

git clone $repo ref_repo
rm -rf ref_repo/.git

diff -ar got_repo ref_repo
if [ $? -ne 0 ]; then
    echo "[KO] clone failed"
    exit 1
else
    echo "[OK] clone: same files and directories"
fi