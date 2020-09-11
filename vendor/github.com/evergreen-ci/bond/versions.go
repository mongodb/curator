/*
MongoDB Versions

The MongoDBVersion type provides support for interacting with MongoDB
versions. This type makes it possible to validate MongoDB version
numbers and ask common questions about MongoDB versions.
*/
package bond

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
)

const (
	endOfLegacy   = "4.5.0-alpha0"
	firstLTS      = "5.0.0"
	devReleaseTag = "alpha"
)

// MongoDBVersion encapsulates information about a MongoDB version.
// Use the associated methods to ask questions about MongoDB
// versions. All parsing of versions happens during construction, and
// individual method calls are very light-weight. Note that
// not all methods are applicable for all versions.
type MongoDBVersion interface {
	// String returns a string representation of the MongoDB version number.
	String() string
	// Parsed returns the parsed version object for the version.
	Parsed() semver.Version
	// Series returns the first two components for the version.
	Series() string
	// IsReleaseCandidate returns true if the version is a release candidate.
	IsReleaseCandidate() bool
	// IsDevelopmentRelease returns true if the version is a development release.
	IsDevelopmentRelease() bool
	// DevelopmentReleaseNumber returns the development release number, if applicable.
	DevelopmentReleaseNumber() int
	// RCNumber returns the RC counter (or -1 if not a release candidate).
	RCNumber() int
	// IsLTS returns true if the release is long-term supported, i.e. the yearly release.
	IsLTS() bool
	// LTS returns most recent LTS series, which may be itself, if applicable.
	LTS() string
	// IsContinuous returns true if the release is a quarterly (non-LTS) release.
	IsContinuous() bool
	// IsRelease returns true if the version is a release.
	IsRelease() bool
	// IsDevelopmentBuild returns true for non-release versions.
	IsDevelopmentBuild() bool
	// IsStableSeries returns true if the legacy version is a stable series.
	IsStableSeries() bool
	// IsDevelopmentSeries returns true if the legacy version is a development series.
	IsDevelopmentSeries() bool
	// StableReleaseSeries returns true if the legacy version is a stable release series.
	StableReleaseSeries() string
	// IsInitialStableReleaseCandidate returns true if the legacy version is a release
	// candidate for the initial release of a stable series.
	IsInitialStableReleaseCandidate() bool

	IsLessThan(version MongoDBVersion) bool
	IsLessThanOrEqualTo(version MongoDBVersion) bool
	IsGreaterThan(version MongoDBVersion) bool
	IsGreaterThanOrEqualTo(version MongoDBVersion) bool
	IsEqualTo(version MongoDBVersion) bool
	IsNotEqualTo(version MongoDBVersion) bool
}

// LegacyMongoDBVersion is a structure representing a version identifier for legacy versions of
// MongoDB, which implements the MongoDBVersion interface.
type LegacyMongoDBVersion struct {
	source   string
	parsed   semver.Version
	isRc     bool
	isDev    bool
	rcNumber int
	series   string
	tag      string
}

// NewMongoDBVersion is a structure representing a version identifier for versions of
// MongoDB, which implements the MongoDBVersion.
type NewMongoDBVersion struct {
	LegacyMongoDBVersion // note not all fields are applicable to NewMongoDBVersion
	isDevRelease         bool
	devReleaseNumber     int
	quarter              string
}

// IsStableSeries is not applicable to new versions, so always return false.
func (v *NewMongoDBVersion) IsStableSeries() bool {
	return false
}

// IsDevelopmentSeries is not applicable to new versions, so always return false.
func (v *NewMongoDBVersion) IsDevelopmentSeries() bool {
	return false
}

// IsInitialStableReleaseCandidate is not applicable to new versions, so always return false.
func (v *NewMongoDBVersion) IsInitialStableReleaseCandidate() bool {
	return false
}

// StableReleaseSeries is not applicable to new versions, so always return the empty string.
func (v *NewMongoDBVersion) StableReleaseSeries() string {
	return ""
}

// Series returns the major and quarter for the version.
func (v *NewMongoDBVersion) Series() string {
	return v.series
}

