package protocol

import (
	"bytes"
	"encoding/gob"
	"io"
	"sync"
)

func init() {
	gob.Register(&CallReq{})
	gob.Register(&ChunkReq{})
	gob.Register(&ChunkRes{})
	gob.Register(&Discovery{})
	gob.Register(&Error{})
	gob.Register(&FileListReq{})
	gob.Register(&FileListRes{})
	gob.Register(&FileMetaReq{})
	gob.Register(&FileMetaRes{})
	gob.Register(&HolePunchProbe{})
	gob.Register(&PeerAnnounce{})
	gob.Register(&PeerListReq{})
	gob.Register(&PeerListRes{})
	gob.Register(&Ping{})
	gob.Register(&Pong{})
}

type Codec struct {
	dec *gob.Decoder
	enc *gob.Encoder
	mu  sync.Mutex
	r   io.Reader
	w   io.Writer
}

func NewCodec() *Codec {
	return &Codec{}
}

func (c *Codec) Decode(r io.Reader) (Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.dec == nil || c.r != r {
		c.dec = gob.NewDecoder(r)
		c.r = r
	}

	var msg Message
	if err := c.dec.Decode(&msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (c *Codec) DecodeFromBytes(data []byte) (Message, error) {
	var msg Message
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (c *Codec) Encode(w io.Writer, msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.enc == nil || c.w != w {
		c.enc = gob.NewEncoder(w)
		c.w = w
	}

	return c.enc.Encode(&msg)
}

func (c *Codec) EncodeToBytes(msg Message) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&msg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
