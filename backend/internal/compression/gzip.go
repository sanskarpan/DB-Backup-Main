package compression

import (
	"compress/gzip"
	"io"

	"github.com/sanskarpan/db-backup/internal/types"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// GzipCompressor implements gzip compression.
type GzipCompressor struct{}

// NewGzipCompressor creates a new gzip compressor.
func NewGzipCompressor() *GzipCompressor {
	return &GzipCompressor{}
}

// Compress compresses data using gzip.
func (c *GzipCompressor) Compress(input io.Reader, output io.Writer, level int) error {
	if level < 0 || level > 9 {
		level = gzip.DefaultCompression
	}

	writer, err := gzip.NewWriterLevel(output, level)
	if err != nil {
		return pkgErrors.ErrCompressionFailed(err)
	}
	defer writer.Close()

	if _, err := io.Copy(writer, input); err != nil {
		return pkgErrors.ErrCompressionFailed(err)
	}

	return writer.Close()
}

// Decompress decompresses gzip data.
func (c *GzipCompressor) Decompress(input io.Reader, output io.Writer) error {
	reader, err := gzip.NewReader(input)
	if err != nil {
		return pkgErrors.ErrDecompressionFailed(err)
	}
	defer reader.Close()

	if _, err := io.Copy(output, reader); err != nil {
		return pkgErrors.ErrDecompressionFailed(err)
	}

	return nil
}

// GetType returns the compression type.
func (c *GzipCompressor) GetType() types.CompressionType {
	return types.CompressionGzip
}

// GetExtension returns the file extension.
func (c *GzipCompressor) GetExtension() string {
	return ".gz"
}