// IsLTS returns true if this is the first release of the year.
func (v *NewMongoDBVersion) IsLTS() bool {
	return v.IsRelease() && v.Parsed().Minor == 0
}

// LTS returns the most recent LTS series.
func (v *NewMongoDBVersion) LTS() string {
	firstLTSVersion, _ := semver.Parse(firstLTS)
	if v.Parsed().LT(firstLTSVersion) {
		// Return empty string for versions that are not preceded by an
		// LTS series.
		return ""
	}

	return fmt.Sprintf("%d.0", v.Parsed().Major)
}

// func IsContinuous returns true if the version is a continuous release.
func (v *NewMongoDBVersion) IsContinuous() bool {
	return v.IsRelease() && v.Parsed().Minor != 0
}

// IsDevelopmentRelease returns true if the version is a development release.
func (v *NewMongoDBVersion) IsDevelopmentRelease() bool {
	return v.isDevRelease
}

// DevelopmentReleaseNumber returns the number of the development release,
// or -1 if not applicable.
func (v *NewMongoDBVersion) DevelopmentReleaseNumber() int {
	return v.devReleaseNumber
}

// CreateMongoDBVersion returns an implementation of the MongoDBVersion.
// If the parsed version is before 4.5.0, then we use the legacy structure.
// Otherwise, we use the modern versioning scheme.
func CreateMongoDBVersion(version string) (MongoDBVersion, error) {
	endOfLegacyVersion, _ := semver.Parse(endOfLegacy)
	v, err := createLegacyMongoDBVersion(version)
	if err != nil {
		return nil, errors.Wrapf(err, "creating initial version")
	}
	if v.Parsed().LT(endOfLegacyVersion) {
		return v, nil
	}
	return createNewMongoDBVersion(*v)
}

// createNewMongoDBVersion takes a string representing a MongoDBVersion and
// returns a NewMongoDBVersion object. All parsing of a version happens during this phase.
func createNewMongoDBVersion(parsedVersion LegacyMongoDBVersion) (*NewMongoDBVersion, error) {
	v := &NewMongoDBVersion{LegacyMongoDBVersion: parsedVersion, devReleaseNumber: -1}
	var err error
	if len(v.String()) < 3 {
		return nil, errors.Errorf("version '%s' is invalid", v.String())
	}
	v.quarter = v.String()[:3]
	if strings.Contains(v.tag, devReleaseTag) {
		v.isDev = false
		v.isDevRelease = true
		if len(v.tag) > len(devReleaseTag) {
			v.devReleaseNumber, err = strconv.Atoi(v.tag[len(devReleaseTag):])
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't parse development release number")
			}
		}
	}

	if v.isDev {
		return nil, errors.New("development builds are not allowed in the new versioning scheme")
	}

	return v, nil
}

// createLegacyMongoDBVersion takes a string representing a MongoDB version and
// returns a LegacyMongoDBVersion object. All parsing of a version happens during this phase.
func createLegacyMongoDBVersion(version string) (*LegacyMongoDBVersion, error) {
	v := &LegacyMongoDBVersion{source: version, rcNumber: -1}
	if strings.HasSuffix(version, "-") {
		v.isDev = true

		if !strings.Contains(version, "pre") {
			version += "pre-"
		}
	}
	if strings.Contains(version, "~") {
		versionParts := strings.Split(version, "~")
		version = versionParts[0]
		version += "-pre-"
		v.tag = strings.Join(versionParts[1:], "")
		v.isDev = true
	}

	parsed, err := semver.Parse(version)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing '%s'", version)
	}
	v.parsed = parsed

	if strings.Contains(version, "rc") {
		v.isRc = true
	}

	tagParts := strings.Split(version, "-")
	if len(tagParts) > 1 {
		v.tag = strings.Join(tagParts[1:], "-")

		if v.isRc {
			// Prerelease may have +buildinfo suffix, like: 1.0.0-rc0+buildinfo
			rcPart := strings.Split(tagParts[1], "+")

			v.rcNumber, err = strconv.Atoi(rcPart[0][2:])
			if err != nil {
				return nil, errors.Wrapf(err, "couldn't parse release candidate number")
			}
			if len(tagParts) > 2 {
				v.isDev = true
			}
		} else {
			v.isDev = true
		}

	}
	if len(version) < 3 {
		return nil, errors.Errorf("version '%s' is invalid", version)
	}
	v.series = fmt.Sprintf("%d.%d", v.Parsed().Major, v.Parsed().Minor)
	return v, err
}

