// Provides utility functions.

package zoossh

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type QueueUnit struct {
	Blurb string
	Err   error
}

type Delimiter struct {
	Pattern string
	Offset  uint
	Skip    uint
}

type Annotation struct {
	Type  string
	Major string
	Minor string
}

func (a *Annotation) String() string {

	return fmt.Sprintf("@type %s %s.%s", a.Type, a.Major, a.Minor)
}

// Equals checks whether the two given annotations have the same content.
func (a *Annotation) Equals(b *Annotation) bool {

	return (*a).Type == (*b).Type && (*a).Major == (*b).Major && (*a).Minor == (*b).Minor
}

// Decodes the given Base64-encoded string and returns the resulting string.
// If there are errors during decoding, an error string is returned.
func Base64ToString(encoded string) (string, error) {

	// dir-spec.txt says that Base64 padding is removed so we have to account
	// for that here.
	if rem := len(encoded) % 4; rem != 0 {
		encoded += strings.Repeat("=", 4-rem)
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(decoded), nil
}

// GetAnnotation obtains and returns the given file's annotation.  If anything
// fails in the process, an error string is returned.
func GetAnnotation(fileName string) (*Annotation, error) {

	fd, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	// Fetch the file's first line which should be the annotation.

	scanner := bufio.NewScanner(fd)
	scanner.Scan()
	annotationText := scanner.Text()

	annotation := new(Annotation)

	// We expect "@type TYPE VERSION".
	words := strings.Split(annotationText, " ")
	if len(words) != 3 {
		return nil, fmt.Errorf("Could not parse file annotation for \"%s\".", fileName)
	}
	annotation.Type = words[1]

	// We expect "MAJOR.MINOR".
	version := strings.Split(words[2], ".")
	if len(version) != 2 {
		return nil, fmt.Errorf("Could not parse file annotation for \"%s\".", fileName)
	}
	annotation.Major = version[0]
	annotation.Minor = version[1]

	return annotation, nil
}

// Checks the type annotation in the given file.  The Annotation struct
// determines what we want to see in the file.  If we don't see the expected
// annotation, an error string is returned.
func CheckAnnotation(fd *os.File, expected map[Annotation]bool) error {

	// The annotation is placed in the first line of the file.  See the
	// following URL for details:
	// <https://collector.torproject.org/formats.html>
	scanner := bufio.NewScanner(fd)
	scanner.Scan()
	annotation := scanner.Text()

	invalidFormat := fmt.Errorf("Unexpected file annotation: %s", annotation)

	// We expect "@type TYPE VERSION".
	words := strings.Split(annotation, " ")
	if len(words) != 3 {
		return invalidFormat
	}

	// We expect "MAJOR.MINOR".
	version := strings.Split(words[2], ".")
	if len(version) != 2 {
		return invalidFormat
	}
	observed := Annotation{words[1], version[0], version[1]}

	for annotation, _ := range expected {
		// We support the observed annotation.
		if annotation.Equals(&observed) {
			return nil
		}
	}

	return invalidFormat
}

// Dissects the given file into string chunks as specified by the given
// delimiter.  The resulting string chunks are then written to the given queue
// where the receiving end parses them.
func DissectFile(fd *os.File, delim Delimiter, queue chan QueueUnit) {

	defer close(queue)

	blurb, err := ioutil.ReadAll(fd)
	if err != nil {
		queue <- QueueUnit{"", err}
	}

	rawContent := string(blurb)

	for {
		// Jump to the end of the next string blurb.
		position := strings.Index(rawContent, delim.Pattern)
		if position == -1 {
			break
		}
		position += int(delim.Offset)

		if delim.Skip > 0 {
			delim.Skip -= 1
		} else {
			queue <- QueueUnit{rawContent[:position], nil}
		}

		// Point to the beginning of the next string blurb.
		rawContent = rawContent[position:]
	}
}

// Convert the given port string to an unsigned 16-bit integer.  If the
// conversion fails or the number cannot be represented in 16 bits, 0 is
// returned.
func StringToPort(portStr string) uint16 {

	portNum, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return uint16(0)
	}

	return uint16(portNum)
}
