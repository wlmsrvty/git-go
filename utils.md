List and print every object in the git database:
```bash
for object in .git/objects/**/*; do
    [ -f $object ] && echo $object && cat $object | python3 -c "import zlib;import sys; a = sys.stdin.buffer.read();print(zlib.decompress(a))" && echo
done
```


