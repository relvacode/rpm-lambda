package yum

import (
	"encoding/xml"
)

type Version struct {
	Epoch string `xml:"epoch,attr"`
	Rel   string `xml:"rel,attr"`
	Ver   string `xml:"ver,attr"`
}

func (v Version) Equals(other Version) bool {
	return v.Ver == other.Ver && v.Rel == other.Rel && v.Epoch == other.Epoch
}

type PackageChecksum struct {
	PkgId string `xml:"pkgid,attr"`
	Checksum
}

type Package struct {
	Type     string          `xml:"type,attr"`
	Name     string          `xml:"name"`
	Arch     string          `xml:"arch"`
	Version  Version         `xml:"version"`
	Location Location        `xml:"location"`
	Size     Size            `xml:"size"`
	Checksum PackageChecksum `xml:"checksum"`
}

// Equals returns true if this PackageData is equal to another in terms of Architecture and Version
func (p Package) Equals(other Package) bool {
	return p.Name == other.Name && p.Arch == other.Arch && p.Version.Equals(other.Version)
}

type PackageData struct {
	XMLName      xml.Name
	PackageCount int       `xml:"packages,attr"`
	Packages     []Package `xml:"package"`
}

// Add a Package to this package list or updates an existing one if the checksum isn't equal.
// Returns true if the package list was updated.
func (pd *PackageData) Add(pkg Package) bool {
	for i, f := range pd.Packages {
		if pkg.Equals(f) {
			if !pkg.Checksum.Equals(f.Checksum.Checksum) {
				pd.Packages[i] = pkg
				return true
			}
			return false
		}
	}
	pd.Packages = append(pd.Packages, pkg)
	pd.PackageCount = len(pd.Packages)
	return true
}
