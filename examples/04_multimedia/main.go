// Example: Multimedia Handling
//
// This example demonstrates how POML handles:
// 1. Images (file paths, URLs, data URIs)
// 2. Audio content
// 3. Video content
// 4. Base64 encoding for API payloads
//
// Run: go run main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

func main() {
	// POML with various multimedia elements
	input := `<poml>
  <meta>
    <id>vision-assistant</id>
    <version>1.0.0</version>
    <owner>examples</owner>
  </meta>

  <role>You are a vision-capable AI assistant that can analyze images and multimedia.</role>

  <task>Analyze images and provide detailed descriptions.</task>
  <task>Answer questions about visual content.</task>

  <!-- Image from URL -->
  <human-msg>
    What's in this image?
    <image src="https://example.com/photo.jpg" alt="A sample photo"/>
  </human-msg>

  <assistant-msg>I can see a beautiful landscape photo showing mountains and a lake.</assistant-msg>

  <!-- Image with data URI (base64 encoded) -->
  <human-msg>
    Compare it with this one:
    <image src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==" alt="Tiny test image"/>
  </human-msg>

  <!-- Audio element -->
  <human-msg>
    Can you transcribe this audio?
    <audio src="https://example.com/speech.mp3" alt="Speech recording"/>
  </human-msg>

  <!-- Video element -->
  <human-msg>
    What's happening in this video?
    <video src="https://example.com/clip.mp4" alt="Video clip"/>
  </human-msg>
</poml>`

	doc, err := poml.ParseString(input)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	if err := doc.Validate(); err != nil {
		log.Fatalf("Validation error: %v", err)
	}

	fmt.Println("=== Document Elements ===")
	for i, elem := range doc.Elements {
		fmt.Printf("%d. Type: %s\n", i+1, elem.Type)
	}

	// Show images separately
	fmt.Println("\n=== Images ===")
	for _, img := range doc.Images {
		fmt.Printf("  Src: %s\n", truncate(img.Src, 60))
		fmt.Printf("  Alt: %s\n", img.Alt)
	}

	// Show audio/video
	fmt.Println("\n=== Audio ===")
	for _, audio := range doc.Audios {
		fmt.Printf("  Src: %s\n", truncate(audio.Src, 60))
	}

	fmt.Println("\n=== Video ===")
	for _, video := range doc.Videos {
		fmt.Printf("  Src: %s\n", truncate(video.Src, 60))
	}

	// OpenAI format handles images specially
	fmt.Println("\n=== OpenAI Chat Format ===")
	fmt.Println("(Images converted to content array with image_url objects)")
	openaiOutput, err := poml.Convert(doc, poml.FormatOpenAIChat, poml.ConvertOptions{
		BaseDir: ".", // For resolving relative paths
	})
	if err != nil {
		log.Fatalf("Convert error: %v", err)
	}

	jsonBytes, _ := json.MarshalIndent(openaiOutput, "", "  ")
	fmt.Println(string(jsonBytes))

	// Pydantic format collects media separately
	fmt.Println("\n=== Pydantic Format ===")
	fmt.Println("(Media collected in separate 'media' array)")
	pydanticOutput, err := poml.Convert(doc, poml.FormatPydantic, poml.ConvertOptions{})
	if err != nil {
		log.Fatalf("Convert error: %v", err)
	}

	// Show just the media array
	if pMap, ok := pydanticOutput.(map[string]any); ok {
		if media, ok := pMap["media"]; ok && media != nil {
			fmt.Println("Media array:")
			mediaJSON, _ := json.MarshalIndent(media, "", "  ")
			fmt.Println(string(mediaJSON))
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
