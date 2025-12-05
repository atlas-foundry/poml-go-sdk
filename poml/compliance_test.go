package poml

import (
	"path/filepath"
	"testing"
)

// Compliance matrix with explicit allow/deny manifests to avoid false positives.
func TestComplianceMatrix(t *testing.T) {
	allow := map[string]ValidateOptions{
		"207_multimedia.poml":             {Extended: ExtendedStrict},
		"components_formatting.poml":      {Extended: ExtendedOff},
		"conversation_multi.poml":         {Extended: ExtendedStrict},
		"core_full.poml":                  {Extended: ExtendedOff},
		"extended_data_block.poml":        {Extended: ExtendedStrict},
		"extended_formatting_mix.poml":    {Extended: ExtendedStrict},
		"extended_media_oversize.poml":    {Extended: ExtendedStrict},
		"folder_tree.poml":                {Extended: ExtendedStrict},
		"formatting_inline.poml":          {Extended: ExtendedOff},
		"meta_attrs.poml":                 {Extended: ExtendedOff},
		"parity_basic.poml":               {Extended: ExtendedOff},
		"parity_extended_attrs.poml":      {Extended: ExtendedStrict},
		"parity_extended_media.poml":      {Extended: ExtendedStrict},
		"parity_extended_mixed.poml":      {Extended: ExtendedStrict},
		"parity_extended_textescape.poml": {Extended: ExtendedStrict},
		"parity_persona.poml":             {Extended: ExtendedOff},
		"richtext_blocks.poml":            {Extended: ExtendedStrict},
		"richtext_code.poml":              {Extended: ExtendedStrict},
		"richtext_inline.poml":            {Extended: ExtendedStrict},
		"richtext_lists.poml":             {Extended: ExtendedStrict},
		"stylesheet_basic.poml":           {Extended: ExtendedStrict},
		"stylesheet_classes.poml":         {Extended: ExtendedStrict},
		"table_csv.poml":                  {Extended: ExtendedStrict},
		"table_inline.poml":               {Extended: ExtendedStrict},
		"template_conditional.poml":       {Extended: ExtendedStrict},
		"template_include.poml":           {Extended: ExtendedStrict},
		"template_let.poml":               {Extended: ExtendedStrict},
		"template_loop.poml":              {Extended: ExtendedStrict},
		"template_variables.poml":         {Extended: ExtendedStrict},
		"token_charlimit.poml":            {Extended: ExtendedStrict},
		"token_priority.poml":             {Extended: ExtendedStrict},
		"ts_reference_basic.poml":         {Extended: ExtendedOff},
		"version_valid.poml":              {Extended: ExtendedStrict},
		"webpage_extract.poml":            {Extended: ExtendedStrict},
	}

	deny := map[string]ValidateOptions{
		"101_explain_character.poml":             {Extended: ExtendedOff},
		"102_render_xml.poml":                    {Extended: ExtendedOff},
		"103_word_todos.poml":                    {Extended: ExtendedOff},
		"104_financial_analysis.poml":            {Extended: ExtendedOff},
		"105_write_blog_post.poml":               {Extended: ExtendedOff},
		"106_research.poml":                      {Extended: ExtendedOff},
		"107_read_report_pdf.poml":               {Extended: ExtendedOff},
		"108_math_calculator.poml":               {Extended: ExtendedOff},
		"109_math_verifier.poml":                 {Extended: ExtendedOff},
		"110_code_review.poml":                   {Extended: ExtendedOff},
		"201_orders_qa.poml":                     {Extended: ExtendedOff},
		"202_arc_agi.poml":                       {Extended: ExtendedOff},
		"203_expense_extract_document.poml":      {Extended: ExtendedOff},
		"204_expense_extract_rules.poml":         {Extended: ExtendedOff},
		"205_expense_check_compliance.poml":      {Extended: ExtendedOff},
		"206_expense_send_email.poml":            {Extended: ExtendedOff},
		"301_generate_poml.poml":                 {Extended: ExtendedOff},
		"extended_data_invalid_syntax.poml":      {Extended: ExtendedStrict},
		"extended_data_missing_syntax.poml":      {Extended: ExtendedStrict},
		"extended_data_unknown_attr.poml":        {Extended: ExtendedStrict},
		"extended_media_invalid_audio_mime.poml": {Extended: ExtendedStrict},
		"extended_media_invalid_mime.poml":       {Extended: ExtendedStrict},
		"extended_media_invalid_size.poml":       {Extended: ExtendedStrict},
		"extended_media_missing_syntax.poml":     {Extended: ExtendedStrict},
		"extended_object_invalid_syntax.poml":    {Extended: ExtendedStrict},
		"extended_op_invalid_kind.poml":          {Extended: ExtendedStrict},
		"extended_off_op.poml":                   {Extended: ExtendedOff},
		"extended_off_figure.poml":               {Extended: ExtendedOff},
		"extended_off_object.poml":               {Extended: ExtendedOff},
		"extended_off_data.poml":                 {Extended: ExtendedOff},
		"extended_off_text.poml":                 {Extended: ExtendedOff},
		"validation_config.poml":                 {Extended: ExtendedStrict},
		"version_invalid_max.poml":               {Extended: ExtendedStrict, EnforceVersions: true},
		"version_invalid_min.poml":               {Extended: ExtendedStrict, EnforceVersions: true},
	}

	examples, err := filepath.Glob(filepath.Join("testdata", "examples", "*.poml"))
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}

	for _, path := range examples {
		base := filepath.Base(path)
		if opts, ok := allow[base]; ok {
			doc, err := ParseFile(path)
			if err != nil {
				t.Fatalf("parse %s: %v", base, err)
			}
			if err := doc.ValidateWithOptions(opts); err != nil {
				t.Fatalf("validate %s (%v): %v", base, opts.Extended, err)
			}
			if _, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: opts.Extended}); err != nil {
				t.Fatalf("convert %s (%v): %v", base, opts.Extended, err)
			}
			continue
		}

		if opts, ok := deny[base]; ok {
			doc, err := ParseFile(path)
			if err != nil {
				t.Fatalf("parse %s: %v", base, err)
			}
			if err := doc.ValidateWithOptions(opts); err == nil {
				t.Fatalf("expected validation failure for %s", base)
			}
			continue
		}

		t.Fatalf("fixture not classified in compliance manifest: %s", base)
	}
}
