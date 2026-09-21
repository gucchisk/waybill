package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileWriter saves files under the given directory.
type FileWriter struct {
	directory string
}

func NewFileWriter(directory string) *FileWriter {
	return &FileWriter{directory: directory}
}

// SaveFile writes to a temporary file and then renames it, so a failure never leaves a partial file.
func (writer *FileWriter) SaveFile(fileName string, content io.Reader) (string, error) {
	savedPath := filepath.Join(writer.directory, filepath.Base(fileName))

	temporaryFile, err := os.CreateTemp(writer.directory, ".waybill-*")
	if err != nil {
		return "", fmt.Errorf("create temporary file: %w", err)
	}
	defer os.Remove(temporaryFile.Name())

	if _, err := io.Copy(temporaryFile, content); err != nil {
		temporaryFile.Close()
		return "", fmt.Errorf("write %s: %w", savedPath, err)
	}
	if err := temporaryFile.Close(); err != nil {
		return "", fmt.Errorf("close %s: %w", savedPath, err)
	}
	if err := os.Chmod(temporaryFile.Name(), 0o644); err != nil {
		return "", fmt.Errorf("chmod %s: %w", savedPath, err)
	}
	if err := os.Rename(temporaryFile.Name(), savedPath); err != nil {
		return "", fmt.Errorf("rename to %s: %w", savedPath, err)
	}
	return savedPath, nil
}
