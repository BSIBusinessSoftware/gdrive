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
		caches:  make(map[fileId]*fileEntry),
	}
}

type fileId string

type fileEntry struct {
	file    *drive.File
	absPath string
}

type remotePathFinder struct {
	service *drive.FilesService
	caches  map[fileId]*fileEntry // id -> entry
}

func (self *remotePathFinder) GetAbsPath(f *drive.File) (string, error) {

	if len(f.Parents) == 0 {
		return RemotePathSep, nil
	}
	id := fileId(f.Id)
	if cache, ok := self.caches[id]; ok {
		if len(cache.absPath) > 0 {
			fmt.Printf("hit %v\n", cache.absPath)
			return cache.absPath, nil
		}
	} else {
		self.caches[id] = &fileEntry{
			file:    f,
			absPath: "",
		}
	}

	var path []string

	for {
		parent, err := self.GetFile(fileId(f.Parents[0]))
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

	// Save in cache
	self.caches[id].absPath = absPath

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

func (self *remotePathFinder) GetFile(id fileId) (*drive.File, error) {
	// Check cache
	if entry, ok := self.caches[id]; ok {
		fmt.Printf("hit %v\n", entry.file.Id)
		return entry.file, nil
	}

	// Fetch file from drive
	f, err := self.service.Get(string(id)).Fields(defaultGetFields...).Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to get file: %s", err)
	}

	// Save in cache
	self.caches[id] = &fileEntry{
		file:    f,
		absPath: "",
	}

	return f, nil
}

func (self *remotePathFinder) GetFileId(absPath string) (fileId, error) {
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
			fmt.Printf("hit %v\n", entry.file.Id)
			return fileId(entry.file.Id), nil
		}
	}

	pathes := strings.Split(absPath[1:], "/")
	var parent fileId = "root"
	var f *drive.File
	for _, path := range pathes {
		entry := self.queryEntryByName(path, fileId(parent))
		if entry == nil {
			return "", fmt.Errorf("path not found: '%v'", absPath)
		}
		f = entry
		parent = fileId(f.Id)
	}

	// Save in Cache
	self.caches[fileId(f.Id)] = &fileEntry{
		file:    f,
		absPath: absPath,
	}
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

func (self *remotePathFinder) queryEntryByName(name string, parent fileId) *drive.File {

	// Check cache
	{
		id := string(parent)
		for _, entry := range self.caches {
			if entry.file.Name == name && entry.file.Parents[0] == id {
				fmt.Printf("hit %v\n", name)
				return entry.file
			}
		}
	}

	conditions := []string{
		"trashed = false",
		fmt.Sprintf("name = '%v'", name),
		fmt.Sprintf("'%v' in parents", parent),
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

	// Save in cache
	for _, f := range files {
		self.caches[fileId(f.Id)] = &fileEntry{
			file:    f,
			absPath: "",
		}
	}

	return files[0]
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
