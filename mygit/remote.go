package mygit

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const master = "master"

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
func Clone(url string, repoName string) error {
	if repoName == "" {
		repoName = path.Base(url)
	}

	fmt.Printf("Cloning into '%s'...\n", repoName)

	// create repo
	err := os.MkdirAll(repoName, 0755)
	if err != nil {
		return err
	}
	err = os.Chdir(repoName)
	if err != nil {
		return err
	}
	// init repo
	err = createInitStructure(noPrint)
	if err != nil {
		return err
	}

	url = sanitizeURL(url)
	remoteRefs, err := discoverRefsSmartHttp(url)
	if err != nil {
		return err
	}

	err = writeRefsToDisk(remoteRefs.refs)
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

	// save packfile
	firstRef := remoteRefs.refs[0]
	tempPackFileName := fmt.Sprintf("tmp_pack_%.5s", firstRef.ObjectId)
	err = os.MkdirAll(path.Join(".git", "objects", "pack"), 0755)
	if err != nil {
		return err
	}
	tempPackFilePath := path.Join(".git", "objects", "pack", tempPackFileName)

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

	_, numObjects, err := parsePackFileHeader(packFile)
	if err != nil {
		return err
	}

	err = verifyPackFileChecksum(packFile)
	if err != nil {
		return err
	}

	fmt.Printf("remote: Number of objects: %d\n", numObjects)

	err = parsePackFile(packFile, numObjects)
	if err != nil {
		return err
	}

	// checkout the HEAD ref and write files to disk
	headRef := remoteRefs.refs[0]
	object, err := NewObject(headRef.ObjectId)
	if err != nil {
		return err
	}

	commitObject, err := parseCommitObject(object.Content)
	if err != nil {
		return err
	}

	treeObject, err := NewObject(commitObject.Tree)
	if err != nil {
		return err
	}
	err = writeTreeToDisk(treeObject, ".")
	if err != nil {
		return err
	}

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

func writeRefsToDisk(refs []*ref) error {
	// remotes/origin/HEAD
	err := os.MkdirAll(".git/refs/remotes/origin/", 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(".git/refs/remotes/origin/HEAD",
		[]byte("ref: refs/remotes/origin/"+master+"\n"), 0644)
	if err != nil {
		return err
	}

	// tags
	err = os.MkdirAll(".git/refs/tags", 0755)
	if err != nil {
		return err
	}

	for _, ref := range refs {
		if ref.Name != "HEAD" {
			refPath := path.Join(".git", ref.Name)
			folder := filepath.Dir(refPath)
			err := os.MkdirAll(folder, 0755)
			if err != nil {
				return err
			}
			err = os.WriteFile(refPath, []byte(ref.ObjectId+"\n"), 0644)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
func parsePackFileHeader(packFile []byte) (uint32, uint32, error) {
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

func parsePackFile(packFile []byte, numberObjects uint32) error {
	packFileBuffer := bytes.NewReader(packFile)

	// skip header (12 bytes)
	packFileBuffer.Seek(12, 0)

	var deltaObjects []deltaObject

	// read object entries
	var i uint32
	for i = 0; i < numberObjects; i++ {
		// read object header
		size, objectType, err := parseObjectHeader(packFileBuffer)
		if err != nil {
			return err
		}

		// commit, tag, tree or blob
		if objectType == OBJ_COMMIT ||
			objectType == OBJ_TAG ||
			objectType == OBJ_TREE ||
			objectType == OBJ_BLOB {

			object, err := parseObject(packFileBuffer)
			if err != nil {
				return err
			}

			if uint32(len(object)) != size {
				return fmt.Errorf("pack file object size mismatch")
			}

			objectTypeStr := PackFileObjectTypeString[objectType]

			// write object to disk
			err = writePackFileObject(objectTypeStr, object)
			if err != nil {
				return err
			}
		} else if objectType == OBJ_OFS_DELTA {
			size, err := parseSize(packFileBuffer)
			if err != nil {
				return err
			}
			object, err := parseObject(packFileBuffer)
			if err != nil {
				return err
			}
			if uint32(len(object)) != size {
				return fmt.Errorf("pack file %s object size mismatch",
					PackFileObjectTypeString[objectType])
			}

			// TODO: OBJECT OFFSET DELTA
			panic("not implemented")

		} else if objectType == OBJ_REF_DELTA {
			hash := make([]byte, 20)
			_, err := packFileBuffer.Read(hash)
			if err != nil {
				return err
			}

			object, err := parseObject(packFileBuffer)
			if err != nil {
				return err
			}

			if uint32(len(object)) != size {
				return fmt.Errorf("pack file %s object size mismatch",
					PackFileObjectTypeString[objectType])
			}

			deltaObjects = append(deltaObjects, deltaObject{
				baseObject: fmt.Sprintf("%x", hash),
				data:       object,
			})

		} else {
			return fmt.Errorf("invalid object type: %d", objectType)
		}
	}

	if len(deltaObjects) > 0 {
		err := applyDeltas(deltaObjects)
		if err != nil {
			return err
		}
	}

	return nil
}

// ======================== Object parsing ========================

type PackFileObjectType int

const (
	OBJ_COMMIT    PackFileObjectType = 1
	OBJ_TREE      PackFileObjectType = 2
	OBJ_BLOB      PackFileObjectType = 3
	OBJ_TAG       PackFileObjectType = 4
	OBJ_OFS_DELTA PackFileObjectType = 6
	OBJ_REF_DELTA PackFileObjectType = 7
)

var PackFileObjectTypeString = map[PackFileObjectType]string{
	OBJ_COMMIT:    "commit",
	OBJ_TREE:      "tree",
	OBJ_BLOB:      "blob",
	OBJ_TAG:       "tag",
	OBJ_OFS_DELTA: "ofs-delta",
	OBJ_REF_DELTA: "ref-delta",
}

func parseObjectHeader(packFileBuffer *bytes.Reader) (size uint32, objectType PackFileObjectType, err error) {
	// read the first byte
	firstByte, err := packFileBuffer.ReadByte()
	if err != nil {
		return 0, 0, err
	}

	// Packfile object header format:
	// ┌─────────┬──────────┬──────────┐
	// │MSB 1 bit│Type 3 bit│Size 4 bit│
	// └─────────┴──────────┴──────────┘
	// ┌─────────┬──────────┐
	// │MSB 1 bit│Size 7 bit│
	// └─────────┴──────────┘
	// remaining bytes for size to read...
	//
	// get only the first 4 bits for MSB and type
	firstFourBytes := firstByte >> 4
	objectType = PackFileObjectType(firstFourBytes & 0x7)
	MSB := firstByte & 0x80 >> 7
	size = uint32(firstByte & 0x0f)

	shift := uint(4)

	for MSB != 0 {
		// read the next byte
		b, err := packFileBuffer.ReadByte()
		if err != nil {
			return 0, 0, err
		}

		// update MSB
		MSB = b & 0x80 >> 7
		size += uint32(b&0x7f) << shift
		shift += 7
	}
	return
}

func parseObject(packFileBuffer *bytes.Reader) ([]byte, error) {
	zlibReader, err := zlib.NewReader(packFileBuffer)
	if err != nil {
		return nil, err
	}
	defer zlibReader.Close()

	object, err := io.ReadAll(zlibReader)
	if err != nil {
		return nil, err
	}

	return object, nil
}

// reads a variable length integer from the packfile buffer
// based on the MSB / SIZE format
//
// [MSB 1 bit][SIZE 7 bit]
// [MSB 1 bit][SIZE 7 bit]
// ...
func parseSize(data *bytes.Reader) (uint32, error) {
	b, err := data.ReadByte()
	if err != nil {
		return 0, err
	}

	size := uint32(b & 0x7f)
	msb := b & 0x80 >> 7
	shift := 7

	for msb != 0 {
		b, err = data.ReadByte()
		if err != nil {
			return 0, err
		}
		size += uint32(b&0x7f) << shift
		msb = b & 0x80 >> 7
		shift += 7
	}

	return size, nil
}

// ======================== Delta object ========================

type deltaObject struct {
	baseObject string
	data       []byte
}

func applyDeltas(deltaObjects []deltaObject) error {
	// We would need to check if deltaObjects can be resolved
	// that is: if the root base object is already written to disk

	fmt.Printf("remote: Resolving deltas: %d\n", len(deltaObjects))
	for len(deltaObjects) > 0 {
		noBaseDeltaObjects := []deltaObject{}
		atLeastOneBaseObject := false

		for _, deltaObject := range deltaObjects {
			if objectExists(deltaObject.baseObject) {
				atLeastOneBaseObject = true
				err := applyDelta(deltaObject)
				if err != nil {
					return err
				}
			} else {
				noBaseDeltaObjects = append(noBaseDeltaObjects, deltaObject)
			}
		}

		if !atLeastOneBaseObject {
			return fmt.Errorf("no base object found for delta objects, cannot resolve")
		}

		deltaObjects = noBaseDeltaObjects
	}

	return nil
}

func applyDelta(deltaObject deltaObject) error {
	baseObject, err := NewObject(deltaObject.baseObject)
	if err != nil {
		return err
	}

	deltaData := bytes.NewReader(deltaObject.data)
	baseSize, err := parseSize(deltaData) // source buffer size
	if err != nil {
		return err
	}

	// check if the base object size matches the size in the delta object
	if baseSize != uint32(len(baseObject.Content)) {
		return fmt.Errorf("base object size mismatch")
	}

	expectedSize, err := parseSize(deltaData) // target buffer size
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(nil)

	for deltaData.Len() > 0 {
		opCode, err := deltaData.ReadByte()
		if err != nil {
			return err
		}

		// check MSB
		// if MSB is set => copy instruction
		// else => insert instruction

		// Check gitformat-pack.txt
		// https://github.com/git/git/blob/795ea8776befc95ea2becd8020c7a284677b4161/Documentation/gitformat-pack.txt

		if opCode&0x80 != 0 { // copy instruction
			//  +----------+---------+---------+---------+---------+-------+-------+-------+
			//  | 1xxxxxxx | offset1 | offset2 | offset3 | offset4 | size1 | size2 | size3 |
			//  +----------+---------+---------+---------+---------+-------+-------+-------+
			arg := uint64(0)
			for bit := 0; bit < 7; bit++ {
				if opCode&(1<<bit) != 0 {
					nextByte, err := deltaData.ReadByte()
					if err != nil {
						return err
					}
					arg |= uint64(nextByte) << (bit * 8)
				}
			}
			offset := arg & 0xffffffff     // 4 bytes
			size := (arg >> 32) & 0xffffff // skip the 4 bytes and get 3 last bytes
			if size == 0 {
				size = 0x10000
			}
			buffer.Write(baseObject.Content[offset : offset+size])

		} else { // insert instruction
			size := uint32(opCode & 0x7f)
			data := make([]byte, size)
			_, err := deltaData.Read(data)
			if err != nil {
				return err
			}
			buffer.Write(data)
		}
	}

	if buffer.Len() != int(expectedSize) {
		return fmt.Errorf("delta object size mismatch")
	}

	// write object to disk
	err = writePackFileObject(string(baseObject.Type), buffer.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func objectExists(sha string) bool {
	objectPath := getObjectPath(sha)
	if _, err := os.Stat(objectPath); err != nil || errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

// ======================== Helpers ========================

func toPktLine(value string) string {
	return fmt.Sprintf("%04x%s", len(value)+4, value)
}

func writePackFileObject(objectType string, object []byte) error {
	header := fmt.Sprintf("%s %d\x00", objectType, len(object))
	hash := sha1.New()
	if _, err := hash.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := hash.Write(object); err != nil {
		return err
	}

	sha := fmt.Sprintf("%x", hash.Sum(nil))

	err := writeObject(sha, []byte(header), bytes.NewReader(object))
	if err != nil {
		return err
	}

	return nil
}
