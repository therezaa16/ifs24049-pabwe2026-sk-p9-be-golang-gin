package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jsonServer(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// ---------- SearchClient ----------

func TestSearchShapes(t *testing.T) {
	cases := map[string]string{
		"results": `{"results":[{"title":"A","url":"http://a","snippet":"s"},{"title":"B","link":"http://b","content":"c"},{"title":"tanpa url"},"bukan-objek",{"name":"C","href":"http://c","description":"d"},{"title":"D","url":"http://d"}]}`,
		"nested":  `{"data":{"results":[{"title":"A","url":"http://a"},{"title":"B","url":"http://b"},{"title":"C","url":"http://c"},{"title":"D","url":"http://d"}]}}`,
		"array":   `[{"title":"A","url":"http://a"},{"title":"B","url":"http://b"},{"title":"C","url":"http://c"}]`,
	}
	for name, body := range cases {
		srv := jsonServer(200, body)
		got, err := NewSearchClient(srv.URL, "k").Search(context.Background(), "golang", 3)
		srv.Close()
		require.NoError(t, err, name)
		assert.NotEmpty(t, got, name)
		assert.LessOrEqual(t, len(got), 3, name)
	}
}

func TestSearchEmptyAndUnknownShape(t *testing.T) {
	for _, body := range []string{`{"foo":"bar"}`, `"teks"`, `{"data":"x"}`} {
		srv := jsonServer(200, body)
		got, err := NewSearchClient(srv.URL, "").Search(context.Background(), "q", 3)
		srv.Close()
		require.NoError(t, err)
		assert.Empty(t, got)
	}
}

func TestSearchSendsQueryAndAuth(t *testing.T) {
	var gotQ, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQ, gotAuth = r.URL.Query().Get("q"), r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	_, err := NewSearchClient(srv.URL+"?x=1", "kunci").Search(context.Background(), "halo dunia", 3)
	require.NoError(t, err)
	assert.Equal(t, "halo dunia", gotQ)
	assert.Equal(t, "Bearer kunci", gotAuth)
}

func TestSearchErrors(t *testing.T) {
	srv := jsonServer(500, `{}`)
	_, err := NewSearchClient(srv.URL, "").Search(context.Background(), "q", 3)
	srv.Close()
	assert.Error(t, err)

	srv = jsonServer(200, `{rusak`)
	_, err = NewSearchClient(srv.URL, "").Search(context.Background(), "q", 3)
	srv.Close()
	assert.Error(t, err)

	_, err = NewSearchClient("http://[::1", "").Search(context.Background(), "q", 3) // URL tidak valid
	assert.Error(t, err)

	var nilCtx context.Context
	_, err = NewSearchClient("http://localhost", "").Search(nilCtx, "q", 3) //nolint:staticcheck
	assert.Error(t, err)

	srv = jsonServer(200, `[]`)
	url := srv.URL
	srv.Close() // koneksi ditolak
	_, err = NewSearchClient(url, "").Search(context.Background(), "q", 3)
	assert.Error(t, err)
}

func TestClip(t *testing.T) {
	assert.Equal(t, "abc", clip("  abc  ", 10))
	assert.Equal(t, "ab", clip("abcdef", 2))
	assert.Equal(t, "", str(map[string]any{"a": 1}, "a", "b"))
}

// ---------- ChatClient ----------

func TestChatComplete(t *testing.T) {
	var auth, path string
	var payload chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&payload)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"halo"}}]}`))
	}))
	defer srv.Close()

	out, err := NewChatClient(srv.URL+"/", "kunci", "model-x").Complete(context.Background(), "sys", "usr")
	require.NoError(t, err)
	assert.Equal(t, "halo", out)
	assert.Equal(t, "Bearer kunci", auth)
	assert.Equal(t, "/chat/completions", path)
	assert.Equal(t, "model-x", payload.Model)
	assert.Equal(t, maxTokens, payload.MaxTokens)
	assert.Len(t, payload.Messages, 2)
}

func TestChatErrors(t *testing.T) {
	for _, c := range []struct {
		status int
		body   string
	}{
		{500, `{}`},
		{200, `{rusak`},
		{200, `{"choices":[]}`},
		{200, `{"choices":[{"message":{"content":"   "}}]}`},
	} {
		srv := jsonServer(c.status, c.body)
		_, err := NewChatClient(srv.URL, "k", "m").Complete(context.Background(), "s", "u")
		srv.Close()
		assert.Error(t, err, c.body)
	}

	var nilCtx context.Context
	_, err := NewChatClient("http://localhost", "k", "m").Complete(nilCtx, "s", "u") //nolint:staticcheck
	assert.Error(t, err)

	srv := jsonServer(200, `{}`)
	url := srv.URL
	srv.Close()
	_, err = NewChatClient(url, "k", "m").Complete(context.Background(), "s", "u")
	assert.Error(t, err)
}

// ---------- Service ----------

type fakeSearch struct {
	src   []Source
	err   error
	calls int
}

func (f *fakeSearch) Search(_ context.Context, _ string, _ int) ([]Source, error) {
	f.calls++
	return f.src, f.err
}

type fakeChat struct {
	out    string
	err    error
	prompt string
}

func (f *fakeChat) Complete(_ context.Context, _, user string) (string, error) {
	f.prompt = user
	return f.out, f.err
}

func TestGenerateJSONWithSources(t *testing.T) {
	s := &fakeSearch{src: []Source{{Title: "Judul", URL: "http://a", Snippet: "isi"}, {URL: "http://b"}}}
	c := &fakeChat{out: "```json\n" + `{"title":"T","category":"teknologi","excerpt":"E","content":"# C\n\nIsi"}` + "\n```"}
	res, err := NewService(s, c).Generate(context.Background(), GenerateRequest{Topic: "  Golang  ", Category: "Bisnis"})
	require.NoError(t, err)
	assert.Equal(t, "T", res.Title)
	assert.Equal(t, "Teknologi", res.Category)
	assert.Equal(t, "E", res.Excerpt)
	assert.Contains(t, res.Content, "## Sumber Rujukan")
	assert.Contains(t, res.Content, "[Judul](http://a)")
	assert.Contains(t, res.Content, "[http://b](http://b)")
	assert.Len(t, res.Sources, 2)
	assert.Contains(t, c.prompt, "Topik: Golang")
	assert.Contains(t, c.prompt, "Kategori yang diinginkan: Bisnis")
	assert.Contains(t, c.prompt, "[1] Judul")
}

