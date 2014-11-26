package deb

// Represents a version in the deban packaging system
type Version struct {
	//It is a single small unsigned integer, default to zero
	Epoch uint32

	//The upstream version, should never be empty. starts by a number
	//and contains only [0-a-ZA-Z\.+~] and optionally : if epoch is
	//zero and - if there is no debian revision
	UpstreamVersion string

	//The debian version, optional contains only alphanum and + - ~ if 0 it can be ommited
	DebianRevision string
}

func ParseVersion(s string) (*Version, error) {
	return nil, NotYetImplemented()
}
