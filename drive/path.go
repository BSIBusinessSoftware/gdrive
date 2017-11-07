package drive

import (
	"fmt"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

const RemotePathSep = "/"

var defaultGetFields []googleapi.Field
var defaultQueryFields []googleapi.Field

func init() {
	defaultGetFields = []googleapi.Field{"id", "name", "md5Checksum", "mimeType", "size", "createdTime", "parents"}
	defaultQueryFields = []googleapi.Field{"nextPageToken", "files(id,name,md5Checksum,mimeType,size,createdTime,parents)"}
}

func (self *Drive) newPathFinder() *remotePathFinder {
	return &remotePathFinder{
		service: self.service.Files,
		caches:  make(map[string]*fileEntry),
	}
}

type fileEntry struct {
	file    *drive.File
	absPath string
}

type remotePathFinder struct {
	service *drive.FilesService
	caches  map[string]*fileEntry // id -> entry
}

func (self *remotePathFinder) GetAbsPath(f *drive.File) (string, error) {

	if len(f.Parents) == 0 {
		return RemotePathSep, nil
	}
	if cache, ok := self.caches[f.Id]; ok {
		if len(cache.absPath) > 0 {
			return cache.absPath, nil
		}
	} else {
		self.saveCache(f, "")
	}

	var path []string

	for {
		parent, err := self.GetFile(f.Parents[0])
		if err != nil {
			return "", err
		}

		// Stop when we find the root dir
		if len(parent.Parents) == 0 {
			break
		}

		// Insert parent name at beginning
		path = append([]string{f.Name}, path...)
		f = parent
	}

	absPath := RemotePathSep + strings.Join(append(path, f.Name), RemotePathSep)

	// Cache absPath
	self.caches[f.Id].absPath = absPath

	return absPath, nil
}

func (self *remotePathFinder) JoinPath(pathes ...string) string {
	items := []string{}
	for _, path := range pathes {
		path = strings.TrimSuffix(path, RemotePathSep)
		items = append(items, path)
	}
	return strings.Join(items, RemotePathSep)
}

func (self *remotePathFinder) GetFile(id string) (*drive.File, error) {
	// Check cache
	if entry, ok := self.caches[id]; ok {
		return entry.file, nil
	}

	// Fetch file from drive
	f, err := self.service.Get(string(id)).Fields(defaultGetFields...).Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to get file: %s", err)
	}

	self.saveCache(f, "")

	return f, nil
}

func (self *remotePathFinder) GetFileId(absPath string) (string, error) {
	if !strings.HasPrefix(absPath, "/") {
		return "", fmt.Errorf("'%s' is not absolute path", absPath)
	}

	absPath = strings.TrimRight(absPath, "/")
	if absPath == "" {
		return "root", nil
	}

	// Check cache
	for _, entry := range self.caches {
		if entry.absPath == absPath {
			return entry.file.Id, nil
		}
	}

	pathes := strings.Split(absPath[1:], "/")
	var parent string = "root"
	var f *drive.File
	for _, path := range pathes {
		entry := self.queryEntryByName(path, parent)
		if entry == nil {
			return "", fmt.Errorf("path not found: '%v'", absPath)
		}
		f = entry
		parent = f.Id
	}

	self.saveCache(f, absPath)

	return parent, nil
}

func (self *remotePathFinder) SecureFileId(expr string) string {
	if strings.Contains(expr, "/") {
		id, err := self.GetFileId(expr)
		if err == nil {
			return string(id)
		}
	}
	return expr
}

func (self *remotePathFinder) queryEntryByName(name string, parentId string) *drive.File {

	// Check cache
	for _, entry := range self.caches {
		if entry.file.Name == name && entry.file.Parents[0] == parentId {
			return entry.file
		}
	}

	conditions := []string{
		"trashed = false",
		fmt.Sprintf("name = '%v'", name),
		fmt.Sprintf("'%v' in parents", parentId),
	}
	query := strings.Join(conditions, " and ")

	var files []*drive.File
	self.service.List().Q(query).Fields(defaultQueryFields...).Pages(context.TODO(), func(fl *drive.FileList) error {
		files = append(files, fl.Files...)
		return nil
	})

	if len(files) == 0 {
		return nil
	}

	for _, f := range files {
		self.saveCache(f, "")
	}

	return files[0]
}

func (self *remotePathFinder) saveCache(f *drive.File, absPath string) {
	self.caches[f.Id] = &fileEntry{
		file:    f,
		absPath: absPath,
	}
}

func isDoc(f *drive.File) bool {
	if isDir(f) {
		return false
	}
	if isBinary(f) {
		return false
	}
	return true
}
