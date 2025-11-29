package poml

import "testing"

func TestExtendedStrictAcceptsDefaultAllowlistMIMEs(t *testing.T) {
	body := `<poml mode="extended">
	<meta><id>allow</id><version>1</version><owner>sdk</owner></meta>
	<role>r</role><task>t</task>
	<figure src="data:image/tiff;base64,Zw==" alt="pic" syntax="image/tiff"/>
	<figure src="data:video/quicktime;base64,Zw==" alt="clip" syntax="video/quicktime"/>
	<object syntax="audio/flac" data="data:audio/flac;base64,Zw=="></object>
	</poml>`
	doc, err := ParseString(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := doc.ValidateWithOptions(ValidateOptions{Extended: ExtendedStrict}); err != nil {
		t.Fatalf("validate: %v", err)
	}
}
