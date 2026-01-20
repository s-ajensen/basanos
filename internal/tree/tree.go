package tree

import (
	"fmt"
	"path/filepath"

	"basanos/internal/fs"
	"basanos/internal/spec"
)

type SpecTree struct {
	Path     string
	Context  *spec.Context
	Children []*SpecTree
}

func LoadContext(filesystem fs.FileSystem, dirPath string) (*spec.Context, error) {
	contextFile := filepath.Join(dirPath, "context.yaml")
	data, err := filesystem.ReadFile(contextFile)
	if err != nil {
		return nil, err
	}
	ctx, err := spec.ParseContext(data)
	if err != nil {
		return nil, err
	}
	errors := spec.Validate(ctx, contextFile)
	if len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %s: %s: %s", errors[0].File, errors[0].Path, errors[0].Message)
	}
	return ctx, nil
}

func LoadSpecTreeRecursive(filesystem fs.FileSystem, rootFilePath string, rootSpecPath string) (*SpecTree, error) {
	ctx, err := LoadContext(filesystem, rootFilePath)
	if err != nil {
		return nil, err
	}

	tree := &SpecTree{Path: rootSpecPath, Context: ctx}

	entries, err := filesystem.ReadDir(rootFilePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		childFilePath := filepath.Join(rootFilePath, entry.Name())
		contextFile := filepath.Join(childFilePath, "context.yaml")
		if _, err := filesystem.Stat(contextFile); err != nil {
			continue
		}
		childSpecPath := filepath.Join(rootSpecPath, entry.Name())
		child, err := LoadSpecTreeRecursive(filesystem, childFilePath, childSpecPath)
		if err != nil {
			return nil, err
		}
		tree.Children = append(tree.Children, child)
	}

	return tree, nil
}

func LoadSpecTree(filesystem fs.FileSystem, rootPath string) (*SpecTree, error) {
	return LoadSpecTreeRecursive(filesystem, rootPath, filepath.Base(rootPath))
}
