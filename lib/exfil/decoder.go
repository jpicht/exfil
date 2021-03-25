package exfil

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
)

// Decode takes a partial domain and decodes it to a packat
func Decode(payload string) (*packet, error) {
	parts := strings.Split(payload, ".")
	lenMarker := parts[len(parts)-1]
	parts = parts[0 : len(parts)-1]

	expected, err := strconv.Atoi(lenMarker[1:])

	if err != nil || expected != (len(payload)-len(lenMarker)-1) {
		return nil, ERR_PAYLOAD_INCOMPLETE
	}

	for i, p := range parts {
		parts[i], err = idna.ToUnicode(p)

		if err != nil {
			return nil, err
		}

		parts[i] = strings.Map(
			func(r rune) rune {
				if rr, ok := ReverseMapping[r]; ok {
					return rr
				}
				return r
			},
			parts[i],
		)
	}

	data, err := ioutil.ReadAll(
		base64.NewDecoder(
			Encoding,
			bytes.NewBufferString(strings.Join(parts, "")),
		),
	)

	if err != nil {
		return nil, ERR_PAYLOAD_INCOMPLETE
	}

	return packetFromBytes(data)
}
