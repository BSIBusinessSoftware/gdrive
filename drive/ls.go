package drive

import (
	"fmt"
	"io"
	"math"
	"text/tabwriter"

	drive "google.golang.org/api/drive/v3"
)

type ListDirectoryArgs struct {
	Out       io.Writer
	Id        string
	Recursive bool
	ShowDoc   bool
}

func (args *ListDirectoryArgs) normalize(drive *Drive) {
	finder := drive.newPathFinder()
	args.Id = finder.secureFileId(args.Id)
}

func (self *Drive) ListDirectory(args ListDirectoryArgs) (err error) {
	args.normalize(self)

	f, err := self.newPathFinder().getFile(args.Id)
	if err != nil {
		return fmt.Errorf("Failed to get file: %s", err)
	}
	if isDir(f) {
		printer := NewDirectoryPrinter(self, &args)
		printer.Print(f, "")
	}
	return
}

type DirectoryPrinter struct {
	Drive      *Drive
	PathFinder *remotePathFinder
	Args       *ListDirectoryArgs
}

func NewDirectoryPrinter(drive *Drive, args *ListDirectoryArgs) *DirectoryPrinter {
	return &DirectoryPrinter{
		Drive:      drive,
		PathFinder: drive.newPathFinder(),
		Args:       args,
	}
}

func (printer *DirectoryPrinter) Print(file *drive.File, absPath string) error {
	w := new(tabwriter.Writer)
	w.Init(printer.Args.Out, 0, 0, 3, ' ', 0)

	if len(absPath) == 0 {
		name, err := printer.PathFinder.getAbsPath(file)
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
		if isDoc(f) && !printer.Args.ShowDoc {
			continue
		}

		fullpath := printer.PathFinder.JoinPath(absPath, f.Name)
		if isDir(f) {
			directories = append(directories, directory{f, fullpath})
		}

		term := ""
		if isDir(f) {
			term = RemotePathSep
		}
		fmt.Fprintf(w, "%v%v\n", fullpath, term)
	}

	if printer.Args.Recursive {
		fmt.Fprintf(w, "\n")
		for _, d := range directories {
			printer.Print(d.f, d.fullpath)
		}
	}

	return nil
}
