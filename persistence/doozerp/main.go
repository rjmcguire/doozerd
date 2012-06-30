// doozerp is a persistence client for doozerd.  It is implemented as
// a regular doozer client that connects to a cluster, monitors I/O and
// writes mutations to pesistent medium.  To signal its users, doozerp
// maintains a clone of the namespace in /ctl/persistence/<n>/.  A mutation
// in the mirrored tree signals the succesful logging of the associated
// mutation to disk.
package main

import (
	"flag"
	"fmt"
	"github.com/4ad/doozer"
	"github.com/4ad/doozerd/persistence"
	"os"
	"strconv"
)

var (
	buri	= flag.String("b", "", "the DzNS uri")
	f	= flag.Bool("f", false, "try to fix a broken journal")
	j	= flag.String("j", "journal", "file to log mutations")
	r	= flag.Bool("r", false, "restore from file")
	uri	= flag.String("a", "doozer:?ca=127.0.0.1:8046", "the address to bind to")
	v	= flag.Bool("v", false, "print each mutation on stdout")
)

var (
	conn	*doozer.Conn		// connection to the cluster.
	id	= 0			// client id.
	journal	*persistence.Journal	// journal to log to.
	notify	= make(chan *mutation)	// ack. write operation.
	store	= make(chan *mutation)	// save to disk.
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: doozerp [options]")
	flag.PrintDefaults()
	os.Exit(1)
}

func logf(format string, args ...interface{}) {
	fmt.Fprint(os.Stderr, "doozerp: ")
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func log(args ...interface{})	{ logf("%v", args...) }

func fatalf(format string, args ...interface{}) {
	logf(format, args...)
	os.Exit(2)
}

func fatal(args ...interface{})	{ fatalf("%v", args...) }

// dial connects to the server.
func dial() {
	var err error
	conn, err = doozer.DialUri(*uri, *buri)
	if err != nil {
		fatal(err)
	}
	setid()
}

// setid determines and sets the client id.
func setid() {
	body, rev, err := conn.Get("/ctl/persistence/id", nil)
	id, _ = strconv.Atoi(string(body))
	id++
	body = []byte(fmt.Sprint(id))
	_, err = conn.Set("/ctl/persistence/id", rev, body)
	if err != nil {
		fatal(err)
	}
}

// sets ephemeral key for other clients
func seteph() {
	_, rev, _ := conn.Get("/eph", nil)
	_, err := conn.Set("/eph", rev, []byte("doozerp " + fmt.Sprint(id)))
	if err != nil {
		fatal(err)
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()

	var err error
	journal, err = persistence.NewJournal(*j)
	if err != nil {
		fatal(err)
	}

	dial()
	if *r {
		restore()
	}
	seteph()
				
	go Store()
	go Notify()
	monitor()
}
