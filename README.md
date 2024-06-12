# git-go

A small [Git](https://www.git-scm.com/) implementation written in [Go](https://go.dev/).

The main goals of this project were to learn about Git internals and Go. This small Git implementation is capable of initializing a repository, creating commits and cloning a public repository.

## Features

### Commands

Basic commands:
- `init`:        Initialize the git directory structure
- `commit`:      Record changes to the repository
- `log`:         Show commit logs for a commit ID

Plumbing commands:
- `cat-file`:    Provide content or type and size information for repository objects
- `hash-object`: Compute object ID and optionally creates a blob from a file
- `ls-tree`: 	List the contents of a tree object
- `write-tree`: 	Create a tree object from the current working directory
- `commit-tree`: Create a new commit object

Remote commands:
- `clone`:       Clone a repository into a new directory
- `ls-remote`:   List references in a remote repository

### Clone

Clone command is implemented using the smart HTTP protocol only. What it does:
1. Discover references in the remote repository
2. Negotiate packfile transfer
3. Transfer and unpack packfile (with deltas unpacking)

Clone uses loose objects, unpacking the packfile fully to `.git/objects`.

## Resources

- [Pro Git Book](https://git-scm.com/book/en/v2)
- [Git Internals Plumbing and Porcelain](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain)
- [Git Internals - git objects](https://git-scm.com/book/en/v2/Git-Internals-Git-Objects#_git_commit_objects)
- [Git’s database internals I: packed object store](https://github.blog/2022-08-29-gits-database-internals-i-packed-object-store/)
- [Git Smart HTTP Transfer protocol doc](https://www.git-scm.com/docs/http-protocol)
- [Git Smart HTTP Transfer protocol stackoverflow](https://stackoverflow.com/questions/68062812/what-does-the-git-smart-https-protocol-fully-look-like-in-all-its-glory)
- [Git clone in haskell from the bottom up](https://stefan.saasen.me/articles/git-clone-in-haskell-from-the-bottom-up/#reimplementing-git-clone-in-haskell-from-the-bottom-up)
- [Unpacking git packfiles](https://codewords.recurse.com/issues/three/unpacking-git-packfiles)
- [gitprotocol-pack.txt](https://github.com/git/git/blob/795ea8776befc95ea2becd8020c7a284677b4161/Documentation/gitprotocol-pack.txt)
- `man gitformat-pack`
- [Git from the Bottom Up](https://jwiegley.github.io/git-from-the-bottom-up/)
- [CodeCrafters Challenge](https://app.codecrafters.io/courses/git/overview)