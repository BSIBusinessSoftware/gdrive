package drive

import (
	"fmt"
	"path/filepath"
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
		files:   make(map[string]*drive.File),
	}
}

type remotePathFinder struct {
	service *drive.FilesService
	files   map[string]*drive.File
}

func (self *remotePathFinder) absPath(f *drive.File) (string, error) {
	name := f.Name

	if len(f.Parents) == 0 {
		return name, nil
	}

	var path []string

	for {
		parent, err := self.getFile(f.Parents[0])
		if err != nil {
			return "", err
		}

		// Stop when we find the root dir
		if len(parent.Parents) == 0 {
			break
		}

		path = append([]string{parent.Name}, path...)
		f = parent
	}

	path = append(path, name)
	return filepath.Join(path...), nil
}

func (self *remotePathFinder) getAbsPath(f *drive.File) (string, error) {

	if len(f.Parents) == 0 {
		return RemotePathSep, nil
	}

	path, err := self.absPath(f)
	if err != nil {
		return "", err
	}
	items := strings.Split(path, string(filepath.Separator))
	return RemotePathSep + strings.Join(items, RemotePathSep), nil
}

func (self *remotePathFinder) JoinPath(pathes ...string) string {
	items := []string{}
	for _, path := range pathes {
		path = strings.TrimSuffix(path, RemotePathSep)
		items = append(items, path)
	}
	return strings.Join(items, RemotePathSep)
}

func (self *remotePathFinder) getFile(id string) (*drive.File, error) {
	// Check cache
	if f, ok := self.files[id]; ok {
		return f, nil
	}

	// Fetch file from drive
	f, err := self.service.Get(id).Fields(defaultGetFields...).Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to get file: %s", err)
	}

	// Save in cache
	self.files[f.Id] = f

	return f, nil
}

func (self *remotePathFinder) getFileId(abspath string) (string, error) {
	if !strings.HasPrefix(abspath, "/") {
		return "", fmt.Errorf("'%s' is not absolute path", abspath)
	}

	abspath = strings.Trim(abspath, "/")
	if abspath == "" {
		return "root", nil
	}
	pathes := strings.Split(abspath, "/")
	var parent = "root"
	for _, path := range pathes {
		entry := self.queryEntryByName(path, parent)
		if entry == nil {
			return "", fmt.Errorf("path not found: '%v'", abspath)
		}
		parent = entry.Id
	}
	return parent, nil
}

func (self *remotePathFinder) secureFileId(expr string) string {
	if strings.Contains(expr, "/") {
		id, err := self.getFileId(expr)
		if err == nil {
			return id
		}
	}
	return expr
}

func (self *remotePathFinder) queryEntryByName(name string, parent string) *drive.File {
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
