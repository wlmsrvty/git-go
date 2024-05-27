package mygit

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"regexp"
	"strings"
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

type CommitObject struct {
	Hash    string
	Tree    string
	Parents []string

	AuthorName         string
	AuthorEmail        string
	AuthorDateSeconds  string
	AuthorDateTimeZone string

	CommitterName         string
	CommitterEmail        string
	CommitterDateSeconds  string
	CommitterDateTimeZone string

	Message string
}

func parseCommitObject(object *Object) (*CommitObject, error) {
	// commit object format
	// https://stackoverflow.com/questions/22968856/what-is-the-file-format-of-a-git-commit-object-data-structure

	content := object.Content
	buffer := bytes.NewBuffer(content)
	prefix, err := buffer.ReadString(' ')
	if err != nil {
		return nil, err
	}
	if prefix != "tree " {
		return nil, fmt.Errorf("invalid commit object")
	}

	tree, err := buffer.ReadString('\n')
	if err != nil {
		return nil, err
	}

	line, err := buffer.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// parents
	parents := []string{}
	for {
		if !strings.HasPrefix(line, "parent ") {
			break
		}
		parent := strings.Split(line, " ")[1]
		parent = strings.TrimSpace(parent)
		parents = append(parents, parent)

		line, err = buffer.ReadString('\n')
		if err != nil {
			return nil, err
		}
	}

	// author
	// author {author_name} <{author_email}> {author_date_seconds} {author_date_timezone}
	if !strings.HasPrefix(line, "author ") {
		return nil, fmt.Errorf("invalid commit object")
	}
	const authorRegex = `author ([^<]+) <([^>]+)> (\d+) (.*)`
	re := regexp.MustCompile(authorRegex)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 5 {
		return nil, fmt.Errorf("invalid commit objet: error parsing author line")
	}
	authorName := matches[1]
	authorEmail := matches[2]
	authorDateSeconds := matches[3]
	authorDateTimeZone := matches[4]

	line, err = buffer.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// committer
	// committer {committer_name} <{committer_email}> {committer_date_seconds} {committer_date_timezone}
	if !strings.HasPrefix(line, "committer ") {
		return nil, fmt.Errorf("invalid commit object")
	}
	const committerRegex = `committer ([^<]+) <([^>]+)> (\d+) (.*)`
	re = regexp.MustCompile(committerRegex)
	matches = re.FindStringSubmatch(line)
	if len(matches) < 5 {
		return nil, fmt.Errorf("invalid commit objet: error parsing committer line")
	}
	committerName := matches[1]
	committerEmail := matches[2]
	committerDateSeconds := matches[3]
	committerDateTimeZone := matches[4]

	line, err = buffer.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line != "\n" {
		return nil, fmt.Errorf("invalid commit object")
	}

	// commit message
	bufMessage := bytes.NewBuffer(nil)
	io.Copy(bufMessage, buffer)
	message := bufMessage.String()

	return &CommitObject{
		Hash:    object.Hash,
		Tree:    tree[:len(tree)-1],
		Parents: parents,

		AuthorName:         authorName,
		AuthorEmail:        authorEmail,
		AuthorDateSeconds:  authorDateSeconds,
		AuthorDateTimeZone: authorDateTimeZone,

		CommitterName:         committerName,
		CommitterEmail:        committerEmail,
		CommitterDateSeconds:  committerDateSeconds,
		CommitterDateTimeZone: committerDateTimeZone,

		Message: message,
	}, nil
}
