package compression

import (
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/sanskarpan/db-backup/internal/types"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// ZstdCompressor implements zstandard compression
type ZstdCompressor struct{}

// NewZstdCompressor creates a new zstd compressor
func NewZstdCompressor() *ZstdCompressor {
	return &ZstdCompressor{}
}

// Compress compresses data using zstd
func (c *ZstdCompressor) Compress(input io.Reader, output io.Writer, level int) error {
	// Map level (1-9) to zstd levels
	var encoderLevel zstd.EncoderLevel
	switch {
	case level <= 3:
		encoderLevel = zstd.SpeedFastest
	case level <= 6:
		encoderLevel = zstd.SpeedDefault
	default:
		encoderLevel = zstd.SpeedBetterCompression
	}

	writer, err := zstd.NewWriter(output, zstd.WithEncoderLevel(encoderLevel))
	if err != nil {
		return pkgErrors.ErrCompressionFailed(err)
	}
	defer writer.Close()

	if _, err := io.Copy(writer, input); err != nil {
		return pkgErrors.ErrCompressionFailed(err)
	}

	return writer.Close()
}

// Decompress decompresses zstd data
func (c *ZstdCompressor) Decompress(input io.Reader, output io.Writer) error {
	reader, err := zstd.NewReader(input)
	if err != nil {
		return pkgErrors.ErrDecompressionFailed(err)
	}
	defer reader.Close()

	if _, err := io.Copy(output, reader); err != nil {
		return pkgErrors.ErrDecompressionFailed(err)
	}

	return nil
}

// GetType returns the compression type
func (c *ZstdCompressor) GetType() types.CompressionType {
	return types.CompressionZstd
}

// GetExtension returns the file extension
func (c *ZstdCompressor) GetExtension() string {
	return ".zst"
}
