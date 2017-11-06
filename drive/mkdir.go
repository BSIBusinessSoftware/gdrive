package drive

import (
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
)

const DirectoryMimeType = "application/vnd.google-apps.folder"

type MkdirArgs struct {
	Out         io.Writer
	Name        string
	Description string
	Parents     []string
}

func (args *MkdirArgs) normalize(drive *Drive) {
	if len(args.Parents) > 0 {
		var ids []string
		finder := drive.newPathFinder()
		for _, parent := range args.Parents {
			id := finder.SecureFileId(parent)
			ids = append(ids, id)
		}
		args.Parents = ids
	}
}

func (self *Drive) Mkdir(args MkdirArgs) error {
	args.normalize(self)

	f, err := self.mkdir(args)
	if err != nil {
		return err
	}
	fmt.Fprintf(args.Out, "Directory %s created\n", f.Id)
	return nil
}

func (self *Drive) mkdir(args MkdirArgs) (*drive.File, error) {
	dstFile := &drive.File{
		Name:        args.Name,
		Description: args.Description,
		MimeType:    DirectoryMimeType,
	}

	// Set parent folders
	dstFile.Parents = args.Parents

	// Create directory
	f, err := self.service.Files.Create(dstFile).Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to create directory: %s", err)
	}

	return f, nil
}
