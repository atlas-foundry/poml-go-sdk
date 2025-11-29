package poml

import (
	"regexp"
	"strings"
)

var tagLikeRe = regexp.MustCompile(`</?[A-Za-z][\w\-\:]*[^>]*>`)

// liftEmbeddedTags extracts XML-like tags when they appear at top level inside mixed text.
// It preserves original whitespace and comments by splitting around tag-like patterns.
func liftEmbeddedTags(input string) string {
	if !strings.Contains(input, "<") || !strings.Contains(input, ">") {
		return input
	}
	var out []string
	last := 0
	matches := tagLikeRe.FindAllStringIndex(input, -1)
	for _, m := range matches {
		start, end := m[0], m[1]
		if start > last {
			out = append(out, input[last:start])
		}
		out = append(out, input[start:end])
		last = end
	}
	if last < len(input) {
		out = append(out, input[last:])
	}
	return strings.Join(out, "")
}
