package services

import (
	"strings"
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

type TokenCounter struct {
	encoderCache map[string]*tiktoken.Tiktoken
	mu           sync.RWMutex
}

func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		encoderCache: make(map[string]*tiktoken.Tiktoken),
	}
}

func (tc *TokenCounter) CountTokens(text string, model string) (uint32, error) {
	encoder, err := tc.getEncoder(model)
	if err != nil {
		return tc.fallbackCount(text), nil
	}

	tokens := encoder.Encode(text, nil, nil)
	return uint32(len(tokens)), nil
}

func (tc *TokenCounter) getEncoder(model string) (*tiktoken.Tiktoken, error) {
	encodingName := tc.getEncodingForModel(model)

	tc.mu.RLock()
	if enc, ok := tc.encoderCache[encodingName]; ok {
		tc.mu.RUnlock()
		return enc, nil
	}
	tc.mu.RUnlock()

	tc.mu.Lock()
	defer tc.mu.Unlock()

	if enc, ok := tc.encoderCache[encodingName]; ok {
		return enc, nil
	}

	enc, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return nil, err
	}

	tc.encoderCache[encodingName] = enc
	return enc, nil
}

func (tc *TokenCounter) getEncodingForModel(model string) string {
	model = strings.ToLower(model)

	switch {
	case strings.HasPrefix(model, "gpt-4o"),
		strings.HasPrefix(model, "gpt-4-turbo"),
		strings.HasPrefix(model, "gpt-4-1106"),
		strings.HasPrefix(model, "gpt-4-0125"),
		strings.HasPrefix(model, "gpt-3.5-turbo-1106"),
		strings.HasPrefix(model, "gpt-3.5-turbo-0125"):
		return "cl100k_base"
	case strings.HasPrefix(model, "gpt-4"),
		strings.HasPrefix(model, "gpt-3.5-turbo"):
		return "cl100k_base"
	case strings.HasPrefix(model, "text-embedding-3"),
		strings.HasPrefix(model, "text-embedding-ada"):
		return "cl100k_base"
	case strings.HasPrefix(model, "claude"),
		strings.HasPrefix(model, "gemini"),
		strings.HasPrefix(model, "mistral"),
		strings.HasPrefix(model, "mixtral"),
		strings.HasPrefix(model, "llama"),
		strings.HasPrefix(model, "command"):
		return "cl100k_base"
	default:
		return "cl100k_base"
	}
}

func (tc *TokenCounter) fallbackCount(text string) uint32 {
	words := strings.Fields(text)
	return uint32(float64(len(words)) * 1.3)
}
