package poml

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version (major.minor.patch).
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

// ParseVersion parses a semantic version string (e.g., "1.2.3", "1.0.0-beta").
func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Version{}, fmt.Errorf("empty version string")
	}

	var v Version
	var core string

	// Split prerelease suffix
	if idx := strings.Index(s, "-"); idx >= 0 {
		core = s[:idx]
		v.Prerelease = s[idx+1:]
	} else {
		core = s
	}

	parts := strings.Split(core, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return Version{}, fmt.Errorf("invalid version format: %s", s)
	}

	var err error
	v.Major, err = strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	if len(parts) >= 2 {
		v.Minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return Version{}, fmt.Errorf("invalid minor version: %s", parts[1])
		}
	}

	if len(parts) >= 3 {
		v.Patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return Version{}, fmt.Errorf("invalid patch version: %s", parts[2])
		}
	}

	return v, nil
}

// String returns the version as a string.
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	return s
}

// Compare returns -1 if v < other, 0 if v == other, 1 if v > other.
// Prerelease versions are considered less than release versions.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	// Handle prerelease: no prerelease > prerelease
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	if v.Prerelease < other.Prerelease {
		return -1
	}
	if v.Prerelease > other.Prerelease {
		return 1
	}

	return 0
}

// CheckVersionConstraint validates that current satisfies min <= current <= max.
// Empty min or max strings are treated as unbounded.
func CheckVersionConstraint(current, min, max string) error {
	if min == "" && max == "" {
		return nil
	}

	cur, err := ParseVersion(current)
	if err != nil {
		return fmt.Errorf("invalid current version: %w", err)
	}

	if min != "" {
		minV, err := ParseVersion(min)
		if err != nil {
			return fmt.Errorf("invalid minVersion: %w", err)
		}
		if cur.Compare(minV) < 0 {
			return fmt.Errorf("version %s is below minimum %s", current, min)
		}
	}

	if max != "" {
		maxV, err := ParseVersion(max)
		if err != nil {
			return fmt.Errorf("invalid maxVersion: %w", err)
		}
		if cur.Compare(maxV) > 0 {
			return fmt.Errorf("version %s exceeds maximum %s", current, max)
		}
	}

	return nil
}

// SpecVersion is the Microsoft POML specification version this SDK implements.
// This is used for minVersion/maxVersion constraint checking in POML documents.
// See: https://github.com/microsoft/poml
const SpecVersion = "0.0.8"

// SDKVersion is the current version of the POML Go SDK (for informational purposes).
// Note: Version constraints in POML documents check against SpecVersion, not SDKVersion.
const SDKVersion = "0.2.0"
