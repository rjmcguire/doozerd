package main

import (
	"flag"
	"fmt"
	"github.com/ha/doozerd/persistence"
	"io"
	"os"
	"path"
)

var (
	f        = flag.Bool("f", false, "try to fix a broken journal")
	progname = path.Base(os.Args[0])
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [options] journal", progname)
	flag.PrintDefaults()
	os.Exit(1)
}

func logf(format string, args ...interface{}) {
	fmt.Fprint(os.Stderr, "doozerp: ")
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func log(args ...interface{}) { logf("%v", args...) }

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}

	file := flag.Arg(0)
	j, err := persistence.NewJournal(file)
	if err != nil {
		panic(err)
	}
	for {
		m, err := j.ReadMutation()
		if err == io.EOF {
			break
		}
		if err != nil {
			logf("bad journal file: %v", err)
			if *f {
				err = j.Fsck()
				if err != nil {
					log("can't fix journal")
					os.Exit(2)
				}
				log("journal successfully fixed")
				continue
			} else {
				return
			}
		}
		fmt.Println(m)
	}
}
