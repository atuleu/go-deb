package deb

import (
	"net/mail"
	"time"
)

type ChangesFileRef struct {
	Identifier SourcePackageRef
	Suffix     string
}

// Represents a .changes file content
type ChangesFile struct {
	Ref ChangesFileRef

	//The format of the change file itself
	Format     Version
	Date       time.Time
	Arch       string
	Dist       Distribution
	Maintainer *mail.Address

	Description string
	Changes     string

	Md5Files    []FileReference
	Sha1Files   []FileReference
	Sha256Files []FileReference
}
