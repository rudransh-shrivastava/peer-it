package protocol

import (
	"bytes"
	"testing"
)

func TestCodecPingPong(t *testing.T) {
	codec := NewCodec()
	var buf bytes.Buffer

	// Test Ping
	if err := codec.Encode(&buf, &Ping{}); err != nil {
		t.Fatalf("Encode Ping failed: %v", err)
	}

	decoded, err := codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode Ping failed: %v", err)
	}

	if _, ok := decoded.(*Ping); !ok {
		t.Errorf("Expected *Ping, got %T", decoded)
	}

	// Test Pong
	buf.Reset()
	if err := codec.Encode(&buf, &Pong{}); err != nil {
		t.Fatalf("Encode Pong failed: %v", err)
	}

	decoded, err = codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode Pong failed: %v", err)
	}

	if _, ok := decoded.(*Pong); !ok {
		t.Errorf("Expected *Pong, got %T", decoded)
	}
}

func TestCodecFileListReqRes(t *testing.T) {
	codec := NewCodec()
	var buf bytes.Buffer

	// Test FileListReq
	if err := codec.Encode(&buf, &FileListReq{}); err != nil {
		t.Fatalf("Encode FileListReq failed: %v", err)
	}

	decoded, err := codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode FileListReq failed: %v", err)
	}

	if _, ok := decoded.(*FileListReq); !ok {
		t.Errorf("Expected *FileListReq, got %T", decoded)
	}

	// Test FileListRes with multiple files
	buf.Reset()
	res := &FileListRes{
		Files: []FileEntry{
			{Hash: testHash("file1"), Size: 1024, Name: "document.pdf"},
			{Hash: testHash("file2"), Size: 2048, Name: "image.png"},
		},
	}

	if err := codec.Encode(&buf, res); err != nil {
		t.Fatalf("Encode FileListRes failed: %v", err)
	}

	decoded, err = codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode FileListRes failed: %v", err)
	}

	decodedRes, ok := decoded.(*FileListRes)
	if !ok {
		t.Fatalf("Expected *FileListRes, got %T", decoded)
	}

	if len(decodedRes.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(decodedRes.Files))
	}

	if decodedRes.Files[0].Name != "document.pdf" {
		t.Errorf("Expected 'document.pdf', got '%s'", decodedRes.Files[0].Name)
	}

	if decodedRes.Files[1].Size != 2048 {
		t.Errorf("Expected size 2048, got %d", decodedRes.Files[1].Size)
	}
}

func TestCodecFileMetaReqRes(t *testing.T) {
	codec := NewCodec()
	var buf bytes.Buffer

	hash := testHash("testfile")

	// Test FileMetaReq
	req := &FileMetaReq{Hash: hash}
	if err := codec.Encode(&buf, req); err != nil {
		t.Fatalf("Encode FileMetaReq failed: %v", err)
	}

	decoded, err := codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode FileMetaReq failed: %v", err)
	}

	decodedReq, ok := decoded.(*FileMetaReq)
	if !ok {
		t.Fatalf("Expected *FileMetaReq, got %T", decoded)
	}

	if decodedReq.Hash != hash {
		t.Errorf("Hash mismatch")
	}

	// Test FileMetaRes
	buf.Reset()
	res := &FileMetaRes{
		Hash:         hash,
		Size:         1024 * 1024 * 10,
		Name:         "largefile.zip",
		MaxChunkSize: MaxChunkSize,
		Chunks: []ChunkMeta{
			{Index: 0, Size: MaxChunkSize, Hash: testHash("chunk0")},
			{Index: 1, Size: MaxChunkSize, Hash: testHash("chunk1")},
			{Index: 2, Size: 1024, Hash: testHash("chunk2")},
		},
	}

	if err := codec.Encode(&buf, res); err != nil {
		t.Fatalf("Encode FileMetaRes failed: %v", err)
	}

	decoded, err = codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode FileMetaRes failed: %v", err)
	}

	decodedRes, ok := decoded.(*FileMetaRes)
	if !ok {
		t.Fatalf("Expected *FileMetaRes, got %T", decoded)
	}

	if decodedRes.Name != "largefile.zip" {
		t.Errorf("Expected 'largefile.zip', got '%s'", decodedRes.Name)
	}

	if len(decodedRes.Chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(decodedRes.Chunks))
	}
}

