package drive

import (
	"fmt"
	"io"
	"math"
	"strings"
	"text/tabwriter"

	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

type ListDirectoryArgs struct {
	Out       io.Writer
	Id        string
	Recursive bool
}

func (args *ListDirectoryArgs) normalize(drive *Drive) {
	resolver := drive.newIdResolver()
	args.Id = resolver.secureFileId(args.Id)
}

func (self *Drive) ListDirectory(args ListDirectoryArgs) (err error) {
	args.normalize(self)

	f, err := self.newPathfinder().getFile(args.Id)
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}
	if isDir(f) {
		printer := NewDirectoryPrinter(self, args)
		printer.Print(f, "")
	}
	return
}

type DirectoryPrinter struct {
	Out        io.Writer
	Drive      *Drive
	PathFinder *remotePathfinder
	Recursive  bool
}

func NewDirectoryPrinter(drive *Drive, args ListDirectoryArgs) *DirectoryPrinter {
	return &DirectoryPrinter{
		Out:        args.Out,
		Drive:      drive,
		PathFinder: drive.newPathfinder(),
		Recursive:  args.Recursive,
	}
}

func (printer *DirectoryPrinter) Print(file *drive.File, absPath string) error {
	w := new(tabwriter.Writer)
	w.Init(printer.Out, 0, 0, 3, ' ', 0)

	if len(absPath) == 0 {
		name, err := printer.PathFinder.getAbsPath(file, "/")
		if err != nil {
			return err
		}
		absPath = name
	}
	fmt.Fprintf(w, "+ %v:\n", absPath)

	listArgs := listAllFilesArgs{
		query:    fmt.Sprintf("trashed = false and 'me' in owners and '%v' in parents", file.Id),
		fields:   []googleapi.Field{"nextPageToken", "files(id,name,md5Checksum,mimeType,size,createdTime,parents)"},
		maxFiles: math.MaxInt64,
	}

	files, err := printer.Drive.listAllFiles(listArgs)
	if err != nil {
		return fmt.Errorf("Failed to list files: %s", err)
	}

	for _, f := range files {
		fmt.Fprintf(w, "%v\n", strings.Join([]string{absPath, f.Name}, "/"))
	}
	fmt.Fprintf(w, "\n")

	if printer.Recursive {
		for _, f := range files {
			if isDir(f) {
				printer.Print(f, strings.Join([]string{absPath, f.Name}, "/"))
			}
		}
	}

	return nil
}
