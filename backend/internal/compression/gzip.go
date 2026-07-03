package compression

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/sanskarpan/db-backup/internal/types"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// MaxDecompressedSize is the upper bound (in bytes) on the amount of data a
// single Decompress call will write, guarding against decompression-bomb
// attacks (gosec G110). It defaults to 1 TiB, which is comfortably larger than
// any realistic single backup artifact, and can be raised by operators that
// legitimately restore larger streams.
var MaxDecompressedSize int64 = 1 << 40 // 1 TiB

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

	// Bound the number of bytes written to defend against decompression bombs
	// (gosec G110). io.CopyN with MaxDecompressedSize+1 lets us detect when the
	// input would exceed the configured ceiling.
	written, err := io.CopyN(output, reader, MaxDecompressedSize+1)
	if err != nil && err != io.EOF { //nolint:errorlint // io.CopyN documents a bare io.EOF sentinel on clean completion
		return pkgErrors.ErrDecompressionFailed(err)
	}
	if written > MaxDecompressedSize {
		return pkgErrors.ErrDecompressionFailed(
			fmt.Errorf("decompressed size exceeds limit of %d bytes", MaxDecompressedSize),
		)
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
