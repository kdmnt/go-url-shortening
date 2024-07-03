package utils

import (
	"crypto/rand"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// errorReader is a mock io.Reader that always returns an error
type errorReader struct{}

func (r *errorReader) Read([]byte) (n int, err error) {
	return 0, errors.New("mocked random number generation error")
}

func TestGenerateShortURL(t *testing.T) {
	t.Run("Basic Generation", func(t *testing.T) {
		shortURL, err := GenerateShortURL()
		require.NoError(t, err, "GenerateShortURL() should not return an error")
		require.Len(t, shortURL, shortURLLength, "Generated short URL should have the correct length")
		for _, char := range shortURL {
			assert.Contains(t, charset, string(char), "Generated short URL should only contain valid characters")
		}
	})

	t.Run("Multiple Generations", func(t *testing.T) {
		generatedURLs := make(map[string]int)
		totalURLs := 1000000
		for i := 0; i < totalURLs; i++ {
			shortURL, err := GenerateShortURL()
			require.NoError(t, err, "GenerateShortURL() should not return an error")
			generatedURLs[shortURL]++
		}

		duplicates := make(map[string]int)
		for url, count := range generatedURLs {
			if count > 1 {
				duplicates[url] = count
			}
		}

		uniqueURLs := len(generatedURLs)
		duplicateCount := len(duplicates)
		totalDuplicates := totalURLs - uniqueURLs

		t.Logf("Total URLs generated: %d", totalURLs)
		t.Logf("Unique URLs: %d", uniqueURLs)
		t.Logf("Number of duplicated URLs: %d", duplicateCount)
		t.Logf("Total duplicate instances: %d", totalDuplicates)
		t.Logf("Duplication rate: %.6f%%", float64(totalDuplicates)/float64(totalURLs)*100)

		if len(duplicates) > 0 {
			t.Logf("Duplicates: %v", duplicates)
		}

		assert.Empty(t, duplicates, "No short URLs should be duplicated. Duplicates: %v", duplicates)
	})

	t.Run("Error Handling", func(t *testing.T) {
		// Mock the rand.Reader to return an error
		originalReader := rand.Reader
		rand.Reader = &errorReader{}
		defer func() { rand.Reader = originalReader }()

		_, err := GenerateShortURL()
		assert.Error(t, err, "GenerateShortURL() should return an error when random number generation fails")
		assert.Contains(t, err.Error(), "mocked random number generation error", "Error message should contain the mocked error")
	})
}

// BenchmarkGenerateShortURL measures the performance of the GenerateShortURL function.
// It's used to quantify the speed of short URL generation and detect performance regressions.
func BenchmarkGenerateShortURL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GenerateShortURL()
		if err != nil {
			b.Fatal(err)
		}
	}
}