func TestGenerateFallbacks(t *testing.T) {
	// bukan JSON -> seluruh teks jadi konten, judul = topik, kategori Umum, excerpt turunan
	c := &fakeChat{out: "# Judul Liar\n\nIsi bebas tanpa JSON."}
	res, err := NewService(nil, c).Generate(context.Background(), GenerateRequest{Topic: "Topik X"})
	require.NoError(t, err)
	assert.Equal(t, "Topik X", res.Title)
	assert.Equal(t, "Umum", res.Category)
	assert.NotEmpty(t, res.Excerpt)
	assert.Equal(t, "# Judul Liar\n\nIsi bebas tanpa JSON.", res.Content)
	assert.NotNil(t, res.Sources)
	assert.NotContains(t, c.prompt, "Kategori")

	// kategori AI tidak valid -> pakai kategori permintaan
	c = &fakeChat{out: `{"title":"","category":"ngawur","excerpt":"","content":"Konten JSON"}`}
	res, err = NewService(nil, c).Generate(context.Background(), GenerateRequest{Topic: "T", Category: "Opini"})
	require.NoError(t, err)
	assert.Equal(t, "Opini", res.Category)
	assert.Equal(t, "T", res.Title)
	assert.Equal(t, "Konten JSON", res.Excerpt)
}

func TestGenerateSearchControl(t *testing.T) {
	c := &fakeChat{out: `{"title":"T","category":"Umum","excerpt":"E","content":"C"}`}
	off := false
	s := &fakeSearch{src: []Source{{Title: "x", URL: "http://x"}}}
	_, err := NewService(s, c).Generate(context.Background(), GenerateRequest{Topic: "T", UseSearch: &off})
	require.NoError(t, err)
	assert.Equal(t, 0, s.calls)

	s = &fakeSearch{err: errors.New("down")} // kegagalan search diabaikan
	res, err := NewService(s, c).Generate(context.Background(), GenerateRequest{Topic: "T"})
	require.NoError(t, err)
	assert.Equal(t, 1, s.calls)
	assert.Empty(t, res.Sources)
	assert.Equal(t, "C", res.Content)
}

func TestGenerateOpenAIError(t *testing.T) {
	_, err := NewService(nil, &fakeChat{err: errors.New("status 401")}).Generate(context.Background(), GenerateRequest{Topic: "T"})
	assert.ErrorIs(t, err, ErrOpenAI)
	assert.Contains(t, strings.ToLower(err.Error()), "openai")
}

func TestClipRunes(t *testing.T) {
	assert.Equal(t, "ab", clipRunes("abcd", 2))
	assert.Equal(t, "abcd", clipRunes(" abcd ", 10))
}

// ---------- Handler ----------

func post(h *Handler, body string) (*httptest.ResponseRecorder, map[string]any) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/gen", h.Generate)
	req := httptest.NewRequest("POST", "/gen", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var m map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return w, m
}

func TestHandler(t *testing.T) {
	ok := NewHandler(NewService(nil, &fakeChat{out: `{"title":"T","category":"Umum","excerpt":"E","content":"Isi"}`}))
	w, m := post(ok, `{"topic":"Golang"}`)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "success", m["status"])

	w, m = post(ok, `{"category":"Teknologi"}`)
	assert.Equal(t, 400, w.Code)
	assert.Equal(t, "fail", m["status"])
	assert.Contains(t, strings.ToLower(m["message"].(string)), "topic")

	bad := NewHandler(NewService(nil, &fakeChat{err: errors.New("status 401")}))
	w, m = post(bad, `{"topic":"Golang"}`)
	assert.Equal(t, 500, w.Code)
	assert.Equal(t, "error", m["status"])
	assert.Contains(t, strings.ToLower(m["message"].(string)), "openai")
}
