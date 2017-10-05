package drive

import (
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
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

//noinspection GoReceiverNames
func (self *Drive) ListDirectory(args ListDirectoryArgs) (err error) {
	args.normalize(self)

	printer := NewDirectoryPrinter(self, &args)
	printer.Print(args.Id)
	return
}

type DirectoryPrinter struct {
	Drive      *Drive
	PathFinder *remotePathFinder
	Out        io.Writer
	Recursive  bool
	ShowDoc    bool
}

func NewDirectoryPrinter(drive *Drive, args *ListDirectoryArgs) *DirectoryPrinter {
	return &DirectoryPrinter{
		Drive:      drive,
		PathFinder: drive.newPathFinder(),
		Out:        args.Out,
		Recursive:  args.Recursive,
		ShowDoc:    args.ShowDoc,
	}
}

func (printer *DirectoryPrinter) Print(id string) error {
	f, err := printer.PathFinder.getFile(id)
	if err != nil {
		return err
	}
	if isDir(f) {
		printer.printDirectory(f, "")
	} else {

	}
	return nil
}

func (printer *DirectoryPrinter) printDirectory(file *drive.File, fullPath string) error {

	if len(fullPath) == 0 {
		name, err := printer.PathFinder.getAbsPath(file)
		if err != nil {
			return err
		}
		fullPath = name
	}
	fmt.Fprintf(printer.Out, "+ %v:\n", fullPath)

	files, err := printer.Drive.listAllFiles(listAllFilesArgs{
		query:     fmt.Sprintf("trashed = false and 'me' in owners and '%v' in parents", file.Id),
		sortOrder: "folder, name",
	})
	if err != nil {
		return fmt.Errorf("failed to list files: %s", err)
	}

	type directory struct {
		file     *drive.File
		fullPath string
	}

	var directories []directory
	for _, f := range files {
		if isDoc(f) && !printer.ShowDoc {
			continue
		}

		fullPath := printer.PathFinder.JoinPath(fullPath, f.Name)
		if isDir(f) {
			directories = append(directories, directory{f, fullPath})
		}
		printer.printEntry(f, fullPath)
	}

	if printer.Recursive {
		fmt.Fprint(printer.Out, "\n")
		for _, d := range directories {
			printer.printDirectory(d.file, d.fullPath)
		}
	}

	return nil
}

func (printer *DirectoryPrinter) printEntry(f *drive.File, fullPath string) {

	term := ""
	if isDir(f) {
		term = RemotePathSep
	}
	fmt.Fprintf(printer.Out, "%v%v\n", fullPath, term)
}
