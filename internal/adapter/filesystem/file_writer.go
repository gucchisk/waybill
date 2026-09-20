package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileWriter は指定ディレクトリ配下へファイルを保存する。
type FileWriter struct {
	directory string
}

func NewFileWriter(directory string) *FileWriter {
	return &FileWriter{directory: directory}
}

// SaveFile は一時ファイルへ書き込んでから rename することで、失敗時に中途半端なファイルを残さない。
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
