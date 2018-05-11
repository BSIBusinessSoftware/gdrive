package drive

import (
	"fmt"
	"io"
)

type DeleteRevisionArgs struct {
	Out        io.Writer
	FileId     string
	RevisionId string
}

func (args *DeleteRevisionArgs) normalize(drive *Drive) {
	finder := drive.newPathFinder()
	args.FileId = finder.SecureFileId(args.FileId)
}

func (self *Drive) DeleteRevision(args DeleteRevisionArgs) (err error) {
	args.normalize(self)

	rev, err := self.service.Revisions.Get(args.FileId, args.RevisionId).Fields("originalFilename").Do()
	if err != nil {
		return fmt.Errorf("Failed to get revision: %s", err)
	}

	if rev.OriginalFilename == "" {
		return fmt.Errorf("Deleting revisions for this file type is not supported")
	}

	err = self.service.Revisions.Delete(args.FileId, args.RevisionId).Do()
	if err != nil {
		return fmt.Errorf("Failed to delete revision: %s", err)
	}

	fmt.Fprintf(args.Out, "Deleted revision '%s'\n", args.RevisionId)
	return
}