// ConvertVersion takes an un-typed object and attempts to convert it to a
// version object. For use with compactor functions.
func ConvertVersion(v interface{}) (MongoDBVersion, error) {
	switch version := v.(type) {
	case *LegacyMongoDBVersion:
		return version, nil
	case LegacyMongoDBVersion:
		return &version, nil
	case *NewMongoDBVersion:
		return version, nil
	case NewMongoDBVersion:
		return &version, nil
	case MongoDBVersion:
		return version, nil
	case string:
		output, err := CreateMongoDBVersion(version)
		if err != nil {
			return nil, err
		}
		return output, nil
	case semver.Version:
		return CreateMongoDBVersion(version.String())
	default:
		return nil, fmt.Errorf("%v is not a valid version type (%T)", version, version)
	}
}

// String returns a string representation of the MongoDB version number.
func (v *LegacyMongoDBVersion) String() string {
	return v.source
}

// Parsed returns the parsed version object for the version.
func (v *LegacyMongoDBVersion) Parsed() semver.Version {
	return v.parsed
}

// Series return the release series, generally the first two
// components of a version. For example for 3.2.6, the series is 3.2.
func (v *LegacyMongoDBVersion) Series() string {
	return v.series
}

// IsReleaseCandidate returns true for releases that have the "rc[0-9]"
// tag and false otherwise.
func (v *LegacyMongoDBVersion) IsReleaseCandidate() bool {
	return v.IsRelease() && v.isRc
}

// IsStableSeries returns true for stable releases, ones where the
// second component of the version string (i.e. "Minor" in semantic
// versioning terms) are even, and false otherwise.
func (v *LegacyMongoDBVersion) IsStableSeries() bool {
	return v.parsed.Minor%2 == 0
}

// IsDevelopmentSeries returns true for development (snapshot)
// releases. These versions are those where the second component
// (e.g. "Minor" in semantic versioning terms) are odd, and false
// otherwise.
func (v *LegacyMongoDBVersion) IsDevelopmentSeries() bool {
	return !v.IsStableSeries()
}

// StableReleaseSeries returns a series string (e.g. X.Y) for this
// version. For stable releases, the output is the same as
// .Series(). For development releases, this method returns the *next*
// stable series.
func (v *LegacyMongoDBVersion) StableReleaseSeries() string {
	if v.IsStableSeries() {
		return v.Series()
	}

	if v.parsed.Minor < 9 {
		return fmt.Sprintf("%d.%d", v.parsed.Major, v.parsed.Minor+1)
	}

	return fmt.Sprintf("%d.0", v.parsed.Major+1)
}

// IsRelease returns true for all version strings that refer to a
// release, including development, release candidate and GA releases,
// and false otherwise. Other builds, including test builds and
// "nightly" snapshots of MongoDB have version strings, but are not
// releases.
func (v *LegacyMongoDBVersion) IsRelease() bool {
	return !v.isDev
}

// IsLTS isn't applicable to legacy versions so we return false.
func (v *LegacyMongoDBVersion) IsLTS() bool {
	return false
}

// LTS isn't applicable to legacy version so we return an empty string.
func (v *LegacyMongoDBVersion) LTS() string {
	return ""
}

// IsContinuous isn't applicable to legacy versions so return false.
func (v *LegacyMongoDBVersion) IsContinuous() bool {
	return false
}

// IsDevelopmentRelease returns true if the version refers to a development release.
func (v *LegacyMongoDBVersion) IsDevelopmentRelease() bool {
	return v.IsDevelopmentSeries() && v.IsRelease()
}

