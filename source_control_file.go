package deb

import "net/mail"

// Represents a .dsc file content Only Mandatory according to the
// Debian Policy Manual file are represented.
type SourceControlFile struct {
	Identifier SourcePackageRef

	BasePath string
	//A Format for a source file can be 1.0 3.0 (native) or 3.0 (quilt)
	Format string

	// The maintainer email address, which is mandatory
	Maintainer *mail.Address

	// A list of md5 checksumed files
	Md5Files []FileReference
	// A list of sha1 checksumed files
	Sha1Files []FileReference
	// A list of sha256 checksumed files
	Sha256Files []FileReference
}
