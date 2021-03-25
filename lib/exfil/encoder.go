package exfil

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
)

// Encoding configures the base64 library
var Encoding = base64.StdEncoding.WithPadding(base64.NoPadding)

// Encode binary data to subdomain(s)
func Encode(data []byte) string {
	buf := bytes.NewBuffer(nil)
	b64w := base64.NewEncoder(Encoding, buf)
	b64w.Write(data)
	b64w.Close()

	s := buf.String()
	parts := make([]string, (len(s)/partLength)+1)
	for i := range parts {
		start := i * partLength
		end := start + partLength
		if end > len(s) {
			end = len(s)
		}
		part := s[start:end]
		part = strings.Map(func(r rune) rune {
			if rr, ok := Mapping[r]; ok {
				return rr
			}
			return r
		}, part)
		parts[i], _ = idna.ToASCII(part)
	}

	encoded := strings.TrimRight(strings.Join(parts, "."), ".")
	return encoded + ".l" + strconv.Itoa(len(encoded))
}

// GenHeaderMsg generates a header message, announcing the file to the server
func GenHeaderMsg(id uint32, name string, size uint32) (string, error) {
	id = id | 0x80000000
	buf := bytes.NewBuffer(nil)
	err := binary.Write(buf, binary.BigEndian, id)
	if err != nil {
		return "", err
	}
	err = binary.Write(buf, binary.BigEndian, size)
	if err != nil {
		return "", err
	}
	_, err = buf.WriteString(name)
	if err != nil {
		return "", err
	}
	return Encode(buf.Bytes()), nil
}

// GenContentMsg generates a content message, transporting a chunk of the file
func GenContentMsg(id uint32, offset uint32, data []byte) (string, error) {
	id = id & 0x7fffffff
	buf := bytes.NewBuffer(nil)
	err := binary.Write(buf, binary.BigEndian, id)
	if err != nil {
		return "", err
	}
	err = binary.Write(buf, binary.BigEndian, offset)
	if err != nil {
		return "", err
	}
	_, err = buf.Write(data)
	if err != nil {
		return "", err
	}
	return Encode(buf.Bytes()), nil
}

// EncodeFile transforms a file into a series of messages
func EncodeFile(filePath string) (chan string, error) {
	s, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	bn := path.Base(filePath)
	sz := s.Size()
	if sz > 0xffffffff {
		return nil, errors.New("File to large")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	crc := crc32.NewIEEE()
	io.Copy(crc, f)
	f.Seek(0, os.SEEK_SET)

	return encodeFileContent(crc.Sum32(), bn, uint32(sz), f), nil
}

// encodeFileContent implements the packetization of the file
func encodeFileContent(id uint32, name string, size uint32, data io.ReadCloser) chan string {
	c := make(chan string)
	go func(c chan<- string, id, size uint32, data io.ReadCloser) {
		defer close(c)
		defer data.Close()

		msg, err := GenHeaderMsg(id, name, size)
		if err != nil {
			return
		}

		c <- msg

		var i uint32
		var block = make([]byte, blockSize)
		for i = 0; i < size; i += blockSize {
			n, err := data.Read(block)
			if err != nil {
				return
			}

			msg, err := GenContentMsg(id, i, block[0:n])
			if err != nil {
				return
			}

			c <- msg
		}
	}(c, id, size, data)
	return c
}
