package poml

import "testing"

func TestLangChainAggregatesToolCalls(t *testing.T) {
	src := `<poml>
  <tool-request id="c1" name="calc" parameters="{{ { x: 1 } }}"/>
  <tool-request id="c2" name="search" parameters="{{ { q: 'hi' } }}"/>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatLangChain, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := outAny.(map[string]any)
	msgs := out["messages"].([]map[string]any)
	if len(msgs) != 1 {
		t.Fatalf("expected single ai message aggregating tool_calls, got %d", len(msgs))
	}
	data := msgs[0]["data"].(map[string]any)
	calls, ok := data["tool_calls"].([]any)
	if !ok || len(calls) != 2 {
		t.Fatalf("expected 2 tool_calls, got %+v", data["tool_calls"])
	}
}
