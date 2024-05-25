package mygit

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	smartRefDiscoveryURL = "/info/refs?service=git-upload-pack"
)

// Lists references in a remote repository
func DisplayRemoteRefs(url string) error {
	url = sanitizeURL(url)
	remoteRefs, err := discoverRefsSmartHttp(url)
	if err != nil {
		return err
	}

	for _, ref := range remoteRefs.refs {
		fmt.Printf("%s\t%s\n", ref.ObjectId, ref.Name)
	}

	return nil
}

// Clone clones a repository into a new directory
// in the current working directory
func Clone(url string) error {
	url = sanitizeURL(url)
	remoteRefs, err := discoverRefsSmartHttp(url)
	if err != nil {
		return err
	}

	for _, ref := range remoteRefs.refs {
		fmt.Println(ref)
	}

	return nil
}

type remoteRefs struct {
	refs []*Ref
	cap  Cap
}

// https://git-scm.com/docs/http-protocol#_smart_clients
func discoverRefsSmartHttp(url string) (*remoteRefs, error) {
	resp, err := http.Get(url + smartRefDiscoveryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	reader := bufio.NewReader(resp.Body)

	// first pkt-line, service name
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// validate first five bytes
	if err := valideFirstFiveBytes(firstLine); err != nil {
		return nil, err
	}

	// validate service
	if err := validateService(firstLine); err != nil {
		return nil, err
	}

	// second line
	prefix := make([]byte, 4)
	_, err = io.ReadFull(reader, prefix)
	if err != nil {
		return nil, err
	}
	if string(prefix) != "0000" {
		return nil, fmt.Errorf("invalid pkt-line: %v", prefix)
	}

	var remoteRefs remoteRefs

	buf, err := readPktLine(reader)
	for err == nil {
		ref, cap, parseErr := parseRef(buf)
		if parseErr != nil {
			return nil, err
		}
		if cap != "" {
			remoteRefs.cap = cap
		}
		remoteRefs.refs = append(remoteRefs.refs, ref)
		buf, err = readPktLine(reader)
	}
	if err != ErrPktFlush {
		return nil, err
	}

	return &remoteRefs, nil
}

func pktLineValue(line string) (string, error) {
	// size := line[:4]
	value := line[4:]

	// Perform a check on the size

	return value, nil
}

// ======================== Validation ========================

func valideFirstFiveBytes(line string) error {
	match, err := regexp.MatchString("^[0-9a-f]{4}#", line)
	if err != nil {
		return err
	}
	if !match {
		return fmt.Errorf("invalid pkt-line: %s", line)
	}
	return nil
}

func validateService(pktLine string) error {
	value, err := pktLineValue(pktLine)
	if err != nil {
		return err
	}

	regexServiceName := regexp.MustCompile(`^# service=([-\w]+)`)
	matches := regexServiceName.FindStringSubmatch(value)
	if len(matches) < 2 {
		return fmt.Errorf("got invalid service name: %s", value)
	}
	serviceName := matches[1]

	if serviceName != "git-upload-pack" {
		return fmt.Errorf("got wrong service from server: %s", serviceName)
	}

	return nil
}

// ======================== Pkt-line reading ========================

var ErrPktFlush = fmt.Errorf("pkt-line flush")

func readPktLine(reader *bufio.Reader) ([]byte, error) {
	prefixBuf := make([]byte, 4)

	_, err := io.ReadFull(reader, prefixBuf)
	if err != nil {
		return nil, err
	}

	// Convert size to int
	size, err := strconv.ParseInt(string(prefixBuf), 16, 32)
	if err != nil {
		return nil, err
	}

	if size == 0 {
		return nil, ErrPktFlush
	}

	// subtract 4 bytes for the prefix itself
	size -= 4

	buf := make([]byte, size)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// ======================== Refs parsing ========================

type Ref struct {
	ObjectId string
	Name     string
}

type Cap string

func parseRef(buf []byte) (*Ref, Cap, error) {
	readerBytes := bytes.NewReader(buf)
	reader := bufio.NewReader(readerBytes)

	// Read the object ID
	objectId, err := reader.ReadString(' ')
	if err != nil {
		return nil, "", err
	}

	// Read the ref name
	name, err := reader.ReadString('\x00')
	if err != nil && err != io.EOF {
		return nil, "", err
	}

	// Read the capabilities if any
	cap, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, "", err
	}

	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\x00", "")

	objectId = strings.TrimSpace(objectId)

	return &Ref{
		ObjectId: objectId,
		Name:     name,
	}, Cap(cap), nil
}
