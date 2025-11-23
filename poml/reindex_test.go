package poml

import "testing"

func TestDocumentReindex(t *testing.T) {
	doc := Document{
		Elements: []Element{
			{Type: ElementTask},
			{Type: ElementTask},
			{Type: ElementToolDefinition},
			{Type: ElementToolRequest},
			{Type: ElementToolResponse},
			{Type: ElementToolResult},
			{Type: ElementToolError},
			{Type: ElementRuntime},
			{Type: ElementAudio},
			{Type: ElementVideo},
			{Type: ElementObject},
			{Type: ElementImage},
			{Type: ElementDiagram},
			{Type: ElementInput},
			{Type: ElementDocument},
			{Type: ElementStyle},
			{Type: ElementHint},
			{Type: ElementExample},
			{Type: ElementContentPart},
			{Type: ElementOutputFormat},
			{Type: ElementHumanMsg},
			{Type: ElementAssistantMsg},
			{Type: ElementSystemMsg},
		},
	}

	doc.reindex()

	assertIdx := func(idx int, typ ElementType, expect int) {
		if got := doc.Elements[idx].Index; got != expect {
			t.Fatalf("type %s index = %d, want %d", typ, got, expect)
		}
	}

	assertIdx(0, ElementTask, 0)
	assertIdx(1, ElementTask, 1)
	assertIdx(2, ElementToolDefinition, 0)
	assertIdx(3, ElementToolRequest, 0)
	assertIdx(4, ElementToolResponse, 0)
	assertIdx(5, ElementToolResult, 0)
	assertIdx(6, ElementToolError, 0)
	assertIdx(7, ElementRuntime, 0)
	assertIdx(8, ElementAudio, 0)
	assertIdx(9, ElementVideo, 0)
	assertIdx(10, ElementObject, 0)
	assertIdx(11, ElementImage, 0)
	assertIdx(12, ElementDiagram, 0)
	assertIdx(13, ElementInput, 0)
	assertIdx(14, ElementDocument, 0)
	assertIdx(15, ElementStyle, 0)
	assertIdx(16, ElementHint, 0)
	assertIdx(17, ElementExample, 0)
	assertIdx(18, ElementContentPart, 0)
	assertIdx(19, ElementOutputFormat, 0)
	assertIdx(20, ElementHumanMsg, 0)
	assertIdx(21, ElementAssistantMsg, 1) // message indices share counter
	assertIdx(22, ElementSystemMsg, 2)
}
