package compression

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

const PKG_COMPRESSION = "compression"

var compressionExts = map[string]string{
	"none":  "",
	"gzip":  ".gz",
	"bzip2": ".bz2",
	"lz4":   ".lz4",
	"zstd":  ".zst",
}

// CompressFile compresses a file using the specified algorithm
func CompressFile(inputPath, outputPath, algorithm string) error {
	switch algorithm {
	case "none":
		return nil
	case "gzip":
		return compressGzip(inputPath, outputPath)
	case "bzip2":
		return compressBzip2(inputPath, outputPath)
	case "lz4":
		return compressLz4(inputPath, outputPath)
	case "zstd":
		return compressZstd(inputPath, outputPath)
	default:
		return fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}
}

// GetCompressionExt returns the file extension for the specified compression algorithm
func GetCompressionExt(algorithm string) string {
	ext, ok := compressionExts[algorithm]
	if !ok {
		return ""
	}
	return ext
}

// AllCompressionExts returns a slice of all supported compression extensions
func AllCompressionExts() []string {
	exts := make([]string, 0, len(compressionExts))
	for _, ext := range compressionExts {
		if ext != "" {
			exts = append(exts, ext)
		}
	}
	return exts
}

// GetCompressionAlgorithmFromExt returns the compression algorithm from file extension
func GetCompressionAlgorithmFromExt(filename string) string {
	for alg, ext := range compressionExts {
		if ext != "" && strings.HasSuffix(filename, ext) {
			return alg
		}
	}
	return "none"
}

// IsCompressed checks if a file has a compression extension
func IsCompressed(filename string) bool {
	return GetCompressionAlgorithmFromExt(filename) != "none"
}

// ResolveCompressedFilename returns potential compressed versions of a .db file
// If the input already has a compression extension, returns it as-is
// If the input ends with .db, returns all possible compressed versions
func ResolveCompressedFilename(filename string) []string {
	// If already compressed, return as-is
	if IsCompressed(filename) {
		return []string{filename}
	}

	// If ends with .db, generate compressed alternatives
	if strings.HasSuffix(filename, ".db") {
		candidates := []string{}
		base := filename // Keep original .db file as last option

		// Add compressed versions in order of preference (zstd first as it's default)
		for _, alg := range []string{"zstd", "gzip", "lz4", "bzip2"} {
			if ext := GetCompressionExt(alg); ext != "" {
				candidates = append(candidates, base+ext)
			}
		}

		// Add uncompressed as fallback
		candidates = append(candidates, base)
		return candidates
	}

	// For other files, return as-is
	return []string{filename}
}

// ResolveCompressedFile attempts to find the best available version of a snapshot file locally.
// If the filename ends with .db, it checks for compressed versions first, then falls back to uncompressed.
// Returns the actual filename found and whether it was found.
func ResolveCompressedFile(filename string) (string, bool) {
	candidates := ResolveCompressedFilename(filename)

	// Try each candidate in order of preference
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Size() > 0 {
			return candidate, true
		}
	}

	// No file found
	return filename, false
}

// DecompressFile decompresses a file using the algorithm detected from its extension
func DecompressFile(inputPath, outputPath string) error {
	algorithm := GetCompressionAlgorithmFromExt(inputPath)
	if algorithm == "none" {
		// No compression, just copy the file
		return copyFile(inputPath, outputPath)
	}

	switch algorithm {
	case "gzip":
		return decompressGzip(inputPath, outputPath)
	case "bzip2":
		return decompressBzip2(inputPath, outputPath)
	case "lz4":
		return decompressLz4(inputPath, outputPath)
	case "zstd":
		return decompressZstd(inputPath, outputPath)
	default:
		return fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// compressGzip compresses a file using gzip
func compressGzip(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	gzipWriter := gzip.NewWriter(destFile)
	defer gzipWriter.Close()

	_, err = io.Copy(gzipWriter, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to compress with gzip: %w", err)
	}

	return nil
}

// compressBzip2 compresses a file using bzip2
func compressBzip2(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	bzip2Writer, err := bzip2.NewWriter(destFile, nil)
	if err != nil {
		return fmt.Errorf("failed to create bzip2 writer: %w", err)
	}
	defer bzip2Writer.Close()

	_, err = io.Copy(bzip2Writer, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to compress with bzip2: %w", err)
	}

	return nil
}

// compressLz4 compresses a file using lz4
func compressLz4(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	lz4Writer := lz4.NewWriter(destFile)
	defer lz4Writer.Close()

	_, err = io.Copy(lz4Writer, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to compress with lz4: %w", err)
	}

	return nil
}

// compressZstd compresses a file using zstd
func compressZstd(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	zstdWriter, err := zstd.NewWriter(destFile)
	if err != nil {
		return fmt.Errorf("failed to create zstd writer: %w", err)
	}
	defer zstdWriter.Close()

	_, err = io.Copy(zstdWriter, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to compress with zstd: %w", err)
	}

	return nil
}

// decompressGzip decompresses a gzip file
func decompressGzip(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	gzipReader, err := gzip.NewReader(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	_, err = io.Copy(destFile, gzipReader)
	if err != nil {
		return fmt.Errorf("failed to decompress with gzip: %w", err)
	}

	return nil
}

// decompressBzip2 decompresses a bzip2 file
func decompressBzip2(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	bzip2Reader, err := bzip2.NewReader(sourceFile, nil)
	if err != nil {
		return fmt.Errorf("failed to create bzip2 reader: %w", err)
	}
	defer bzip2Reader.Close()

	_, err = io.Copy(destFile, bzip2Reader)
	if err != nil {
		return fmt.Errorf("failed to decompress with bzip2: %w", err)
	}

	return nil
}

// decompressLz4 decompresses an lz4 file
func decompressLz4(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	lz4Reader := lz4.NewReader(sourceFile)

	_, err = io.Copy(destFile, lz4Reader)
	if err != nil {
		return fmt.Errorf("failed to decompress with lz4: %w", err)
	}

	return nil
}

// decompressZstd decompresses a zstd file
func decompressZstd(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	zstdReader, err := zstd.NewReader(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer zstdReader.Close()

	_, err = io.Copy(destFile, zstdReader)
	if err != nil {
		return fmt.Errorf("failed to decompress with zstd: %w", err)
	}

	return nil
}
