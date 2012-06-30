package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/4ad/doozerd/persistence"
	"os"
	"path"
)

var progname = path.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s journal", progname)
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}

	file := flag.Arg(0)
	j, err := persistence.NewJournal(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", progname, err)
		os.Exit(2)
	}
	r := bufio.NewReader(os.Stdin)
	for {
		line, prefix, err := r.ReadLine()
		if err != nil {
			return
		}
		for prefix {
			var l []byte
			l, prefix, _ = r.ReadLine()
			if err != nil {
				return
			}
			line = append(line, l...)
		}
		err = j.WriteMutation(string(line))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", progname, err)
			os.Exit(4)
		}
	}
}
