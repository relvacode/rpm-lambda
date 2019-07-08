package yum

import (
	"encoding/xml"
)

type Metadata struct {
	Type      string   `xml:"type,attr"`
	Location  Location `xml:"location"`
	Timestamp int64    `xml:"timestamp"`

	Checksum        Checksum `xml:"checksum"`
	ContentChecksum Checksum `xml:"open-checksum"`
}

type MetadataData struct {
	XMLName xml.Name
	Data    []Metadata `xml:"data"`
}

func (md MetadataData) IndexOf(t string) int {
	for i, d := range md.Data {
		if d.Type == t {
			return i
		}
	}
	return -1
}

func (md *MetadataData) Update(d Metadata) {
	ix := md.IndexOf(d.Type)
	if ix == -1 {
		md.Data = append(md.Data, d)
	} else {
		md.Data[ix] = d
	}
}
