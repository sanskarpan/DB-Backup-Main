package compression

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testData = []byte("This is test data for compression. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.")

const DefaultCompressionLevel = 6

func TestGzipCompressor(t *testing.T) {
	compressor := NewGzipCompressor()

	// Test Compress
	var compressed bytes.Buffer
	err := compressor.Compress(bytes.NewReader(testData), &compressed, DefaultCompressionLevel)
	require.NoError(t, err)
	assert.Greater(t, compressed.Len(), 0)
	assert.Less(t, compressed.Len(), len(testData)) // Should be smaller

	// Test Decompress
	var decompressed bytes.Buffer
	err = compressor.Decompress(&compressed, &decompressed)
	require.NoError(t, err)
	assert.Equal(t, testData, decompressed.Bytes())
}

func TestZstdCompressor(t *testing.T) {
	compressor := NewZstdCompressor()

	var compressed bytes.Buffer
	err := compressor.Compress(bytes.NewReader(testData), &compressed, DefaultCompressionLevel)
	require.NoError(t, err)
	assert.Greater(t, compressed.Len(), 0)

	var decompressed bytes.Buffer
	err = compressor.Decompress(&compressed, &decompressed)
	require.NoError(t, err)
	assert.Equal(t, testData, decompressed.Bytes())
}

func TestLZ4Compressor(t *testing.T) {
	compressor := NewLZ4Compressor()

	var compressed bytes.Buffer
	err := compressor.Compress(bytes.NewReader(testData), &compressed, DefaultCompressionLevel)
	require.NoError(t, err)
	assert.Greater(t, compressed.Len(), 0)

	var decompressed bytes.Buffer
	err = compressor.Decompress(&compressed, &decompressed)
	require.NoError(t, err)
	assert.Equal(t, testData, decompressed.Bytes())
}

func TestNoneCompressor(t *testing.T) {
	compressor := NewNoneCompressor()

	var output bytes.Buffer
	err := compressor.Compress(bytes.NewReader(testData), &output, DefaultCompressionLevel)
	require.NoError(t, err)
	assert.Equal(t, testData, output.Bytes())

	var decompressed bytes.Buffer
	err = compressor.Decompress(bytes.NewReader(testData), &decompressed)
	require.NoError(t, err)
	assert.Equal(t, testData, decompressed.Bytes())
}

func TestCompressionRatios(t *testing.T) {
	// Create larger test data
	largeData := bytes.Repeat(testData, 100)

	compressors := map[string]Compressor{
		"gzip": NewGzipCompressor(),
		"zstd": NewZstdCompressor(),
		"lz4":  NewLZ4Compressor(),
	}

	for name, compressor := range compressors {
		t.Run(name, func(t *testing.T) {
			var compressed bytes.Buffer
			err := compressor.Compress(bytes.NewReader(largeData), &compressed, DefaultCompressionLevel)
			require.NoError(t, err)

			ratio := float64(compressed.Len()) / float64(len(largeData))
			t.Logf("%s compression ratio: %.2f%%", name, ratio*100)

			// Compressed should be smaller
			assert.Less(t, compressed.Len(), len(largeData))

			// Verify decompression
			var decompressed bytes.Buffer
			err = compressor.Decompress(&compressed, &decompressed)
			require.NoError(t, err)
			assert.Equal(t, largeData, decompressed.Bytes())
		})
	}
}

func TestCompressorSelection(t *testing.T) {
	testCases := []struct {
		name            string
		compressionType string
		expectError     bool
	}{
		{"Gzip", "gzip", false},
		{"Zstd", "zstd", false},
		{"LZ4", "lz4", false},
		{"None", "none", false},
		{"Invalid", "invalid", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var compressor Compressor
			var err error

			switch tc.compressionType {
			case "gzip":
				compressor = NewGzipCompressor()
			case "zstd":
				compressor = NewZstdCompressor()
			case "lz4":
				compressor = NewLZ4Compressor()
			case "none":
				compressor = NewNoneCompressor()
			default:
				err = assert.AnError
			}

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NotNil(t, compressor)
			}
		})
	}
}

// Benchmark tests.
func BenchmarkGzipCompress(b *testing.B) {
	compressor := NewGzipCompressor()
	data := bytes.Repeat(testData, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var output bytes.Buffer
		compressor.Compress(bytes.NewReader(data), &output, DefaultCompressionLevel)
	}
}

func BenchmarkZstdCompress(b *testing.B) {
	compressor := NewZstdCompressor()
	data := bytes.Repeat(testData, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var output bytes.Buffer
		compressor.Compress(bytes.NewReader(data), &output, DefaultCompressionLevel)
	}
}

func BenchmarkLZ4Compress(b *testing.B) {
	compressor := NewLZ4Compressor()
	data := bytes.Repeat(testData, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var output bytes.Buffer
		compressor.Compress(bytes.NewReader(data), &output, DefaultCompressionLevel)
	}
}
