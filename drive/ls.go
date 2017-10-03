package drive

import (
	"fmt"
	"io"
	"math"

	"google.golang.org/api/googleapi"
)

type ListDirectoryArgs struct {
	Out         io.Writer
	Parent      string
	NameWidth   int64
	SortOrder   string
	SkipHeader  bool
	SizeInBytes bool
}

func (args *ListDirectoryArgs) normalize(drive *Drive) {
	resolver := drive.newIdResolver()
	args.Parent = resolver.secureFileId(args.Parent)
}

func (self *Drive) ListDirectory(args ListDirectoryArgs) (err error) {
	args.normalize(self)

	listArgs := listAllFilesArgs{
		query:     fmt.Sprintf("trashed = false and 'me' in owners and '%v' in parents", args.Parent),
		fields:    []googleapi.Field{"nextPageToken", "files(id,name,md5Checksum,mimeType,size,createdTime,parents)"},
		sortOrder: args.SortOrder,
		maxFiles:  math.MaxInt64,
	}

	files, err := self.listAllFiles(listArgs)
	if err != nil {
		return fmt.Errorf("Failed to list files: %s", err)
	}

	PrintFileList(PrintFileListArgs{
		Out:         args.Out,
		Files:       files,
		NameWidth:   int(args.NameWidth),
		SkipHeader:  args.SkipHeader,
		SizeInBytes: args.SizeInBytes,
	})

	return
}
