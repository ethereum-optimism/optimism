package release

import (
	"fmt"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

// BumpKind describes the requested semantic version bump for a component.
type BumpKind string

const (
	BumpPatch  BumpKind = "patch"
	BumpMinor  BumpKind = "minor"
	BumpMajor  BumpKind = "major"
	BumpManual BumpKind = "manual"
)

// Version is a canonicalized semantic version with a leading v prefix.
type Version struct {
	parsed *semver.Version
}

func ParseVersion(input string) (Version, error) {
	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return Version{}, fmt.Errorf("version is empty")
	}
	normalized = strings.TrimPrefix(normalized, "v")
	parsed, err := semver.StrictNewVersion(normalized)
	if err != nil {
		return Version{}, fmt.Errorf("invalid version %q: %w", input, err)
	}
	return Version{parsed: parsed}, nil
}

func MustParseVersion(input string) Version {
	v, err := ParseVersion(input)
	if err != nil {
		panic(err)
	}
	return v
}

func (v Version) String() string {
	if v.parsed == nil {
		return ""
	}
	return "v" + v.parsed.String()
}

func (v Version) Compare(other Version) int {
	switch {
	case v.parsed == nil && other.parsed == nil:
		return 0
	case v.parsed == nil:
		return -1
	case other.parsed == nil:
		return 1
	default:
		return v.parsed.Compare(other.parsed)
	}
}

func (v Version) Core() string {
	if v.parsed == nil {
		return ""
	}
	return fmt.Sprintf("v%d.%d.%d", v.parsed.Major(), v.parsed.Minor(), v.parsed.Patch())
}

func (v Version) IsRC() bool {
	return v.RCNumber() > 0
}

func (v Version) RCNumber() int {
	if v.parsed == nil {
		return 0
	}
	pre := v.parsed.Prerelease()
	if !strings.HasPrefix(pre, "rc.") {
		return 0
	}
	num, err := semver.NewVersion("0.0.0-" + pre)
	if err != nil {
		return 0
	}
	_ = num
	var rc int
	if _, err := fmt.Sscanf(pre, "rc.%d", &rc); err != nil {
		return 0
	}
	if rc < 1 {
		return 0
	}
	return rc
}

func (v Version) IsStableRelease() bool {
	if v.parsed == nil {
		return false
	}
	return v.parsed.Prerelease() == "" && v.parsed.Metadata() == ""
}

func (v Version) Stable() Version {
	if v.parsed == nil {
		return Version{}
	}
	return MustParseVersion(fmt.Sprintf("v%d.%d.%d", v.parsed.Major(), v.parsed.Minor(), v.parsed.Patch()))
}

func ValidateManualReleaseVersion(input string) (Version, error) {
	v, err := ParseVersion(input)
	if err != nil {
		return Version{}, err
	}
	if !v.IsStableRelease() {
		return Version{}, fmt.Errorf("manual release version must be a stable release, got %q", v.String())
	}
	return v, nil
}

func NextReleaseVersion(latestRelease string, bump BumpKind) (Version, error) {
	base, err := ValidateManualReleaseVersion(latestRelease)
	if err != nil {
		return Version{}, fmt.Errorf("parse latest release: %w", err)
	}
	var next Version
	switch bump {
	case BumpPatch:
		next = MustParseVersion(fmt.Sprintf("v%d.%d.%d", base.parsed.Major(), base.parsed.Minor(), base.parsed.Patch()+1))
	case BumpMinor:
		next = MustParseVersion(fmt.Sprintf("v%d.%d.%d", base.parsed.Major(), base.parsed.Minor()+1, 0))
	case BumpMajor:
		next = MustParseVersion(fmt.Sprintf("v%d.%d.%d", base.parsed.Major()+1, 0, 0))
	default:
		return Version{}, fmt.Errorf("unsupported bump kind %q", bump)
	}
	return next, nil
}

func NextRCVersion(targetRelease string, latestRC string) (Version, error) {
	target, err := ValidateManualReleaseVersion(targetRelease)
	if err != nil {
		return Version{}, fmt.Errorf("parse target release: %w", err)
	}
	if strings.TrimSpace(latestRC) == "" {
		return MustParseVersion(target.Core() + "-rc.1"), nil
	}
	rc, err := ParseVersion(latestRC)
	if err != nil {
		return Version{}, fmt.Errorf("parse latest rc: %w", err)
	}
	if !rc.IsRC() {
		return Version{}, fmt.Errorf("latest rc version %q is not an rc release", rc.String())
	}
	if rc.Core() != target.Core() {
		return MustParseVersion(target.Core() + "-rc.1"), nil
	}
	return MustParseVersion(fmt.Sprintf("%s-rc.%d", target.Core(), rc.RCNumber()+1)), nil
}

func ProposeNextRC(latestRelease string, latestRC string, bump BumpKind) (Version, error) {
	target, err := NextReleaseVersion(latestRelease, bump)
	if err != nil {
		return Version{}, err
	}
	return NextRCVersion(target.String(), latestRC)
}

func ProposeNextRCFromManualTarget(manualTarget string, latestRC string) (Version, error) {
	stable, err := ValidateManualReleaseVersion(manualTarget)
	if err != nil {
		return Version{}, err
	}
	return NextRCVersion(stable.String(), latestRC)
}
