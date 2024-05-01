mygit=mygit
return_code=0

$mygit init

echo "Hello World" > hello_world.txt

$mygit hash-object -w hello_world.txt

git add hello_world.txt

hash=$(git write-tree)

$mygit ls-tree --name-only $hash > got.txt

git ls-tree --name-only $hash > ref.txt

diff -u ref.txt got.txt
if [ $? -ne 0 ]; then
    return_code=1
    result="[KO]"
else
    result="[OK]"
fi
echo "$result ls-tree output test"

exit $return_code
