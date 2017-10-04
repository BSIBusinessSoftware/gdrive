package drive

import (
	"fmt"
	"io"
	"math"
	"strings"
	"text/tabwriter"

	drive "google.golang.org/api/drive/v3"
)

const SEP = "/"

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
		name, err := printer.PathFinder.getAbsPath(file, SEP)
		if err != nil {
			return err
		}
		absPath = name
	}
	fmt.Fprintf(w, "+ %v:\n", absPath)

	listArgs := listAllFilesArgs{
		query:     fmt.Sprintf("trashed = false and 'me' in owners and '%v' in parents", file.Id),
		fields:    nil,
		sortOrder: "folder, name",
		maxFiles:  math.MaxInt64,
	}

	files, err := printer.Drive.listAllFiles(listArgs)
	if err != nil {
		return fmt.Errorf("Failed to list files: %s", err)
	}

	type directory struct {
		f        *drive.File
		fullpath string
	}
	var directories []directory

	for _, f := range files {
		fullpath := strings.Join([]string{absPath, f.Name}, SEP)
		if isDir(f) {
			directories = append(directories, directory{f, fullpath})
		}

		term := ""
		if isDir(f) {
			term = SEP
		}
		fmt.Fprintf(w, "%v%v\n", fullpath, term)
	}

	if printer.Recursive {
		fmt.Fprintf(w, "\n")
		for _, d := range directories {
			printer.Print(d.f, d.fullpath)
		}
	}

	return nil
}