package steam

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var unityVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+[abfp][0-9]+(?:-[A-Za-z][A-Za-z0-9]*)?$`)

// ExtractUnityVersionFromDir reads the engine version recorded by Unity in the
// Windows player's serialized globalgamemanagers file.
func ExtractUnityVersionFromDir(dir string) (string, error) {
	file, err := os.Open(filepath.Join(dir, "VRChat_Data", "globalgamemanagers"))
	if err != nil {
		return "", err
	}
	defer file.Close()
	return readUnityVersion(file)
}

func readUnityVersion(file io.Reader) (string, error) {
	reader := bufio.NewReaderSize(file, 128)
	header, err := reader.Peek(20)
	if err != nil {
		return "", fmt.Errorf("read Unity serialized file header: %w", err)
	}
	format := binary.BigEndian.Uint32(header[8:12])
	var headerSize int
	switch {
	case format >= 9 && format <= 21:
		headerSize = 20
	case format == 22:
		headerSize = 48 // The large-file header adds 28 bytes.
	default:
		return "", fmt.Errorf("unsupported Unity serialized file format %d", format)
	}
	if _, err := reader.Discard(headerSize); err != nil {
		return "", fmt.Errorf("read Unity serialized file header: %w", err)
	}
	version, err := reader.ReadSlice(0)
	if err != nil {
		return "", fmt.Errorf("read Unity engine version: %w", err)
	}
	value := string(version[:len(version)-1])
	if !unityVersionPattern.MatchString(value) {
		return "", fmt.Errorf("invalid Unity engine version")
	}
	return value, nil
}
