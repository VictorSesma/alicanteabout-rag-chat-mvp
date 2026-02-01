package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"content-rag-chat/internal/rag"
)

type chatCompletionRequest struct {
	Model          string              `json:"model"`
	Messages       []chatMessage       `json:"messages"`
	Temperature    float32             `json:"temperature,omitempty"`
	ResponseFormat *chatResponseFormat `json:"response_format,omitempty"`
	Stream         bool                `json:"stream,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponseFormat struct {
	Type string `json:"type"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (s *Server) callChatCompletion(ctx context.Context, req chatCompletionRequest) (string, error) {
	body, _ := json.Marshal(req)
	reqID := rag.RequestID(ctx)
	if reqID == "" {
		reqID = "unknown"
	}
	log.Printf("req_id=%s openai chat request_bytes=%d model=%s messages=%d", reqID, len(body), req.Model, len(req.Messages))
	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(res.Body, 10*1024*1024))
	log.Printf("req_id=%s openai chat response_status=%d response_bytes=%d took=%s", reqID, res.StatusCode, len(raw), fmtDuration(time.Since(start)))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("openai chat http %d: %s", res.StatusCode, string(raw))
	}

	var out chatCompletionResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("parse chat response: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("openai error: %s (%s)", out.Error.Message, out.Error.Type)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai: empty choices")
	}
	return out.Choices[0].Message.Content, nil
}

type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func (s *Server) callChatCompletionStream(ctx context.Context, req chatCompletionRequest, onDelta func(string)) error {
	body, _ := json.Marshal(req)
	reqID := rag.RequestID(ctx)
	if reqID == "" {
		reqID = "unknown"
	}
	log.Printf("req_id=%s openai chat stream request_bytes=%d model=%s messages=%d", reqID, len(body), req.Model, len(req.Messages))
	start := time.Now()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := s.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 10*1024*1024))
		return fmt.Errorf("openai chat http %d: %s", res.StatusCode, string(raw))
	}

	sc := bufio.NewScanner(res.Body)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 10*1024*1024)
	var bytesRead int
	for sc.Scan() {
		line := sc.Text()
		bytesRead += len(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk chatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta != "" && onDelta != nil {
			onDelta(delta)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	log.Printf("req_id=%s openai chat stream bytes=%d took=%s", reqID, bytesRead, fmtDuration(time.Since(start)))
	return nil
}
