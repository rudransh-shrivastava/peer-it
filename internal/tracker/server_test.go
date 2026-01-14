package tracker

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/rudransh-shrivastava/peer-it/internal/protocol"
)

func TestNewServer(t *testing.T) {
	listener := newMockListener()
	srv, err := NewServer(Config{
		Listener: listener,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer func() { _ = srv.Shutdown() }()

	if srv.Addr() == "" {
		t.Error("Expected non-empty address")
	}
}

func TestServerAddr(t *testing.T) {
	listener := newMockListener()
	srv, err := NewServer(Config{
		Listener: listener,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer func() { _ = srv.Shutdown() }()

	addr := srv.Addr()
	if addr != "mock://test:0" {
		t.Errorf("Expected mock://test:0, got %s", addr)
	}
}

func TestServerHandlePing(t *testing.T) {
	srv, conn := setupMockServerConn(t)
	defer func() { _ = srv.Shutdown() }()

	srv.handleMessage(t.Context(), conn, &protocol.Ping{})

	if len(conn.sentMsgs) != 1 {
		t.Fatalf("Expected 1 sent message, got %d", len(conn.sentMsgs))
	}

	if _, ok := conn.sentMsgs[0].(*protocol.Pong); !ok {
		t.Errorf("Expected Pong, got %T", conn.sentMsgs[0])
	}
}

func TestServerHandlePeerAnnounce(t *testing.T) {
	srv, conn := setupMockServerConn(t)
	defer func() { _ = srv.Shutdown() }()

	fileName := "file1.txt"
	fileSize := uint64(1024)
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s%d", fileName, fileSize)))

	announce := &protocol.PeerAnnounce{
		FileCount: 1,
		Files: []protocol.FileEntry{
			{Hash: hash, Name: fileName, Size: fileSize},
		},
	}

	srv.handleMessage(t.Context(), conn, announce)

	files := srv.store.ListFiles()
	if len(files) != 1 {
		t.Errorf("Expected 1 file in store, got %d", len(files))
	}
}

func TestGenerateHash(t *testing.T) {
	file := &protocol.FileEntry{Name: "test.txt", Size: 1024}
	hash := generateHash(file)

	expected := sha256.Sum256([]byte("test.txt1024"))
	if hash != expected {
		t.Errorf("Hash mismatch")
	}
}

func TestGenerateHashDifferentInputs(t *testing.T) {
	file1 := &protocol.FileEntry{Name: "file1.txt", Size: 1024}
	file2 := &protocol.FileEntry{Name: "file2.txt", Size: 1024}

	hash1 := generateHash(file1)
	hash2 := generateHash(file2)

	if hash1 == hash2 {
		t.Error("Different files should have different hashes")
	}
}

func TestServerHandlePeerAnnounceMismatchedFileCount(t *testing.T) {
	srv, conn := setupMockServerConn(t)
	defer func() { _ = srv.Shutdown() }()

	announce := &protocol.PeerAnnounce{
		FileCount: 2,
		Files: []protocol.FileEntry{
			{Hash: protocol.FileHash{}, Name: "file1.txt", Size: 1024},
		},
	}

	srv.handleMessage(t.Context(), conn, announce)

	files := srv.store.ListFiles()
	if len(files) != 0 {
		t.Errorf("Expected 0 files (malformed announce), got %d", len(files))
	}
}

func TestServerHandlePeerAnnounceInvalidHash(t *testing.T) {
	srv, conn := setupMockServerConn(t)
	defer func() { _ = srv.Shutdown() }()

	invalidHash := protocol.FileHash{0x01, 0x02, 0x03}
	announce := &protocol.PeerAnnounce{
		FileCount: 1,
		Files: []protocol.FileEntry{
			{Hash: invalidHash, Name: "file1.txt", Size: 1024},
		},
	}

	srv.handleMessage(t.Context(), conn, announce)

	files := srv.store.ListFiles()
	if len(files) != 0 {
		t.Errorf("Expected 0 files (invalid hash), got %d", len(files))
	}
}

func TestServerHandleFileListReqEmpty(t *testing.T) {
	srv, conn := setupMockServerConn(t)
	defer func() { _ = srv.Shutdown() }()

	srv.handleMessage(t.Context(), conn, &protocol.FileListReq{})

	if len(conn.sentMsgs) != 1 {
		t.Fatalf("Expected 1 sent message, got %d", len(conn.sentMsgs))
	}

	res, ok := conn.sentMsgs[0].(*protocol.FileListRes)
	if !ok {
		t.Fatalf("Expected FileListRes, got %T", conn.sentMsgs[0])
	}

	if len(res.Files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(res.Files))
	}
}

func TestServerHandleFileListReq(t *testing.T) {
	srv, conn := setupMockServerConn(t)
	defer func() { _ = srv.Shutdown() }()

	fileName := "testfile.txt"
	fileSize := uint64(1024)
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s%d", fileName, fileSize)))

	announce := &protocol.PeerAnnounce{
		FileCount: 1,
		Files: []protocol.FileEntry{
			{Hash: hash, Name: fileName, Size: fileSize},
		},
	}
	srv.handleMessage(t.Context(), conn, announce)

	srv.handleMessage(t.Context(), conn, &protocol.FileListReq{})

	if len(conn.sentMsgs) != 1 {
		t.Fatalf("Expected 1 sent message, got %d", len(conn.sentMsgs))
	}

	res, ok := conn.sentMsgs[0].(*protocol.FileListRes)
	if !ok {
		t.Fatalf("Expected FileListRes, got %T", conn.sentMsgs[0])
	}

	if len(res.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(res.Files))
	}
}

func TestServerHandleConnReceiveError(t *testing.T) {
	listener := newMockListener()
	srv, _ := NewServer(Config{
		Listener: listener,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	defer func() { _ = srv.Shutdown() }()

	conn := newMockConn("test-peer")
	conn.recvErr = errors.New("receive error")

	srv.handleConn(t.Context(), conn)

	if !conn.closed {
		t.Error("Expected connection to be closed after receive error")
	}
}

func setupMockServerConn(t *testing.T) (*Server, *mockConn) {
	t.Helper()

	listener := newMockListener()
	srv, err := NewServer(Config{
		Listener: listener,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	conn := newMockConn("test-peer")
	return srv, conn
}
