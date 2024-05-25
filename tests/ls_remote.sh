mygit=mygit
return_code=0

repo='https://github.com/LazyVim/starter'

echo "testing in $PWD"

git ls-remote $repo > ref

$mygit ls-remote $repo > got

diff -u ref got
if [ $? -ne 0 ]; then
    return_code=1
    result="[KO]"
else
    result="[OK]"
fi
echo "$result ls-remote test"

exit $return_code
