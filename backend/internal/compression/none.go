package compression

import (
	"io"

	"github.com/sanskarpan/db-backup/internal/types"
)

// NoneCompressor implements pass-through (no compression).
type NoneCompressor struct{}

// NewNoneCompressor creates a new pass-through compressor.
func NewNoneCompressor() *NoneCompressor {
	return &NoneCompressor{}
}

// Compress simply copies data without compression.
func (c *NoneCompressor) Compress(input io.Reader, output io.Writer, level int) error {
	_, err := io.Copy(output, input)
	return err
}

// Decompress simply copies data without decompression.
func (c *NoneCompressor) Decompress(input io.Reader, output io.Writer) error {
	_, err := io.Copy(output, input)
	return err
}

// GetType returns the compression type.
func (c *NoneCompressor) GetType() types.CompressionType {
	return types.CompressionNone
}

// GetExtension returns the file extension.
func (c *NoneCompressor) GetExtension() string {
	return ""
}
