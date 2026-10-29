package storage

import "runtime"

// LegacyMetadataPrefix is the metadata namespace used by the former ocis and
// s3ng storage drivers.
func LegacyMetadataPrefix() string {
	if runtime.GOOS == "freebsd" {
		return "ocis."
	}
	return "user.ocis."
}
