// Package doozerp_testing tests the doozerp command and the doozerl library.
package doozerp_testing

import (
	"errors"
	"fmt"
	"github.com/4ad/doozerd/persistence"
	"github.com/4ad/doozerd/persistence/doozerl"
	"github.com/4ad/doozerd/store"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

var testData = []string{
	"",
	"I am a programmer",
	"When in doubt, use brute force",
	"We have persistant objects, they're called files",
	"If you want to go somewhere, goto is the best way to get there",
	"A well installed microcode bug will be almost impossible to detect",
	"In college, before video games, we would amuse ourselves by posing programming exercises",
}

// decode decodes a mutation into an k, v pair to check agains testData,
// from ../../../store/store.go:/decode.
func decode(mut string) (k int, v string, err error) {
	cm := strings.SplitN(mut, ":", 2)

	if len(cm) != 2 {
		err = errors.New("bad mutation")
		return
	}

	kv := strings.SplitN(cm[1], "=", 2)
	k, err = strconv.Atoi(kv[0][5:])
	if err != nil {
		return
	}
	if len(kv) == 2 {
		v = kv[1]
	}
	return
}

type Cluster struct {
	conn		*doozerl.Conn
	doozerd		*exec.Cmd
	doozerp		*exec.Cmd
	j		string
	doozerdIsDead	bool
	doozerpIsDead	bool
}

func NewCluster(t *testing.T, doozerpArgs ...string) (c *Cluster) {
	c = new(Cluster)
	c.doozerd = exec.Command("doozerd", "-l=127.0.0.1:19999", "-w=false")
	err := c.doozerd.Start()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		c.doozerd.Wait()
		c.doozerdIsDead = true
	}()
	time.Sleep(100 * time.Millisecond)
	if c.doozerdIsDead {
		t.Fatal("doozerd died prematurely")
	}

	c.conn, err = doozerl.DialUri("doozer:?ca=127.0.0.1:19999", "")
	if err != nil {
		c.doozerd.Process.Kill()
		t.Fatal(err)
	}

	f, err := ioutil.TempFile("", "j")
	if err != nil {
		c.conn.Close()
		c.doozerd.Process.Kill()
		t.Fatal(err)
	}
	c.j = f.Name()
	f.Close()
	args := []string{
		"-a=doozer:?ca=127.0.0.1:19999",
		"-j=" + c.j,
	}
	args = append(args, doozerpArgs...)
	c.doozerp = exec.Command("doozerp", args...)
	err = c.doozerp.Start()
	if err != nil {
		c.conn.Close()
		c.doozerd.Process.Kill()
		t.Fatal(err)
	}
	go func() {
		c.doozerp.Wait()
		c.doozerpIsDead = true
	}()
	time.Sleep(100 * time.Millisecond)
	if c.doozerpIsDead {
		t.Fatal("doozerp died prematurely")
	}

	return
}

func (c *Cluster) Close() {
	c.conn.Close()
	c.doozerd.Process.Kill()
	if !c.doozerpIsDead {
		c.doozerp.Wait()
	}
	os.Remove(c.j)
}

func TestNewCluster(t *testing.T) {
	c := NewCluster(t)
	defer c.Close()
}

func TestFix(t *testing.T) {
	f, err := ioutil.TempFile("", "j")
	if err != nil {
		t.Fatal(err)
	}
	j := f.Name()
	f.Close()
	defer os.Remove(j)

	journal, err := persistence.NewJournal(j)
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range testData {
		m := store.MustEncodeSet("/ken/"+strconv.Itoa(k), v, int64(k))
		journal.WriteMutation(m)
	}
	for k, v := range testData {
		m := store.MustEncodeSet("/ken/"+strconv.Itoa(k), v, int64(k))
		journal.WriteMutation(m)
	}
	journal.Close()

	fi, err := os.Stat(j)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Truncate(j, fi.Size()/2+1)
	if err != nil {
		t.Fatal(err)
	}

	c := NewCluster(t, fmt.Sprintf("-j=%s", j), "-r", "-f")
	defer c.Close()
	for k, v := range testData {
		v1, _, err := c.conn.Get("/ken/"+strconv.Itoa(k), nil)
		if err != nil {
			t.Fatal(err)
		}
		v2 := string(v1)
		if v != v2 {
			t.Fatalf("restored data is not what is expected: %s != %s", v, v2)
		}
	}
}

func TestNotify(t *testing.T) {
	c := NewCluster(t)
	defer c.Close()

	for k, v := range testData {
		c.conn.Set("/ken/"+strconv.Itoa(k), -1, []byte(v))
	}
	var notified bool
	go func() {
		time.Sleep(5000 * time.Millisecond)
		if !notified {
			t.Fatal("no save notification")
		}
	}()
	time.Sleep(10 * time.Millisecond)
	for k, _ := range testData {
		ev, err := c.conn.Wait("/ctl/persistence/1/ken/"+strconv.Itoa(k), -1)
		if err != nil {
			t.Fatal(err)
		}
		v, _, err := c.conn.Get("/ctl/persistence/1/ken/"+strconv.Itoa(k), &ev.Rev)
		rev, err := strconv.Atoi(string(v))
		if err != nil {
			t.Fatal(err)
		}
		if int64(rev) > ev.Rev {
			t.Fatalf("wrong notification: %d != %d", rev, ev.Rev)
		}
	}
	notified = true
}

func TestRestore(t *testing.T) {
	f, err := ioutil.TempFile("", "j")
	if err != nil {
		t.Fatal(err)
	}
	j := f.Name()
	f.Close()
	defer os.Remove(j)

	journal, err := persistence.NewJournal(j)
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range testData {
		m := store.MustEncodeSet("/ken/"+strconv.Itoa(k), v, int64(k))
		journal.WriteMutation(m)
	}
	journal.Close()

	c := NewCluster(t, fmt.Sprintf("-j=%s", j), "-r")
	defer c.Close()
	for k, v := range testData {
		v1, _, err := c.conn.Get("/ken/"+strconv.Itoa(k), nil)
		if err != nil {
			t.Fatal(err)
		}
		v2 := string(v1)
		if v != v2 {
			t.Fatalf("restored data is not what is expected: %s != %s", v, v2)
		}
	}
}

func TestSave(t *testing.T) {
	c := NewCluster(t)
	defer c.Close()

	for k, v := range testData {
		c.conn.Set("/ken/"+strconv.Itoa(k), -1, []byte(v))
	}
	j, err := persistence.NewJournal(c.j)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	for k, v := range testData {
		m, err := j.ReadMutation()
		if err != nil {
			t.Fatalf("bad journal file: %v", err)
		}
		k1, v1, err := decode(m)
		if err != nil {
			t.Fatalf("bad journal file: %v", err)
		}
		if k != k1 {
			t.Fatalf("bad journal file: %s != %s", k1, k)
		}
		if v != v1 {
			t.Fatalf("bad journal file: %s != %s", v1, v)
		}
	}
	_, err = j.ReadMutation()
	if err != io.EOF {
		t.Fatal("bad journal file: expected EOF")
	}
}
