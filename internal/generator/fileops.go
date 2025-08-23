package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"autoteam/internal/config"
)

// FileOperations handles file and directory operations for the generator
type FileOperations struct{}

// NewFileOperations creates a new FileOperations instance
func NewFileOperations() *FileOperations {
	return &FileOperations{}
}

// EnsureDirectory creates a directory if it doesn't exist
func (f *FileOperations) EnsureDirectory(path string, perm os.FileMode) error {
	if err := f.ValidatePath(path); err != nil {
		return fmt.Errorf("invalid path %s: %w", path, err)
	}

	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	return nil
}

// RemoveIfExists removes a file or directory if it exists
func (f *FileOperations) RemoveIfExists(path string) error {
	if err := f.ValidatePath(path); err != nil {
		return fmt.Errorf("invalid path %s: %w", path, err)
	}

	if _, err := os.Lstat(path); err == nil {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	return nil
}

// WriteFileIfNotExists writes content to a file only if it doesn't exist
func (f *FileOperations) WriteFileIfNotExists(path string, content []byte, perm os.FileMode) error {
	if err := f.ValidatePath(path); err != nil {
		return fmt.Errorf("invalid path %s: %w", path, err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, content, perm); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
	}

	return nil
}

// CopyDirectory recursively copies a directory from src to dst
func (f *FileOperations) CopyDirectory(src, dst string) error {
	if err := f.ValidatePath(src); err != nil {
		return fmt.Errorf("invalid source path %s: %w", src, err)
	}
	if err := f.ValidatePath(dst); err != nil {
		return fmt.Errorf("invalid destination path %s: %w", dst, err)
	}

	// Create destination directory
	if err := f.EnsureDirectory(dst, config.DirPerm); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory %s: %w", src, err)
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := f.CopyDirectory(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy subdirectory %s: %w", entry.Name(), err)
			}
		} else {
			// Copy file
			if err := f.CopyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// CopyFile copies a single file from src to dst with preserved permissions
func (f *FileOperations) CopyFile(src, dst string) error {
	if err := f.ValidatePath(src); err != nil {
		return fmt.Errorf("invalid source path %s: %w", src, err)
	}
	if err := f.ValidatePath(dst); err != nil {
		return fmt.Errorf("invalid destination path %s: %w", dst, err)
	}

	// Read source file
	srcData, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file %s: %w", src, err)
	}

	// Get source file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source file info for %s: %w", src, err)
	}

	// Write destination file with same permissions
	if err := os.WriteFile(dst, srcData, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to write destination file %s: %w", dst, err)
	}

	return nil
}

// DirectoryExists checks if a directory exists
func (f *FileOperations) DirectoryExists(path string) bool {
	if err := f.ValidatePath(path); err != nil {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// ValidatePath performs basic validation on file paths
func (f *FileOperations) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for path traversal attempts
	cleanPath := filepath.Clean(path)
	if cleanPath != path && !filepath.IsAbs(path) {
		return fmt.Errorf("potentially unsafe path detected")
	}

	return nil
}

// CreateWorkerDirectoryStructure creates the complete directory structure for a worker
func (f *FileOperations) CreateWorkerDirectoryStructure(workerName string) error {
	workerDir := filepath.Join(config.WorkersDir, workerName)

	// Create main worker directory - subdirectories will be created as needed by individual layers
	if err := f.EnsureDirectory(workerDir, config.DirPerm); err != nil {
		return fmt.Errorf("failed to create worker directory for worker %s: %w", workerName, err)
	}

	return nil
}

// FileExists checks if a file exists
func (f *FileOperations) FileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// SetPermissions sets the permissions on a file or directory
func (f *FileOperations) SetPermissions(path string, perm os.FileMode) error {
	if err := f.ValidatePath(path); err != nil {
		return fmt.Errorf("invalid path %s: %w", path, err)
	}
	
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", path, err)
	}
	
	return nil
}
