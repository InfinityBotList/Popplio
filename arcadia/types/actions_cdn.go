package types

import "fmt"

// CdnAssetAction is the union of CDN filesystem operations. ListPath, ReadFile,
// CreateFolder and Delete are unit variants and arrive as bare JSON strings.
type CdnAssetAction struct {
	ListPath     *Unit
	ReadFile     *Unit
	CreateFolder *Unit
	AddFile      *CdnAddFile
	CopyFile     *CdnCopyFile
	Delete       *Unit
	PersistGit   *CdnPersistGit
}

type CdnAddFile struct {
	Overwrite bool     `json:"overwrite"`
	Chunks    []string `json:"chunks"`
	SHA512    string   `json:"sha512"`
}

type CdnCopyFile struct {
	Overwrite      bool   `json:"overwrite"`
	DeleteOriginal bool   `json:"delete_original"`
	CopyTo         string `json:"copy_to"`
}

type CdnPersistGit struct {
	Message    string `json:"message"`
	CurrentDir bool   `json:"current_dir"`
}

func (a *CdnAssetAction) UnmarshalJSON(data []byte) error {
	*a = CdnAssetAction{}

	return unmarshalUnion("CdnAssetAction", data, map[string]func() any{
		"ListPath": func() any {
			a.ListPath = unitSet()
			return nil
		},
		"ReadFile": func() any {
			a.ReadFile = unitSet()
			return nil
		},
		"CreateFolder": func() any {
			a.CreateFolder = unitSet()
			return nil
		},
		"AddFile": func() any {
			a.AddFile = &CdnAddFile{}
			return a.AddFile
		},
		"CopyFile": func() any {
			a.CopyFile = &CdnCopyFile{}
			return a.CopyFile
		},
		"Delete": func() any {
			a.Delete = unitSet()
			return nil
		},
		"PersistGit": func() any {
			a.PersistGit = &CdnPersistGit{}
			return a.PersistGit
		},
	})
}

func (a CdnAssetAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.ListPath != nil:
		return encodeUnit("ListPath")
	case a.ReadFile != nil:
		return encodeUnit("ReadFile")
	case a.CreateFolder != nil:
		return encodeUnit("CreateFolder")
	case a.AddFile != nil:
		return encodeVariant("AddFile", a.AddFile)
	case a.CopyFile != nil:
		return encodeVariant("CopyFile", a.CopyFile)
	case a.Delete != nil:
		return encodeUnit("Delete")
	case a.PersistGit != nil:
		return encodeVariant("PersistGit", a.PersistGit)
	default:
		return nil, fmt.Errorf("CdnAssetAction: no variant set")
	}
}

// CdnAssetItem is one entry of a ListPath response.
type CdnAssetItem struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Size         uint64 `json:"size"`
	LastModified uint64 `json:"last_modified"`
	IsDir        bool   `json:"is_dir"`
	Permissions  uint32 `json:"permissions"`
}
