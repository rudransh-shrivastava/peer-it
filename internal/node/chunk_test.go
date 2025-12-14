package node

import (
	"bytes"
	"testing"

	internaldb "github.com/rudransh-shrivastava/peer-it/internal/db"
)

func TestBuildChunkMap_Empty(t *testing.T) {
	chunks := []internaldb.Chunk{}
	result := BuildChunkMap(4, chunks)

	if len(result) != 4 {
		t.Errorf("expected length 4, got %d", len(result))
	}
	for i, v := range result {
		if v != 0 {
			t.Errorf("expected chunk %d to be 0, got %d", i, v)
		}
	}
}

func TestBuildChunkMap_AllAvailable(t *testing.T) {
	chunks := []internaldb.Chunk{
		{ChunkIndex: 0, IsAvailable: 1},
		{ChunkIndex: 1, IsAvailable: 1},
		{ChunkIndex: 2, IsAvailable: 1},
		{ChunkIndex: 3, IsAvailable: 1},
	}
	result := BuildChunkMap(4, chunks)

	for i, v := range result {
		if v != 1 {
			t.Errorf("expected chunk %d to be 1, got %d", i, v)
		}
	}
}

func TestBuildChunkMap_Partial(t *testing.T) {
	chunks := []internaldb.Chunk{
		{ChunkIndex: 0, IsAvailable: 1},
		{ChunkIndex: 1, IsAvailable: 0},
		{ChunkIndex: 2, IsAvailable: 1},
		{ChunkIndex: 3, IsAvailable: 0},
	}
	result := BuildChunkMap(4, chunks)

	expected := []int32{1, 0, 1, 0}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("chunk %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

func TestFindMissingChunkIndex_AllAvailable(t *testing.T) {
	chunks := []internaldb.Chunk{
		{IsAvailable: 1},
		{IsAvailable: 1},
		{IsAvailable: 1},
	}
	result := FindMissingChunkIndex(chunks)
	if result != -1 {
		t.Errorf("expected -1, got %d", result)
	}
}

func TestFindMissingChunkIndex_FirstMissing(t *testing.T) {
	chunks := []internaldb.Chunk{
		{IsAvailable: 0},
		{IsAvailable: 1},
		{IsAvailable: 1},
	}
	result := FindMissingChunkIndex(chunks)
	if result != 0 {
		t.Errorf("expected 0, got %d", result)
	}
}

func TestFindMissingChunkIndex_MiddleMissing(t *testing.T) {
	chunks := []internaldb.Chunk{
		{IsAvailable: 1},
		{IsAvailable: 0},
		{IsAvailable: 1},
	}
	result := FindMissingChunkIndex(chunks)
	if result != 1 {
		t.Errorf("expected 1, got %d", result)
	}
}

func TestIsDownloadComplete_True(t *testing.T) {
	chunks := []internaldb.Chunk{
		{IsAvailable: 1},
		{IsAvailable: 1},
		{IsAvailable: 1},
	}
	if !IsDownloadComplete(chunks) {
		t.Error("expected complete")
	}
}

func TestIsDownloadComplete_False(t *testing.T) {
	chunks := []internaldb.Chunk{
		{IsAvailable: 1},
		{IsAvailable: 0},
		{IsAvailable: 1},
	}
	if IsDownloadComplete(chunks) {
		t.Error("expected incomplete")
	}
}

func TestIsDownloadComplete_Empty(t *testing.T) {
	chunks := []internaldb.Chunk{}
	if !IsDownloadComplete(chunks) {
		t.Error("expected empty to be complete")
	}
}

func TestReadChunkData(t *testing.T) {
	data := []byte("0123456789ABCDEFGHIJ")
	reader := bytes.NewReader(data)

	chunk, err := ReadChunkData(reader, 1, 5, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "56789"
	if string(chunk) != expected {
		t.Errorf("expected %q, got %q", expected, string(chunk))
	}
}

func TestReadChunkData_FirstChunk(t *testing.T) {
	data := []byte("HelloWorld")
	reader := bytes.NewReader(data)

	chunk, err := ReadChunkData(reader, 0, 5, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(chunk) != "Hello" {
		t.Errorf("expected 'Hello', got %q", string(chunk))
	}
}

func TestWriteChunkData(t *testing.T) {
	buf := make([]byte, 20)
	writer := &bytesWriterAt{buf: buf}

	err := WriteChunkData(writer, 1, 5, []byte("XXXXX"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "\x00\x00\x00\x00\x00XXXXX\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
	if string(buf) != expected {
		t.Errorf("expected %q, got %q", expected, string(buf))
	}
}

type bytesWriterAt struct {
	buf []byte
}

func (b *bytesWriterAt) WriteAt(p []byte, off int64) (n int, err error) {
	copy(b.buf[off:], p)
	return len(p), nil
}
