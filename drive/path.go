package drive

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

func (self *Drive) newPathfinder() *remotePathfinder {
	return &remotePathfinder{
		service: self.service.Files,
		files:   make(map[string]*drive.File),
	}
}

type remotePathfinder struct {
	service *drive.FilesService
	files   map[string]*drive.File
}

func (self *remotePathfinder) absPath(f *drive.File) (string, error) {
	name := f.Name

	if len(f.Parents) == 0 {
		return name, nil
	}

	var path []string

	for {
		parent, err := self.getParent(f.Parents[0])
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

func (self *remotePathfinder) getParent(id string) (*drive.File, error) {
	// Check cache
	if f, ok := self.files[id]; ok {
		return f, nil
	}

	// Fetch file from drive
	f, err := self.service.Get(id).Fields("id", "name", "parents").Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to get file: %s", err)
	}

	// Save in cache
	self.files[f.Id] = f

	return f, nil
}

type driveIdResolver struct {
	service *drive.FilesService
}

func (drive *Drive) newIdResolver() *driveIdResolver {
	return &driveIdResolver{
		service: drive.service.Files,
	}
}

func (self *driveIdResolver) getFileId(abspath string) (string, error) {
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

func (self *driveIdResolver) secureFileId(expr string) string {
	if strings.Contains(expr, "/") {
		id, err := self.getFileId(expr)
		if err == nil {
			return id
		}
	}
	return expr
}

func (self *driveIdResolver) queryEntryByName(name string, parent string) *drive.File {
	conditions := []string{
		"trashed = false",
		fmt.Sprintf("name = '%v'", name),
		fmt.Sprintf("'%v' in parents", parent),
	}
	query := strings.Join(conditions, " and ")
	fields := []googleapi.Field{"nextPageToken", "files(id,name,parents)"}

	var files []*drive.File
	self.service.List().Q(query).Fields(fields...).Pages(context.TODO(), func(fl *drive.FileList) error {
		files = append(files, fl.Files...)
		return nil
	})

	if len(files) == 0 {
		return nil
	}

	return files[0]
}
