Output "docs/vhs/poml-go-builder.gif"
Set Shell zsh
Set FontFamily "Menlo"
Set FontSize 18
Set Width 1400
Set Height 900
Set Theme Laser
Set PlaybackSpeed 1.05
Set TypingSpeed 32ms
Set WindowBar Colorful
Set BorderRadius 18
Set Margin 32
Set Padding 28
Set CursorBlink false

Type "cd /Users/jade/the/poml-go-sdk"
Enter
Type "clear"
Enter
Type 'echo "Building POML in Go with the fluent Builder"'
Enter
Sleep 400ms
Type "cat > builder_demo.go <<'EOF'"
Enter
Type "package main"
Enter
Type ""
Enter
Type "import ("
Enter
Type '  "fmt"'
Enter
Type '  "log"'
Enter
Type ""
Enter
Type '  "github.com/atlas-foundry/poml-go-sdk/poml"'
Enter
Type ")"
Enter
Type ""
Enter
Type "func main() {"
Enter
Type "  builder := poml.NewBuilder()"
Enter
Type '  builder.Meta("demo.tape", "1.0.0", "poml-go-sdk")'
Enter
Type '  builder.Role("Concise overview of POML rendered live.")'
Enter
Type '  builder.System("Narrate the scene in three upbeat bullets.")'
Enter
Type '  builder.Human("Show why structured prompts are maintainable.")'
Enter
Type ""
Enter
Type "  doc := builder.Build()"
Enter
Type ""
Enter
Type "  if err := poml.Validate(doc); err != nil {"
Enter
Type '    log.Fatalf("validate failed: %v", err)'
Enter
Type "  }"
Enter
Type ""
Enter
Type "  rendered, err := poml.Render(doc)"
Enter
Type "  if err != nil {"
Enter
Type '    log.Fatalf("render failed: %v", err)'
Enter
Type "  }"
Enter
Type ""
Enter
Type '  fmt.Println("Rendered POML:\n" + string(rendered))'
Enter
Type ""
Enter
Type "  out, err := poml.Convert(doc, poml.FormatOpenAIChat, poml.ConvertOptions{})"
Enter
Type "  if err != nil {"
Enter
Type '    log.Fatalf("convert failed: %v", err)'
Enter
Type "  }"
Enter
Type ""
Enter
Type '  fmt.Println("\nOpenAI chat JSON:\n" + string(out))'
Enter
Type "}"
Enter
Type "EOF"
Enter
Sleep 500ms
Type "go run builder_demo.go"
Enter
Sleep 1800ms
Type "rm builder_demo.go"
Enter
