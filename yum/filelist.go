package yum

import (
	"encoding/xml"
)

type Filelist struct {
	PkgID   string   `xml:"pkgid,attr"`
	Name    string   `xml:"name,attr"`
	Arch    string   `xml:"arch,attr"`
	Version Version  `xml:"version"`
	Files   []string `xml:"file"`
}

type FilelistData struct {
	XMLName      xml.Name
	PackageCount int        `xml:"packages,attr"`
	Packages     []Filelist `xml:"package"`
}

func (fl *FilelistData) Add(f Filelist) bool {
	for i, p := range fl.Packages {
		if p.PkgID == f.PkgID {
			fl.Packages[i] = f
			return true
		}
	}

	fl.Packages = append(fl.Packages, f)
	fl.PackageCount = len(fl.Packages)
	return true
}