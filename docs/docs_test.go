package docs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggo/swag"
)

func TestJSON(t *testing.T) {
	var doc map[string]any
	require.NoError(t, json.Unmarshal(JSON("example.com", "https"), &doc))
	assert.Equal(t, "2.0", doc["swagger"])
	assert.Equal(t, "example.com", doc["host"])
	assert.Equal(t, []any{"https"}, doc["schemes"])
	paths := doc["paths"].(map[string]any)
	assert.Contains(t, paths, "/posts/{id}")
	assert.Contains(t, paths, "/public/posts/{slug}")
	assert.Contains(t, paths["/posts"], "post")

	var bare map[string]any
	require.NoError(t, json.Unmarshal(JSON("", ""), &bare))
	assert.NotContains(t, bare, "host")
	assert.NotContains(t, bare, "schemes")
}

func TestSpecRegistered(t *testing.T) {
	assert.Equal(t, string(JSON("", "")), spec{}.ReadDoc())
	doc, err := swag.ReadDoc()
	require.NoError(t, err)
	assert.Contains(t, doc, "Delcom Blog API")
}

func TestContains(t *testing.T) {
	assert.True(t, contains("/posts/{id}", "{id}"))
	assert.False(t, contains("/posts", "{id}"))
}
