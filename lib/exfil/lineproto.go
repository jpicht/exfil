package exfil

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type (
	packet struct {
		t  packetType
		id uint32
		h  *header
		c  *content
	}
	header struct {
		size uint32
		name string
	}
	content struct {
		offset uint32
		data   []byte
	}
	packetType uint32
)

var (
	ERR_PAYLOAD_INCOMPLETE = errors.New("Payload incomplete")

	TYPE_CONTENT packetType = 0
	TYPE_HEADER  packetType = 0x80000000
)

func newPacket(id, mixed uint32, data []byte) *packet {
	p := &packet{
		t:  packetType(id) & TYPE_HEADER,
		id: id & 0x7fffffff,
	}
	if p.IsHeader() {
		p.h = &header{
			size: mixed,
			name: string(data),
		}
	} else {
		p.c = &content{
			offset: mixed,
			data:   data,
		}
	}
	return p
}

func packetFromBytes(data []byte) (*packet, error) {
	if len(data) < 8 {
		return nil, ERR_PAYLOAD_INCOMPLETE
	}

	reader := bytes.NewBuffer(data)

	var decoded [2]uint32
	err := binary.Read(reader, binary.BigEndian, &decoded)
	if err != nil {
		return nil, err
	}

	return newPacket(decoded[0], decoded[1], data[8:]), nil
}

func (p *packet) Content() *content {
	return p.c
}

func (p *packet) Header() *header {
	return p.h
}

func (p *packet) Id() uint32 {
	return p.id
}

func (p *packet) IsHeader() bool {
	return p.t == TYPE_HEADER
}

func (h *header) Name() string {
	return h.name
}

func (h *header) Size() uint32 {
	return h.size
}

func (c *content) Data() []byte {
	return c.data
}

func (c *content) Offset() uint32 {
	return c.offset
}
