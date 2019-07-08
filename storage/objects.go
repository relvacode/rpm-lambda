package storage

import (
	"git.illumina.com/relvacode/rpm-lambda/yum"
	"time"
)

const (
	RepoMDXML   = "repodata/repomd.xml"
	PrimaryXML  = "repodata/primary.xml.gz"
	FilelistXML = "repodata/filelists.xml.gz"
)

type XMLObject struct {
	Key             string
	ObjectChecksum  yum.Checksum
	ContentChecksum yum.Checksum
}

type FilelistXMLObject struct {
	XMLObject
}

func (o FilelistXMLObject) Metadata() yum.Metadata {
	return yum.Metadata{
		Type: "filelists",
		Location: yum.Location{
			Href: o.XMLObject.Key,
		},
		Timestamp:       time.Now().Unix(),
		Checksum:        o.XMLObject.ObjectChecksum,
		ContentChecksum: o.XMLObject.ContentChecksum,
	}
}

type PrimaryXMLObject struct {
	XMLObject
}

func (p PrimaryXMLObject) Metadata() yum.Metadata {
	return yum.Metadata{
		Type: "primary",
		Location: yum.Location{
			Href: p.XMLObject.Key,
		},
		Timestamp:       time.Now().Unix(),
		Checksum:        p.XMLObject.ObjectChecksum,
		ContentChecksum: p.XMLObject.ContentChecksum,
	}
}
