package node

import (
	"strings"
	"testing"
)

func TestHashFile(t *testing.T) {
	data := "hello world"
	hash, err := HashFile(strings.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SHA256 of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("expected %s, got %s", expected, hash)
	}
}

func TestHashFile_Empty(t *testing.T) {
	hash, err := HashFile(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SHA256 of empty string
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Errorf("expected %s, got %s", expected, hash)
	}
}

func TestCalculateTotalChunks(t *testing.T) {
	tests := []struct {
		fileSize  int64
		chunkSize int64
		expected  int
	}{
		{1024, 256, 4},
		{1000, 256, 4},
		{256, 256, 1},
		{0, 256, 0},
		{1, 256, 1},
		{257, 256, 2},
		{100, 0, 0}, // zero chunk size
	}

	for _, tt := range tests {
		result := CalculateTotalChunks(tt.fileSize, tt.chunkSize)
		if result != tt.expected {
			t.Errorf("CalculateTotalChunks(%d, %d) = %d, want %d",
				tt.fileSize, tt.chunkSize, result, tt.expected)
		}
	}
}

func TestExtractFileName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/home/user/file.txt", "file.txt"},
		{"file.txt", "file.txt"},
		{"/file.txt", "file.txt"},
		{"a/b/c/d.txt", "d.txt"},
		{"", ""},
	}

	for _, tt := range tests {
		result := ExtractFileName(tt.path)
		if result != tt.expected {
			t.Errorf("ExtractFileName(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestBuildDownloadPath(t *testing.T) {
	result := BuildDownloadPath("0", "test.txt")
	expected := "downloads/daemon-0/test.txt"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildSocketPath(t *testing.T) {
	result := BuildSocketPath("1")
	expected := "/tmp/pit-daemon-1.sock"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
