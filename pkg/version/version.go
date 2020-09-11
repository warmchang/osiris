package version

// Values for these are injected by the build
var (
	version string
	commit  string
	date    string
)

// Version returns the Osiris version. This is either a semantic version
// number or else, in the case of unreleased code, the string "devel".
func Version() string {
	return version
}

// Commit returns the git commit SHA for the code that Osiris was built from.
func Commit() string {
	return commit
}

// Date returns the date when Osiris was built.
func Date() string {
	return date
}
