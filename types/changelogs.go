package types

type ChangelogEntry struct {
	Version          string   `json:"version" validate:"required" description:"The version for the changelog entry. (4.3.0 etc.)"`
	ExtraDescription string   `json:"extra_description" description:"The extra description for the version, if applicable"`
	Prerelease       bool     `json:"prerelease" validate:"required" description:"Whether or not this is a prerelease."`
	Added            []string `json:"added" validate:"required" description:"The added features for the version."`
	Updated          []string `json:"updated" validate:"required" description:"The changed features for the version."`
	Removed          []string `json:"removed" validate:"required" description:"The removed features for the version."`
}

type Changelog struct {
	Entries []ChangelogEntry `json:"entries" validate:"required,dive,required" description:"The changelog entries."`
}
