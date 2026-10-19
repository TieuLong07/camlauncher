package sync

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractArchive expands a trusted modpack archive while rejecting paths outside destination.
func ExtractArchive(archivePath, destination string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	root, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	for _, entry := range reader.File {
		target := filepath.Join(root, filepath.FromSlash(entry.Name))
		cleanTarget, err := filepath.Abs(target)
		if err != nil || (cleanTarget != root && !strings.HasPrefix(cleanTarget, root+string(os.PathSeparator))) {
			return fmt.Errorf("archive chứa đường dẫn không hợp lệ: %s", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return err
		}
		source, err := entry.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, entry.Mode())
		if err == nil {
			_, err = io.Copy(out, source)
			closeErr := out.Close()
			if err == nil {
				err = closeErr
			}
		}
		_ = source.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// ExtractSpecificFiles extracts only requested relative paths from a zip archive.
func ExtractSpecificFiles(archivePath, destination string, filesToExtract []string) error {
	if len(filesToExtract) == 0 {
		return nil
	}
	lookup := make(map[string]bool)
	for _, f := range filesToExtract {
		clean := filepath.ToSlash(filepath.Clean(f))
		clean = strings.TrimPrefix(clean, "/")
		lookup[strings.ToLower(clean)] = true
		lookup[strings.ToLower(filepath.Base(clean))] = true
	}

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	root, err := filepath.Abs(destination)
	if err != nil {
		return err
	}

	for _, entry := range reader.File {
		entryClean := filepath.ToSlash(filepath.Clean(entry.Name))
		entryClean = strings.TrimPrefix(entryClean, "/")
		baseName := filepath.Base(entryClean)

		if !lookup[strings.ToLower(entryClean)] && !lookup[strings.ToLower(baseName)] {
			continue
		}

		target := filepath.Join(root, filepath.FromSlash(entry.Name))
		cleanTarget, err := filepath.Abs(target)
		if err != nil || (cleanTarget != root && !strings.HasPrefix(cleanTarget, root+string(os.PathSeparator))) {
			continue
		}

		if entry.FileInfo().IsDir() {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return err
		}

		source, err := entry.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, entry.Mode())
		if err == nil {
			_, err = io.Copy(out, source)
			_ = out.Close()
		}
		_ = source.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
