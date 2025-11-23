Output "docs/vhs/poml-cli.gif"
Set Shell zsh
Set FontFamily "Menlo"
Set FontSize 18
Set Width 1400
Set Height 900
Set Theme TokyoNightStorm
Set PlaybackSpeed 1.1
Set TypingSpeed 35ms
Set WindowBar Colorful
Set BorderRadius 18
Set Margin 32
Set Padding 28
Set CursorBlink false

Type "cd /Users/jade/the/poml-go-sdk"
Enter
Type "clear"
Enter
Type 'echo "POML CLI quickstart: parse, validate, convert"'
Enter
Sleep 400ms
Type "cat > hello.poml <<'EOF'"
Enter
Type "<poml>"
Enter
Type "  <system>You are a crisp, upbeat assistant.</system>"
Enter
Type "  <user>Walk me through poml-go-sdk in one breezy paragraph.</user>"
Enter
Type "</poml>"
Enter
Type "EOF"
Enter
Sleep 400ms
Type "cat hello.poml"
Enter
Sleep 600ms
Type "cat > poml-cli.go <<'EOF'"
Enter
Type "package main"
Enter
Type ""
Enter
Type "import ("
Enter
Type '  "flag"'
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
Type '  file := flag.String("file", "hello.poml", "path to a POML file")'
Enter
Type '  format := flag.String("format", "openai_chat", "message_dict|dict|openai_chat|langchain")'
Enter
Type "  flag.Parse()"
Enter
Type ""
Enter
Type "  doc, err := poml.ParseFile(*file)"
Enter
Type "  if err != nil {"
Enter
Type '    log.Fatalf("parse failed: %v", err)'
Enter
Type "  }"
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
Type "  out, err := poml.Convert(doc, poml.Format(*format), poml.ConvertOptions{})"
Enter
Type "  if err != nil {"
Enter
Type '    log.Fatalf("convert failed: %v", err)'
Enter
Type "  }"
Enter
Type ""
Enter
Type '  fmt.Printf("format=%s\n%s\n", *format, string(out))'
Enter
Type "}"
Enter
Type "EOF"
Enter
Sleep 500ms
Type "go run poml-cli.go --file hello.poml --format openai_chat"
Enter
Sleep 1500ms
Type "go run poml-cli.go --file hello.poml --format dict"
Enter
Sleep 1400ms
Type "rm hello.poml poml-cli.go"
Enter
