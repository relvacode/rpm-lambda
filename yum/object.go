package yum

import (
	"crypto/sha256"
	"fmt"
	"hash"
)

type Location struct {
	Href string `xml:"href,attr"`
}

type Checksum struct {
	Type     string `xml:"type,attr"`
	Checksum string `xml:",chardata"`
}

func (c Checksum) Equals(other Checksum) bool {
	return c.Type == other.Type && c.Checksum == other.Checksum
}

func NewChecksumGenerator(t string, f func() hash.Hash) *ChecksumGenerator {
	return &ChecksumGenerator{
		t:    t,
		Hash: f(),
	}
}

type ChecksumGenerator struct {
	t string
	hash.Hash
}

func (cs *ChecksumGenerator) Sum() Checksum {
	return Checksum{
		Type:     cs.t,
		Checksum: fmt.Sprintf("%x", cs.Hash.Sum(nil)),
	}
}

// SHA256 creates a new ChecksumGenerator using SHA256
func SHA256() *ChecksumGenerator {
	return NewChecksumGenerator("sha256", sha256.New)
}
