package drive

import (
	"fmt"
	"io"
	"strings"
)

type IdArgs struct {
	Out     io.Writer
	AbsPath string
	Error   bool
}

func (self *Drive) Id(args IdArgs) error {
	fmt.Fprintf(args.Out, "AbsPath='%v', Error='%v'\n", args.AbsPath, args.Error)

	if !strings.HasPrefix(args.AbsPath, "/") {
		return fmt.Errorf("'%s' is not absolute path", args.AbsPath)
	}

	return nil
}
