package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveFile(t *testing.T) {
	directory := t.TempDir()
	savedPath, err := NewFileWriter(directory).SaveFile("sha256-abc.json", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if savedPath != filepath.Join(directory, "sha256-abc.json") {
		t.Errorf("savedPath = %q", savedPath)
	}
	if body, _ := os.ReadFile(savedPath); string(body) != "hello" {
		t.Errorf("body = %q", body)
	}
	entries, _ := os.ReadDir(directory)
	if len(entries) != 1 {
		t.Errorf("temporary file left behind: %v", entries)
	}
}
