package yum

type Repository struct {
	Metadata *MetadataData
	Packages *PackageData
	Filelist *FilelistData
}

// Update updates this repository with a given RPMObject.
func (repo *Repository) Update(objects ...*RPMObject) bool {
	var packages bool
	var filelist bool
	for _, f := range objects {
		ok := repo.Packages.Add(f.Package())
		if ok {
			packages = true
		}
		ok = repo.Filelist.Add(f.Filelist())
		if ok {
			filelist = true
		}
	}
	return packages || filelist
}
