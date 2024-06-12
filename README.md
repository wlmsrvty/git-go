# git-go

![Build Status](https://github.com/wlmsrvty/git-go/actions/workflows/build.yml/badge.svg)

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

## Build and test

### Build

```bash
$ go build -o git-go
$ ./git-go
Usage: git-go <command> [<args>...]

Commands:
    init        Initialize the git directory structure
    cat-file    Provide content or type and size information for repository objects
    hash-object Compute object ID and optionally creates a blob from a file
    ls-tree     List the contents of a tree object
    write-tree  Create a tree object from the current working directory
    commit-tree Create a new commit object
    clone       Clone a repository into a new directory
    ls-remote   List references in a remote repository
    log         Show commit logs for a commit ID
    commit      Record changes to the repository
```

### Test

```bash
$ ./testsuite.sh
Build successful
=== Running cat_file.sh ===
Initialized empty Git repository in /tmp/tmp.X9EiYylEQ5/.git/
[OK] good output
=> tests/cat_file.sh passed
...
```

## Example

```bash
$ mkdir test_repo
$ cd test_repo
$ git-go init
Initialized empty Git repository in /tmp/tmp.XpIeQSf7Ns/test_repo/.git/
$ git-go commit -m "first"
[70dbd5eedb3c888ee6be6f1dc00e45f190a8a856] first
$ git-go log
commit 70dbd5eedb3c888ee6be6f1dc00e45f190a8a856
Author:	william <mygit>
Date: 	Thu Jun 13 00:11:03 2024 +0200

	first

$ git-go cat-file -p 70dbd5eedb3c888ee6be6f1dc00e45f190a8a856
tree 581caa0fe56cf01dc028cc0b089d364993e046b6
author william <mygit> 1718230263 +0200
committer william <mygit> 1718230263 +0200

first
$ echo 'test' > test.txt
$ git-go hash-object test.txt
9daeafb9864cf43055ae93beb0afd6c7d144bfa4
$ git-go ls-tree 581caa0fe56cf01dc028cc0b089d364993e046b6
100644 blob 980a0d5f19a64b4b30a87d4206aade58726b60e3	hello.txt
$ cd $(mktemp -d)
$ git-go init
Initialized empty Git repository in /tmp/tmp.rl0ccY5hyN/.git/
$ echo 'hello world' > hello.txt
$ echo 'test' > test.txt
$ echo 'draft' > draft
$ git-go write-tree
16e7f5e6e626c055a249af61ff9631f863f0b5a7
$ cd $(mktemp -d)
$ git-go clone https://github.com/githubtraining/hellogitworld
Cloning into 'hellogitworld'...
remote: Number of objects: 1147
remote: Resolving deltas: 418
$ ls hellogitworld
build.gradle  fix.txt  pom.xml  README.txt  resources  runme.sh  src
```

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
