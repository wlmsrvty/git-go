mygit=mygit
return_code=0

$mygit init

if [ ! -d ".git" ]; then
    echo "[KO] no .git folder"
    exit 1
else
    echo "[OK] .git/ present"
fi
if [ ! -d ".git/objects" ]; then
    echo "[KO] no .git/objects folder"
    exit 1
else
    echo "[OK] .git/objects present"
fi
if [ ! -d ".git/refs" ]; then
    echo "[KO] no .git/refs folder"
    exit 1
else
    echo "[OK] .git/refs present"
fi