package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Completer mengirim prompt ke LLM dan mengembalikan teks jawaban.
type Completer interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

// ChatClient memanggil {OPENAI_ENDPOINT}/chat/completions.
type ChatClient struct {
	endpoint string
	key      string
	model    string
	http     *http.Client
}

// NewChatClient membuat client OpenAI Chat Completions (timeout 120 detik).
func NewChatClient(endpoint, key, model string) *ChatClient {
	return &ChatClient{
		endpoint: strings.TrimRight(endpoint, "/"), key: key, model: model,
		http: &http.Client{Timeout: 120 * time.Second},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// maxTokens dibatasi agar kredit LLM hemat.
const maxTokens = 1400

// Complete mengirim satu putaran chat.
func (c *ChatClient) Complete(ctx context.Context, system, user string) (string, error) {
	payload, _ := json.Marshal(chatRequest{
		Model: c.model, Temperature: 0.6, MaxTokens: maxTokens,
		Messages: []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	var out chatResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return "", errors.New("respons kosong")
	}
	return out.Choices[0].Message.Content, nil
}
