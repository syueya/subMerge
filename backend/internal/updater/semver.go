package updater

import (
	"fmt"
	"strings"
)

// Version is a parsed Semantic Versioning 2.0.0 version.
type Version struct {
	Major      string
	Minor      string
	Patch      string
	Prerelease []string
	Build      []string
}

// ParseVersion parses a SemVer 2.0.0 version. A leading v is accepted because
// GitHub release tags commonly use it, but it is not included in String.
func ParseVersion(raw string) (Version, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "v") {
		raw = raw[1:]
	}
	if raw == "" {
		return Version{}, fmt.Errorf("invalid semantic version %q", raw)
	}

	coreAndPre, buildRaw, hasBuild := strings.Cut(raw, "+")
	if hasBuild && (buildRaw == "" || strings.Contains(buildRaw, "+")) {
		return Version{}, fmt.Errorf("invalid semantic version %q", raw)
	}
	core, preRaw, hasPre := strings.Cut(coreAndPre, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid semantic version %q", raw)
	}
	major, err := parseNumericIdentifier(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid semantic version %q: %w", raw, err)
	}
	minor, err := parseNumericIdentifier(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid semantic version %q: %w", raw, err)
	}
	patch, err := parseNumericIdentifier(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid semantic version %q: %w", raw, err)
	}

	v := Version{Major: major, Minor: minor, Patch: patch}
	if hasPre {
		v.Prerelease, err = parseIdentifiers(preRaw, true)
		if err != nil {
			return Version{}, fmt.Errorf("invalid semantic version %q: %w", raw, err)
		}
	}
	if hasBuild {
		v.Build, err = parseIdentifiers(buildRaw, false)
		if err != nil {
			return Version{}, fmt.Errorf("invalid semantic version %q: %w", raw, err)
		}
	}
	return v, nil
}

func parseNumericIdentifier(raw string) (string, error) {
	if raw == "" || (len(raw) > 1 && raw[0] == '0') {
		return "", fmt.Errorf("invalid numeric identifier %q", raw)
	}
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return "", fmt.Errorf("invalid numeric identifier %q", raw)
		}
	}
	return raw, nil
}

func parseIdentifiers(raw string, prerelease bool) ([]string, error) {
	parts := strings.Split(raw, ".")
	for _, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("empty identifier")
		}
		numeric := true
		for _, ch := range part {
			if (ch < '0' || ch > '9') && (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') && ch != '-' {
				return nil, fmt.Errorf("invalid identifier %q", part)
			}
			if ch < '0' || ch > '9' {
				numeric = false
			}
		}
		if prerelease && numeric && len(part) > 1 && part[0] == '0' {
			return nil, fmt.Errorf("numeric prerelease identifier %q has a leading zero", part)
		}
	}
	return parts, nil
}

func (v Version) String() string {
	out := fmt.Sprintf("%s.%s.%s", v.Major, v.Minor, v.Patch)
	if len(v.Prerelease) > 0 {
		out += "-" + strings.Join(v.Prerelease, ".")
	}
	if len(v.Build) > 0 {
		out += "+" + strings.Join(v.Build, ".")
	}
	return out
}

// Compare returns -1, 0, or 1 when v has lower, equal, or higher precedence
// than other. Build metadata does not affect precedence.
func (v Version) Compare(other Version) int {
	for _, pair := range [][2]string{{v.Major, other.Major}, {v.Minor, other.Minor}, {v.Patch, other.Patch}} {
		if comparison := compareNumericStrings(pair[0], pair[1]); comparison != 0 {
			return comparison
		}
	}
	if len(v.Prerelease) == 0 && len(other.Prerelease) == 0 {
		return 0
	}
	if len(v.Prerelease) == 0 {
		return 1
	}
	if len(other.Prerelease) == 0 {
		return -1
	}
	for i := 0; i < len(v.Prerelease) && i < len(other.Prerelease); i++ {
		left, right := v.Prerelease[i], other.Prerelease[i]
		if left == right {
			continue
		}
		leftNumeric := isNumeric(left)
		rightNumeric := isNumeric(right)
		switch {
		case leftNumeric && rightNumeric:
			return compareNumericStrings(left, right)
		case leftNumeric:
			return -1
		case rightNumeric:
			return 1
		case left < right:
			return -1
		default:
			return 1
		}
	}
	if len(v.Prerelease) < len(other.Prerelease) {
		return -1
	}
	if len(v.Prerelease) > len(other.Prerelease) {
		return 1
	}
	return 0
}

func isNumeric(raw string) bool {
	if raw == "" {
		return false
	}
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func compareNumericStrings(left, right string) int {
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

// CompareVersions parses and compares two semantic versions.
func CompareVersions(left, right string) (int, error) {
	a, err := ParseVersion(left)
	if err != nil {
		return 0, err
	}
	b, err := ParseVersion(right)
	if err != nil {
		return 0, err
	}
	return a.Compare(b), nil
}
