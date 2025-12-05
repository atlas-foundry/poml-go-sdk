// Package token provides token counting and limit enforcement for POML documents.
package token

import (
	"strings"
	"unicode"
)

// Counter provides token counting functionality.
type Counter interface {
	Count(text string) (int64, error)
}

// DefaultCounter uses a simple approximation (chars / 4).
type DefaultCounter struct{}

// Count estimates tokens using the chars/4 approximation.
func (c DefaultCounter) Count(text string) (int64, error) {
	// Simple approximation: ~4 characters per token
	// This is a rough estimate for English text with GPT-style tokenization
	charCount := int64(len(text))
	return (charCount + 3) / 4, nil // Round up
}

// CharCounter counts characters.
type CharCounter struct{}

// Count returns the exact character count.
func (c CharCounter) Count(text string) (int64, error) {
	return int64(len(text)), nil
}

// WordCounter counts words (whitespace-separated tokens).
type WordCounter struct{}

// Count returns the word count.
func (c WordCounter) Count(text string) (int64, error) {
	words := strings.Fields(text)
	return int64(len(words)), nil
}

// EstimateTokens provides a rough token estimate for text.
// Uses the common approximation of ~4 characters per token.
func EstimateTokens(text string) int64 {
	counter := DefaultCounter{}
	count, _ := counter.Count(text)
	return count
}

// CountChars returns the exact character count.
func CountChars(text string) int64 {
	return int64(len(text))
}

// CountWords returns the word count.
func CountWords(text string) int64 {
	return int64(len(strings.Fields(text)))
}

// CountRunes returns the Unicode rune (code point) count.
func CountRunes(text string) int64 {
	count := int64(0)
	for range text {
		count++
	}
	return count
}

// TruncateToCharLimit truncates text to fit within a character limit.
// Tries to break at word boundaries when possible.
func TruncateToCharLimit(text string, limit int64) string {
	if int64(len(text)) <= limit {
		return text
	}

	// Find last space before limit
	lastSpace := -1
	for i := 0; i < int(limit) && i < len(text); i++ {
		if unicode.IsSpace(rune(text[i])) {
			lastSpace = i
		}
	}

	if lastSpace > 0 && lastSpace > int(limit)/2 {
		return strings.TrimSpace(text[:lastSpace]) + "..."
	}

	// If limit is too small for ellipsis, just truncate
	if limit <= 3 {
		return text[:limit]
	}
	return text[:limit-3] + "..."
}

// TruncateToTokenLimit truncates text to fit within a token limit.
func TruncateToTokenLimit(text string, limit int64) string {
	estimated := EstimateTokens(text)
	if estimated <= limit {
		return text
	}

	// Estimate character limit from token limit
	charLimit := limit * 4
	return TruncateToCharLimit(text, charLimit)
}

// PrioritizedContent represents content with a priority for limit enforcement.
type PrioritizedContent struct {
	Content  string
	Priority int // Higher priority = keep first
	ID       string
}

// EnforceLimitWithPriority keeps high-priority content within limits.
// Returns the filtered content that fits within the limit.
func EnforceLimitWithPriority(contents []PrioritizedContent, charLimit int64) []PrioritizedContent {
	if charLimit <= 0 {
		return contents
	}

	// Sort by priority (descending)
	sorted := make([]PrioritizedContent, len(contents))
	copy(sorted, contents)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Priority > sorted[i].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Take content until limit reached
	var result []PrioritizedContent
	var total int64

	for _, c := range sorted {
		contentLen := int64(len(c.Content))
		if total+contentLen <= charLimit {
			result = append(result, c)
			total += contentLen
		} else if total < charLimit {
			// Partial content
			remaining := charLimit - total
			truncated := TruncateToCharLimit(c.Content, remaining)
			result = append(result, PrioritizedContent{
				Content:  truncated,
				Priority: c.Priority,
				ID:       c.ID,
			})
			break
		}
	}

	return result
}
