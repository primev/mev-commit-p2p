package mevcommit

var (
	version    string
	commitHash string
)

// Version returns the version of the binary.
func Version() string {
	if version == "" || commitHash == "" {
		return "dev-dirty"
	}
	return version + "-" + commitHash
}
