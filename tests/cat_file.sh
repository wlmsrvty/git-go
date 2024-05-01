mygit=mygit

$mygit init

test_text="test content"

hash=$(echo "$test_text" | git hash-object -w --stdin)

echo $test_text > ref.txt

$mygit cat-file -p $hash > got.txt

diff -u ref.txt got.txt
if [ $? -ne 0 ]; then
    echo "[KO] cat-file failed"
    exit 1
else
    echo "[OK] good output"
fi