func TestCodecChunkReqRes(t *testing.T) {
	codec := NewCodec()
	var buf bytes.Buffer

	fileHash := testHash("myfile")

	// Test ChunkReq
	req := &ChunkReq{FileHash: fileHash, ChunkIndex: 42}
	if err := codec.Encode(&buf, req); err != nil {
		t.Fatalf("Encode ChunkReq failed: %v", err)
	}

	decoded, err := codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode ChunkReq failed: %v", err)
	}

	decodedReq, ok := decoded.(*ChunkReq)
	if !ok {
		t.Fatalf("Expected *ChunkReq, got %T", decoded)
	}

	if decodedReq.ChunkIndex != 42 {
		t.Errorf("Expected chunk index 42, got %d", decodedReq.ChunkIndex)
	}

	// Test ChunkRes
	buf.Reset()
	chunkData := []byte("This is some chunk data for testing purposes.")
	res := &ChunkRes{FileHash: fileHash, ChunkIndex: 42, Data: chunkData}

	if err := codec.Encode(&buf, res); err != nil {
		t.Fatalf("Encode ChunkRes failed: %v", err)
	}

	decoded, err = codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode ChunkRes failed: %v", err)
	}

	decodedRes, ok := decoded.(*ChunkRes)
	if !ok {
		t.Fatalf("Expected *ChunkRes, got %T", decoded)
	}

	if !bytes.Equal(decodedRes.Data, chunkData) {
		t.Errorf("Chunk data mismatch")
	}
}

func TestCodecPeerAnnounce(t *testing.T) {
	codec := NewCodec()
	var buf bytes.Buffer

	msg := &PeerAnnounce{
		NodeID:    testNodeID("test-node-1"),
		Port:      59000,
		FileCount: 10,
	}

	if err := codec.Encode(&buf, msg); err != nil {
		t.Fatalf("Encode PeerAnnounce failed: %v", err)
	}

	decoded, err := codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode PeerAnnounce failed: %v", err)
	}

	decodedMsg, ok := decoded.(*PeerAnnounce)
	if !ok {
		t.Fatalf("Expected *PeerAnnounce, got %T", decoded)
	}

	if decodedMsg.Port != 59000 {
		t.Errorf("Expected port 59000, got %d", decodedMsg.Port)
	}

	if decodedMsg.FileCount != 10 {
		t.Errorf("Expected file count 10, got %d", decodedMsg.FileCount)
	}
}

func TestCodecPeerListReqRes(t *testing.T) {
	codec := NewCodec()
	var buf bytes.Buffer

	fileHash := testHash("shared-file")

	// Test PeerListReq
	if err := codec.Encode(&buf, &PeerListReq{FileHash: fileHash}); err != nil {
		t.Fatalf("Encode PeerListReq failed: %v", err)
	}

	decoded, err := codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode PeerListReq failed: %v", err)
	}

	if _, ok := decoded.(*PeerListReq); !ok {
		t.Fatalf("Expected *PeerListReq, got %T", decoded)
	}

	// Test PeerListRes
	buf.Reset()
	res := &PeerListRes{
		FileHash: fileHash,
		Peers: []PeerInfo{
			{NodeID: testNodeID("peer1"), IP: testIPv4(192, 168, 1, 100), Port: 59001},
			{NodeID: testNodeID("peer2"), IP: testIPv4(192, 168, 1, 101), Port: 59002},
		},
	}

	if err := codec.Encode(&buf, res); err != nil {
		t.Fatalf("Encode PeerListRes failed: %v", err)
	}

	decoded, err = codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode PeerListRes failed: %v", err)
	}

	decodedRes, ok := decoded.(*PeerListRes)
	if !ok {
		t.Fatalf("Expected *PeerListRes, got %T", decoded)
	}

	if len(decodedRes.Peers) != 2 {
		t.Errorf("Expected 2 peers, got %d", len(decodedRes.Peers))
	}
}

