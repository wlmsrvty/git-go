package mygit

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const (
	smartRefDiscoveryPath = "/info/refs?service=git-upload-pack"
	gitUploadPackPath     = "/git-upload-pack"
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

	if len(remoteRefs.refs) == 0 {
		return fmt.Errorf("no refs found in remote repository")
	}

	// create negotation request
	negotationRequest := createNegotationRequest("", remoteRefs.refs)

	// get the packfile
	requestReader := strings.NewReader(negotationRequest)
	resp, err := http.Post(url+"/git-upload-pack", "application/x-git-upload-pack-request", requestReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error /git-upload-pack: %s", resp.Status)
	}

	// create repo
	repoName := path.Base(url)
	err = createRepo(repoName)
	if err != nil {
		return err
	}

	// save packfile
	firstRef := remoteRefs.refs[0]
	tempPackFileName := fmt.Sprintf("tmp_pack_%.5s", firstRef.ObjectId)
	tempPackFilePath := path.Join(repoName, ".git", "objects", "pack", tempPackFileName)

	tempPackFile, err := os.Create(tempPackFilePath)
	if err != nil {
		return err
	}
	defer tempPackFile.Close()

	// skip "NAK" packet line
	// "0008NAK\n"
	// the rest corresponds to the packfile
	_, err = resp.Body.Read(make([]byte, 8))
	if err != nil {
		return err
	}

	_, err = io.Copy(tempPackFile, resp.Body)
	if err != nil {
		return err
	}

	// parse packfile
	tempPackFile.Seek(0, 0)
	packFile, err := io.ReadAll(tempPackFile)
	if err != nil {
		return err
	}

	verison, numObjects, err := parsePackfileHeader(packFile)
	if err != nil {
		return err
	}

	err = verifyPackFileChecksum(packFile)
	if err != nil {
		return err
	}

	fmt.Println(verison, numObjects)

	return nil
}

type remoteRefs struct {
	refs []*ref
	cap  capabilities
}

// https://git-scm.com/docs/http-protocol#_smart_clients
func discoverRefsSmartHttp(url string) (*remoteRefs, error) {
	resp, err := http.Get(url + smartRefDiscoveryPath)
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

type ref struct {
	ObjectId string
	Name     string
}

type capabilities string

func parseRef(buf []byte) (*ref, capabilities, error) {
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
	cap = strings.TrimSpace(cap)

	return &ref{
		ObjectId: objectId,
		Name:     name,
	}, capabilities(cap), nil
}

// ================= Pack file negotation =================

// constructs a request in the form:
//
//	```
//		0077want 8c25759f3c2b14e9eab301079c8b505b59b3e1ef multi_ack_detailed side-band-64k thin-pack ofs-delta agent=git/1.8.2
//		0032want 8c25759f3c2b14e9eab301079c8b505b59b3e1ef
//		0032want 4574b4c7bb073b6b661abd0558a639f7a32b3f8f
//	```
func createNegotationRequest(cap capabilities, reflist []*ref) string {
	if len(reflist) == 0 {
		return ""
	}
	firstLine := fmt.Sprintf("want %s %s\n", reflist[0].ObjectId, cap)
	var sb strings.Builder
	sb.WriteString(toPktLine(firstLine))
	for _, ref := range reflist[1:] {
		value := fmt.Sprintf("want %s\n", ref.ObjectId)
		sb.WriteString(toPktLine(value))
	}

	const flushPkt = "0000"
	sb.WriteString(flushPkt)
	sb.WriteString(toPktLine("done\n"))

	return sb.String()
}

// ======================== Pack file parsing ========================

// Parses the packfile header
// returns version, number of objects
func parsePackfileHeader(packFile []byte) (uint32, uint32, error) {
	packFileBuffer := bytes.NewReader(packFile)

	magic := make([]byte, 4)
	_, err := packFileBuffer.Read(magic)
	if err != nil {
		return 0, 0, err
	}
	if string(magic) != "PACK" {
		return 0, 0, fmt.Errorf("invalid packfile header: %s", magic)
	}

	versionBytes := make([]byte, 4)
	_, err = packFileBuffer.Read(versionBytes)
	if err != nil {
		return 0, 0, err
	}
	version := binary.BigEndian.Uint32(versionBytes)

	numObjectsBytes := make([]byte, 4)
	_, err = packFileBuffer.Read(numObjectsBytes)
	if err != nil {
		return 0, 0, err
	}

	numObjects := binary.BigEndian.Uint32(numObjectsBytes)

	return version, numObjects, nil
}

func verifyPackFileChecksum(packFile []byte) error {
	checksum := packFile[len(packFile)-20:]
	packFileContent := packFile[:len(packFile)-20]

	computedChecksum := sha1.Sum(packFileContent)

	if !bytes.Equal(checksum, computedChecksum[:]) {
		return fmt.Errorf("invalid packfile checksum")
	}

	return nil
}

// ======================== Helpers ========================

func toPktLine(value string) string {
	return fmt.Sprintf("%04x%s", len(value)+4, value)
}

func createRepo(repoName string) error {
	err := os.MkdirAll(path.Join(repoName, ".git", "objects", "pack"), 0755)
	if err != nil {
		return err
	}

	return nil
}
