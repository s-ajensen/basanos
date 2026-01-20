package fs

import (
	"io/fs"
	"os"
	"strings"
	"time"
)

type MemoryFS struct {
	files map[string][]byte
	dirs  map[string]bool
}

func NewMemoryFS() *MemoryFS {
	return &MemoryFS{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *MemoryFS) AddFile(path string, content []byte) {
	m.files[path] = content
}

func (m *MemoryFS) AddDir(path string) {
	m.dirs[path] = true
}

func (m *MemoryFS) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (m *MemoryFS) WriteFile(path string, data []byte) error {
	m.files[path] = data
	return nil
}

func (m *MemoryFS) AppendFile(path string, data []byte) error {
	m.files[path] = append(m.files[path], data...)
	return nil
}

func (m *MemoryFS) ReadDir(path string) ([]os.DirEntry, error) {
	if !m.dirs[path] {
		return nil, os.ErrNotExist
	}
	var entries []os.DirEntry
	seen := make(map[string]bool)
	prefix := path + "/"
	for filePath := range m.files {
		if strings.HasPrefix(filePath, prefix) {
			rest := strings.TrimPrefix(filePath, prefix)
			name := strings.Split(rest, "/")[0]
			if !seen[name] {
				seen[name] = true
				entries = append(entries, &memDirEntry{name: name, isDir: strings.Contains(rest, "/")})
			}
		}
	}
	for dirPath := range m.dirs {
		if strings.HasPrefix(dirPath, prefix) {
			rest := strings.TrimPrefix(dirPath, prefix)
			name := strings.Split(rest, "/")[0]
			if !seen[name] {
				seen[name] = true
				entries = append(entries, &memDirEntry{name: name, isDir: true})
			}
		}
	}
	return entries, nil
}

func (m *MemoryFS) Stat(path string) (os.FileInfo, error) {
	if _, ok := m.files[path]; ok {
		return &memFileInfo{name: path, isDir: false}, nil
	}
	if m.dirs[path] {
		return &memFileInfo{name: path, isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (m *MemoryFS) Abs(path string) (string, error) {
	return "/" + path, nil
}

type memDirEntry struct {
	name  string
	isDir bool
}

func (e *memDirEntry) Name() string               { return e.name }
func (e *memDirEntry) IsDir() bool                { return e.isDir }
func (e *memDirEntry) Type() fs.FileMode          { return 0 }
func (e *memDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

type memFileInfo struct {
	name  string
	isDir bool
}

func (f *memFileInfo) Name() string       { return f.name }
func (f *memFileInfo) Size() int64        { return 0 }
func (f *memFileInfo) Mode() fs.FileMode  { return 0 }
func (f *memFileInfo) ModTime() time.Time { return time.Time{} }
func (f *memFileInfo) IsDir() bool        { return f.isDir }
func (f *memFileInfo) Sys() any           { return nil }

func (m *MemoryFS) AllFiles() []string {
	var files []string
	for path := range m.files {
		files = append(files, path)
	}
	return files
}
