package yum

import (
	"context"
	"github.com/sassoftware/go-rpmutils"
	"io"
	"io/ioutil"
)

type Size struct {
	Archive   int64 `xml:"archive,attr"`
	Package   int64 `xml:"package,attr"`
	Installed int64 `xml:"installed,attr"`
}

type RPM struct {
	Release  *rpmutils.NEVRA
	Checksum Checksum
	Size     Size
	Files    []rpmutils.FileInfo
}

type RPMObject struct {
	Key string
	RPM
}

func (f *RPMObject) Package() Package {
	return Package{
		Type: "rpm",
		Name: f.Release.Name,
		Arch: f.Release.Arch,
		Version: Version{
			Epoch: f.Release.Epoch,
			Rel:   f.Release.Release,
			Ver:   f.Release.Version,
		},
		Size: f.Size,
		Location: Location{
			Href: f.Key,
		},
		Checksum: PackageChecksum{
			PkgId:    "YES",
			Checksum: f.Checksum,
		},
	}
}

func (f *RPMObject) Filelist() Filelist {
	files := make([]string, len(f.Files))
	for i, fn := range f.Files {
		files[i] = fn.Name()
	}

	return Filelist{
		PkgID: f.Checksum.Checksum,
		Name:  f.RPM.Release.Name,
		Arch:  f.RPM.Release.Arch,
		Files: files,
		Version: Version{
			Epoch: f.Release.Epoch,
			Ver:   f.Release.Version,
			Rel:   f.Release.Release,
		},
	}
}

type byteCounter struct {
	count int
}

func (bc *byteCounter) Size() int {
	return bc.count
}

func (bc *byteCounter) Write(b []byte) (int, error) {
	bc.count += len(b)
	return len(b), nil
}

func ScanRPM(ctx context.Context, data io.Reader) (*RPM, error) {
	var (
		checksum = SHA256()
		bc       = new(byteCounter)
		r        = io.TeeReader(data, io.MultiWriter(bc, checksum))
	)

	rpm, err := rpmutils.ReadHeader(r)
	if err != nil {
		return nil, err
	}

	release, err := rpm.GetNEVRA()
	if err != nil {
		return nil, err
	}

	files, err := rpm.GetFiles()
	if err != nil {
		return nil, err
	}

	installed, err := rpm.InstalledSize()
	if err != nil {
		return nil, err
	}

	payload, err := rpm.PayloadSize()
	if err != nil {
		return nil, err
	}

	// ensure any remaining data is consumed to the sha1 writer
	_, err = io.Copy(ioutil.Discard, r)
	if err != nil {
		return nil, err
	}

	return &RPM{
		Release: release,
		Files:   files,
		Size: Size{
			Package:   int64(bc.Size()),
			Archive:   payload,
			Installed: installed,
		},
		Checksum: checksum.Sum(),
	}, nil
}
