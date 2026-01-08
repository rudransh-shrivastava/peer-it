package protocol

import (
	"bytes"
	"encoding/gob"
	"io"
)

func init() {
	gob.Register(&ChunkReq{})
	gob.Register(&ChunkRes{})
	gob.Register(&Discovery{})
	gob.Register(&Error{})
	gob.Register(&FileListReq{})
	gob.Register(&FileListRes{})
	gob.Register(&FileMetaReq{})
	gob.Register(&FileMetaRes{})
	gob.Register(&HolePunchProbe{})
	gob.Register(&HolePunchReq{})
	gob.Register(&PeerAnnounce{})
	gob.Register(&PeerListReq{})
	gob.Register(&PeerListRes{})
	gob.Register(&Ping{})
	gob.Register(&Pong{})
}

type Codec struct{}

func NewCodec() *Codec {
	return &Codec{}
}

func (c *Codec) Decode(r io.Reader) (Message, error) {
	var msg Message
	if err := gob.NewDecoder(r).Decode(&msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (c *Codec) DecodeFromBytes(data []byte) (Message, error) {
	return c.Decode(bytes.NewReader(data))
}

func (c *Codec) Encode(w io.Writer, msg Message) error {
	return gob.NewEncoder(w).Encode(&msg)
}

func (c *Codec) EncodeToBytes(msg Message) ([]byte, error) {
	var buf bytes.Buffer
	if err := c.Encode(&buf, msg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
