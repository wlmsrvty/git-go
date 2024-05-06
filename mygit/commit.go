package mygit

import (
	"bytes"
	"crypto/sha1"
	"fmt"
)

func CommitTree(treeSha string, parentCommit string, commitMessage string) error {
	object, err := NewObject(treeSha)
	if err != nil {
		return err
	}

	if object.Type != ObjectTypeTree {
		return fmt.Errorf("given object is not a tree")
	}

	// TODO: check parentCommit object
	var parentCommitObject *Object = nil
	if parentCommit != "" {
		parentCommitObject, err = NewObject(parentCommit)
		if err != nil {
			return err
		}

		if parentCommitObject.Type != ObjectTypeCommit {
			return fmt.Errorf("given parent object is not a commit")
		}
	}

	// commit-object format: https://stackoverflow.com/questions/22968856/what-is-the-file-format-of-a-git-commit-object-data-structure

	commitContent := bytes.Buffer{}

	// tree <tree_sha>
	commitContent.WriteString(fmt.Sprintf("tree %s\n", treeSha))

	// parent <parent_commit>
	if parentCommit != "" {
		commitContent.WriteString(fmt.Sprintf("parent %s\n", parentCommit))
	}

	// author
	authorName := getAuthorName()
	if authorName == "" {
		return fmt.Errorf("user name not set")
	}
	authorEmail := getAuthorEmail()
	if authorEmail == "" {
		return fmt.Errorf("user email not set")
	}
	authorText := fmt.Sprintf("author %s <%s> %s\n", authorName, authorEmail, getAuthorDate())
	commitContent.WriteString(authorText)

	// committer
	committerName := getCommitterName()
	if committerName == "" {
		return fmt.Errorf("committer name not set")
	}
	commiterEmail := getCommitterEmail()
	if commiterEmail == "" {
		return fmt.Errorf("committer email not set")
	}
	committerText := fmt.Sprintf("committer %s <%s> %s\n", committerName, commiterEmail, getCommitterDate())
	commitContent.WriteString(committerText)

	commitContent.WriteString("\n")
	commitContent.WriteString(commitMessage)
	commitContent.WriteString("\n")

	commitContentBytes := commitContent.Bytes()

	commitRaw := bytes.Buffer{}
	commitRaw.WriteString(fmt.Sprintf("commit %d\x00", commitContent.Len()))
	commitRaw.Write(commitContentBytes)

	commitRawBytes := commitRaw.Bytes()

	hash := sha1.New()
	hash.Write(commitRawBytes)
	hashString := fmt.Sprintf("%x", hash.Sum(nil))

	writeAnyObject(hashString, commitRawBytes)

	fmt.Println(hashString)

	return nil
}