func TestCodecDiscovery(t *testing.T) {
	codec := NewCodec()
	var buf bytes.Buffer

	msg := &Discovery{
		NodeID:    testNodeID("discoverable-node"),
		Port:      59000,
		FileCount: 5,
	}

	if err := codec.Encode(&buf, msg); err != nil {
		t.Fatalf("Encode Discovery failed: %v", err)
	}

	decoded, err := codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode Discovery failed: %v", err)
	}

	decodedMsg, ok := decoded.(*Discovery)
	if !ok {
		t.Fatalf("Expected *Discovery, got %T", decoded)
	}

	if decodedMsg.FileCount != 5 {
		t.Errorf("Expected file count 5, got %d", decodedMsg.FileCount)
	}
}

func TestCodecError(t *testing.T) {
	codec := NewCodec()
	var buf bytes.Buffer

	msg := &Error{
		Code:    ErrFileNotFound,
		Message: "The requested file does not exist",
	}

	if err := codec.Encode(&buf, msg); err != nil {
		t.Fatalf("Encode Error failed: %v", err)
	}

	decoded, err := codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode Error failed: %v", err)
	}

	decodedMsg, ok := decoded.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", decoded)
	}

	if decodedMsg.Code != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", decodedMsg.Code)
	}

	if decodedMsg.Message != "The requested file does not exist" {
		t.Errorf("Message mismatch: %s", decodedMsg.Message)
	}
}

func TestCodecEmptyFileList(t *testing.T) {
	codec := NewCodec()
	var buf bytes.Buffer

	res := &FileListRes{Files: []FileEntry{}}

	if err := codec.Encode(&buf, res); err != nil {
		t.Fatalf("Encode empty FileListRes failed: %v", err)
	}

	decoded, err := codec.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode empty FileListRes failed: %v", err)
	}

	decodedRes, ok := decoded.(*FileListRes)
	if !ok {
		t.Fatalf("Expected *FileListRes, got %T", decoded)
	}

	if len(decodedRes.Files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(decodedRes.Files))
	}
}

func TestCodecEncodeToBytes(t *testing.T) {
	codec := NewCodec()

	data, err := codec.EncodeToBytes(&Ping{})
	if err != nil {
		t.Fatalf("EncodeToBytes failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty data")
	}
}

func TestCodecDecodeFromBytes(t *testing.T) {
	codec := NewCodec()

	data, err := codec.EncodeToBytes(&Pong{})
	if err != nil {
		t.Fatalf("EncodeToBytes failed: %v", err)
	}

	decoded, err := codec.DecodeFromBytes(data)
	if err != nil {
		t.Fatalf("DecodeFromBytes failed: %v", err)
	}

	if _, ok := decoded.(*Pong); !ok {
		t.Errorf("Expected *Pong, got %T", decoded)
	}
}

func TestMessageTypeString(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		expected string
	}{
		{MsgPing, "PING"},
		{MsgPong, "PONG"},
		{MsgFileListReq, "FILE_LIST_REQ"},
		{MsgError, "ERROR"},
		{MessageType(0xFFFF), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.msgType.String(); got != tt.expected {
			t.Errorf("%v.String() = %s, want %s", tt.msgType, got, tt.expected)
		}
	}
}

func TestErrorCodeString(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrUnknown, "UNKNOWN"},
		{ErrInvalidMsg, "INVALID_MESSAGE"},
		{ErrFileNotFound, "FILE_NOT_FOUND"},
		{ErrorCode(0xFFFE), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.code.String(); got != tt.expected {
			t.Errorf("%v.String() = %s, want %s", tt.code, got, tt.expected)
		}
	}
}

// --- Test helpers ---

func testHash(s string) [HashSize]byte {
	var h [HashSize]byte
	copy(h[:], []byte(s))
	return h
}

func testNodeID(s string) [NodeIDSize]byte {
	var id [NodeIDSize]byte
	copy(id[:], []byte(s))
	return id
}

func testIPv4(a, b, c, d byte) [16]byte {
	var ip [16]byte
	ip[10] = 0xff
	ip[11] = 0xff
	ip[12] = a
	ip[13] = b
	ip[14] = c
	ip[15] = d
	return ip
}
