package poml

import (
	"os"
	"path/filepath"
	"testing"
)

// Parity fixture sanity: ensure required on/off goldens exist so compliance stays meaningful.
func TestParityFixtureCoverage(t *testing.T) {
	requireExamples := []string{
		filepath.Join("testdata", "examples", "parity_extended_media.message_dict.json"),
		filepath.Join("testdata", "examples", "parity_extended_media.openai_chat.json"),
		filepath.Join("testdata", "examples", "parity_extended_media.langchain.json"),
		filepath.Join("testdata", "examples", "parity_extended_media.off.message_dict.json"),
		filepath.Join("testdata", "examples", "parity_extended_media.off.openai_chat.json"),
		filepath.Join("testdata", "examples", "parity_extended_media.off.langchain.json"),

		filepath.Join("testdata", "examples", "extended_data_block.message_dict.json"),
		filepath.Join("testdata", "examples", "extended_data_block.openai_chat.json"),
		filepath.Join("testdata", "examples", "extended_data_block.langchain.json"),
		filepath.Join("testdata", "examples", "extended_data_block.off.message_dict.json"),
		filepath.Join("testdata", "examples", "extended_data_block.off.openai_chat.json"),
		filepath.Join("testdata", "examples", "extended_data_block.off.langchain.json"),

		filepath.Join("testdata", "examples", "parity_extended_attrs.message_dict.json"),
		filepath.Join("testdata", "examples", "parity_extended_attrs.openai_chat.json"),
		filepath.Join("testdata", "examples", "parity_extended_attrs.langchain.json"),
		filepath.Join("testdata", "examples", "parity_extended_attrs.off.message_dict.json"),
	}

	requireGoldens := []string{
		filepath.Join("testdata", "golden", "extended_media_invalid_mime.txt"),
		filepath.Join("testdata", "golden", "extended_media_invalid_audio_mime.txt"),
		filepath.Join("testdata", "golden", "extended_media_invalid_size.txt"),
		filepath.Join("testdata", "golden", "extended_data_missing_syntax.txt"),
		filepath.Join("testdata", "golden", "extended_data_invalid_syntax.txt"),
		filepath.Join("testdata", "golden", "extended_data_unknown_attr.txt"),
		filepath.Join("testdata", "golden", "extended_object_invalid_syntax.txt"),
		filepath.Join("testdata", "golden", "extended_off_errors.txt"),
		filepath.Join("testdata", "golden", "extended_op_invalid_kind.txt"),
		filepath.Join("testdata", "examples", "components_formatting.poml"),
		filepath.Join("testdata", "examples", "formatting_inline.poml"),
		filepath.Join("testdata", "examples", "extended_formatting_mix.poml"),
	}

	for _, path := range append(requireExamples, requireGoldens...) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required parity fixture missing: %s (%v)", path, err)
		}
	}
}
