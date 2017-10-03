package drive

import (
	"fmt"
	"io"
)

type IdArgs struct {
	Out     io.Writer
	AbsPath string
	Error   bool
}

func (self *Drive) Id(args IdArgs) error {
	//fmt.Fprintf(args.Out, "AbsPath='%v', Error='%v'\n", args.AbsPath, args.Error)

	resolver := self.newIdResolver()
	Id, err := resolver.getFileID(args.AbsPath)
	if err != nil && args.Error == true {
		return err
	}
	fmt.Print(Id)
	return nil
}
