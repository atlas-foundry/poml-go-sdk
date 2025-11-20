package poml

import "errors"

// Format enumerates output conversion targets inspired by the Python SDK.
type Format string

const (
	FormatMessageDict Format = "message_dict"
	FormatDict        Format = "dict"
	FormatOpenAIChat  Format = "openai_chat"
	FormatLangChain   Format = "langchain"
	FormatPydantic    Format = "pydantic"
)

// ConvertOptions holds knobs for conversion (context, runtime flags, etc.).
// This will expand when format support is implemented.
type ConvertOptions struct {
	// Context may include auxiliary data (e.g., file paths) needed for conversions.
	Context map[string]any
}

// ErrNotImplemented signals that a conversion target is not yet supported.
var ErrNotImplemented = errors.New("conversion not implemented")

// Convert transforms a parsed Document into the requested format.
// TODO: implement parity with python/poml conversion outputs.
func Convert(doc Document, format Format, opts ConvertOptions) (any, error) {
	return nil, ErrNotImplemented
}

// ConvertString parses a POML string and converts it in one step.
func ConvertString(body string, format Format, opts ConvertOptions) (any, error) {
	doc, err := ParseString(body)
	if err != nil {
		return nil, err
	}
	return Convert(doc, format, opts)
}
