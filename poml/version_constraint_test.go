package poml

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"1.0.0", Version{1, 0, 0, ""}, false},
		{"1.2.3", Version{1, 2, 3, ""}, false},
		{"0.0.1", Version{0, 0, 1, ""}, false},
		{"2.0", Version{2, 0, 0, ""}, false},
		{"3", Version{3, 0, 0, ""}, false},
		{"1.0.0-alpha", Version{1, 0, 0, "alpha"}, false},
		{"1.0.0-beta.1", Version{1, 0, 0, "beta.1"}, false},
		{"", Version{}, true},
		{"invalid", Version{}, true},
		{"1.2.3.4", Version{}, true},
		{"a.b.c", Version{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0", "1.0.0-alpha", 1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-beta", "1.0.0-alpha", 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			a, _ := ParseVersion(tt.a)
			b, _ := ParseVersion(tt.b)
			got := a.Compare(b)
			if got != tt.want {
				t.Errorf("Version(%s).Compare(%s) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		v    Version
		want string
	}{
		{Version{1, 0, 0, ""}, "1.0.0"},
		{Version{1, 2, 3, ""}, "1.2.3"},
		{Version{1, 0, 0, "alpha"}, "1.0.0-alpha"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.v.String(); got != tt.want {
				t.Errorf("Version.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckVersionConstraint(t *testing.T) {
	tests := []struct {
		current, min, max string
		wantErr           bool
	}{
		{"1.0.0", "", "", false},
		{"1.0.0", "1.0.0", "", false},
		{"1.0.0", "", "1.0.0", false},
		{"1.0.0", "1.0.0", "1.0.0", false},
		{"1.5.0", "1.0.0", "2.0.0", false},
		{"0.9.0", "1.0.0", "", true},
		{"2.1.0", "", "2.0.0", true},
		{"0.5.0", "1.0.0", "2.0.0", true},
		{"3.0.0", "1.0.0", "2.0.0", true},
		{"invalid", "1.0.0", "", true},
		{"1.0.0", "invalid", "", true},
		{"1.0.0", "", "invalid", true},
	}

	for _, tt := range tests {
		name := tt.current + "_" + tt.min + "_" + tt.max
		t.Run(name, func(t *testing.T) {
			err := CheckVersionConstraint(tt.current, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckVersionConstraint(%q, %q, %q) error = %v, wantErr %v",
					tt.current, tt.min, tt.max, err, tt.wantErr)
			}
		})
	}
}

func TestSpecVersion(t *testing.T) {
	_, err := ParseVersion(SpecVersion)
	if err != nil {
		t.Errorf("SpecVersion %q is not a valid version: %v", SpecVersion, err)
	}
}

func TestSDKVersion(t *testing.T) {
	_, err := ParseVersion(SDKVersion)
	if err != nil {
		t.Errorf("SDKVersion %q is not a valid version: %v", SDKVersion, err)
	}
}
