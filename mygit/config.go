package mygit

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"time"
)

// https://git-scm.com/book/en/v2/Git-Internals-Environment-Variables

func currentUserName() string {
	user, err := user.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: cannot get current user")
		return ""
	}
	return user.Username
}

func getUTCOffset(t time.Time) string {
	_, offset := t.Zone()
	offset_hours := offset / 3600
	offset_minutes := (offset % 3600) / 60
	if offset_minutes < 0 {
		offset_hours = -offset_hours
	}
	offset_data := fmt.Sprintf("%02d%02d", offset_hours, offset_minutes)
	if offset_hours >= 0 {
		offset_data = "+" + offset_data
	}
	return offset_data
}

func currentDate() string {
	// always display in UTC
	// https://git-scm.com/docs/git-commit/2.24.0#_date_formats
	t := time.Now()
	offset := getUTCOffset(t)
	return strconv.FormatInt(t.UTC().Unix(), 10) + " " + offset
}

func getAuthorName() string {
	authorName := os.Getenv("GIT_AUTHOR_NAME")
	if authorName == "" {
		authorName = currentUserName()
	}
	if authorName == "" {
		authorName = "mygit"
	}
	return authorName
}

func getAuthorEmail() string {
	authorEmail := os.Getenv("GIT_AUTHOR_EMAIL")
	if authorEmail == "" {
		authorEmail = "mygit"
	}
	return authorEmail
}

func getAuthorDate() string {
	authorDate := os.Getenv("GIT_AUTHOR_DATE")
	if authorDate == "" {
		authorDate = currentDate()
	}
	return authorDate
}

func getCommitterName() string {
	committerName := os.Getenv("GIT_COMMITTER_NAME")
	if committerName == "" {
		committerName = getAuthorName()
	}
	if committerName == "" {
		committerName = "mygit"
	}
	return committerName
}

func getCommitterEmail() string {
	commiterEmail := os.Getenv("GIT_COMMITTER_EMAIL")
	if commiterEmail == "" {
		commiterEmail = getAuthorEmail()
	}
	if commiterEmail == "" {
		commiterEmail = "mygit"
	}
	return commiterEmail
}

func getCommitterDate() string {
	committerDate := os.Getenv("GIT_COMMITTER_DATE")
	if committerDate == "" {
		committerDate = currentDate()
	}
	return committerDate
}
