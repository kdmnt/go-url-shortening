package types

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLResponseJSONTags(t *testing.T) {
	response := URLResponse{
		ShortURL:    "abc123",
		OriginalURL: "https://example.com",
	}

	jsonData, err := json.Marshal(response)
	require.NoError(t, err, "Failed to marshal URLResponse")

	var unmarshaled map[string]interface{}
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON")

	expectedKeys := []string{"short_url", "original_url"}
	for _, key := range expectedKeys {
		_, ok := unmarshaled[key]
		assert.True(t, ok, "Expected JSON key %q not found", key)
	}
}

func TestURLRequestJSONTags(t *testing.T) {
	request := URLRequest{
		URL: "https://example.com",
	}

	jsonData, err := json.Marshal(request)
	require.NoError(t, err, "Failed to marshal URLRequest")

	var unmarshaled map[string]interface{}
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON")

	_, ok := unmarshaled["url"]
	assert.True(t, ok, "Expected JSON key \"url\" not found")
}

func TestURLRequestValidationTag(t *testing.T) {
	field, ok := reflect.TypeOf(URLRequest{}).FieldByName("URL")
	require.True(t, ok, "URL field not found in URLRequest struct")

	tag := field.Tag.Get("validate")
	require.Equal(t, "required,url", tag, "Unexpected validate tag for URL field")
}
