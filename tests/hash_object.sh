mygit=mygit
return_code=0

$mygit init

test_text="test content"
echo $test_text > content.md

echo "d670460b4b4aece5915caf5c68d12f560a9fe3e4" > ref.txt

$mygit hash-object -w content.md > got.txt
diff -u ref.txt got.txt
if [ $? -ne 0 ]; then
    echo "[KO] not the same hash"
    exit 1
else
    echo "[OK] same hash"
fi

cp .git/objects/d6/70460b4b4aece5915caf5c68d12f560a9fe3e4 got
git hash-object -w content.md > /dev/null
cp .git/objects/d6/70460b4b4aece5915caf5c68d12f560a9fe3e4 ref

diff -u ref got
if [ $? -ne 0 ]; then
    echo "[KO] object file written not correct"
    exit 1
else
    echo "[OK] same object file written"
fi

exit $return_code