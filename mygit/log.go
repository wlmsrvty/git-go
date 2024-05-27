package mygit

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Prints the commit history of the given object ID
func Log(oid string) error {
	if oid == "" {
		var err error
		oid, err = getHeadOID()
		if err != nil {
			return err
		}
	}

	object, err := NewObject(oid)
	if err != nil {
		return err
	}

	if object.Type != ObjectTypeCommit {
		return fmt.Errorf("%s is not a commit object", oid)
	}

	commitObject, err := parseCommitObject(object)
	if err != nil {
		return err
	}

	commits := []*CommitObject{commitObject}

	for len(commits) > 0 {
		commit := commits[0]
		commits = commits[1:]

		if err := displayCommit(commit); err != nil {
			return err
		}

		for _, parent := range commit.Parents {
			parentObject, err := NewObject(parent)
			if err != nil {
				return err
			}

			if parentObject.Type != ObjectTypeCommit {
				return fmt.Errorf("%s is not a commit object", parent)
			}

			parentCommitObject, err := parseCommitObject(parentObject)
			if err != nil {
				return err
			}

			commits = append(commits, parentCommitObject)
		}
	}

	return nil
}

func displayCommit(commit *CommitObject) error {
	fmt.Printf("commit %s\n", commit.Hash)
	fmt.Printf("Author:\t%s <%s>\n", commit.AuthorName, commit.AuthorEmail)

	// format date
	i, err := strconv.ParseInt(commit.AuthorDateSeconds, 10, 64)
	if err != nil {
		return err
	}
	timeZone, err := strconv.ParseInt(commit.AuthorDateTimeZone, 10, 64)
	if err != nil {
		return err
	}
	negative := 1
	if timeZone < 0 {
		negative = -1
		timeZone = -timeZone
	}
	hours := int(timeZone / 100)
	minutes := int(timeZone % 100)
	tm := time.Unix(i, 0).In(time.FixedZone("", (hours*60*60+minutes*60)*negative))
	fmt.Printf("Date: \t%s %s\n", tm.Format(time.ANSIC), commit.AuthorDateTimeZone)

	fmt.Printf("\n\t%s\n", strings.ReplaceAll(commit.Message, "\n", "\n\t"))

	return nil
}

func getHeadOID() (string, error) {
	// get HEAD commit
	data, err := os.ReadFile(".git/HEAD")
	if err != nil {
		return "", err
	}
	ref := strings.Split(string(data), " ")[1]
	ref = strings.TrimSpace(ref)
	data, err = os.ReadFile(".git/" + ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