// DevelopmentReleaseNumber is not applicable to legacy versions, so it returns -1.
func (v *LegacyMongoDBVersion) DevelopmentReleaseNumber() int {
	return -1
}

// IsDevelopmentBuild returns true for all non-release builds,
// including nightly snapshots and all testing and development
// builds.
func (v *LegacyMongoDBVersion) IsDevelopmentBuild() bool {
	return v.isDev
}

// IsInitialStableReleaseCandidate returns true for release
// candidates for the initial public release of a new stable release
// series.
func (v *LegacyMongoDBVersion) IsInitialStableReleaseCandidate() bool {
	if v.IsStableSeries() {
		return v.parsed.Patch == 0 && v.IsReleaseCandidate()
	}
	return false
}

// RCNumber returns an integer for the RC counter. For non-rc releases,
// returns -1.
func (v *LegacyMongoDBVersion) RCNumber() int {
	return v.rcNumber
}

// IsLessThan returns true when "version" is less than (e.g. earlier)
// than the object itself.
func (v *LegacyMongoDBVersion) IsLessThan(version MongoDBVersion) bool {
	return v.Parsed().LT(version.Parsed())
}

// IsLessThanOrEqualTo returns true when "version" is less than or
// equal to (e.g. earlier or the same as) the object itself.
func (v *LegacyMongoDBVersion) IsLessThanOrEqualTo(version MongoDBVersion) bool {
	// semver considers release candidates equal to GA, so we have to special case this

	if v.IsEqualTo(version) {
		return true
	}

	return v.Parsed().LT(version.Parsed())
}

// IsGreaterThan returns true when "version" is greater than (e.g. later)
// than the object itself.
func (v *LegacyMongoDBVersion) IsGreaterThan(version MongoDBVersion) bool {
	return v.Parsed().GT(version.Parsed())
}

// IsGreaterThanOrEqualTo returns true when "version" is greater than
// or equal to (e.g. the same as or later than) the object itself.
func (v *LegacyMongoDBVersion) IsGreaterThanOrEqualTo(version MongoDBVersion) bool {
	if v.IsEqualTo(version) {
		return true
	}
	return v.Parsed().GT(version.Parsed())
}

// IsEqualTo returns true when "version" is the same as the object
// itself.
func (v *LegacyMongoDBVersion) IsEqualTo(version MongoDBVersion) bool {
	return v.String() == version.String()
}

// IsNotEqualTo returns true when "version" is the different from the
// object itself.
func (v *LegacyMongoDBVersion) IsNotEqualTo(version MongoDBVersion) bool {
	return v.String() != version.String()
}

/////////////////////////////////////////////
//
// Support for Sorting Slices of MongoDB Versions
//
/////////////////////////////////////////////

// MongoDBVersionSlice is an alias for []MongoDBVersion that supports
// the sort.Sorter interface, and makes it possible to sort slices of
// MongoDB versions.
type MongoDBVersionSlice []MongoDBVersion

// Len is  required  by the sort.Sorter interface. Returns
// the length of the slice.
func (s MongoDBVersionSlice) Len() int {
	return len(s)
}

// Less is a required by the sort.Sorter interface. Uses blang/semver
// to compare two versions.
func (s MongoDBVersionSlice) Less(i, j int) bool {
	left := s[i]
	right := s[j]

	return left.Parsed().LT(right.Parsed())
}

// Swap is a required by the sort.Sorter interface. Changes the
// position of two elements in the slice.
func (s MongoDBVersionSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// String() adds suport for the Stringer interface, which makes it
// possible to print slices of MongoDB versions as comma separated
// lists.
func (s MongoDBVersionSlice) String() string {
	var out []string

	for _, v := range s {
		if len(v.String()) == 0 {
			// some elements end up empty.
			continue
		}

		out = append(out, v.String())
	}

	return strings.Join(out, ", ")
}

// Sort provides a wrapper around sort.Sort() for slices of MongoDB
// versions objects.
func (s MongoDBVersionSlice) Sort() {
	sort.Sort(s)
}
