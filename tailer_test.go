package lineio_test

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/JackKCWong/lineio"
	. "github.com/onsi/gomega"
)

func TestSmokeTail(t *testing.T) {
	g := NewGomegaWithT(t)

	fd, err := ioutil.TempFile(os.TempDir(), "TestSmokeTail")
	g.Expect(err).ShouldNot(HaveOccurred())

	log.Printf("%s created", fd.Name())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		fd.WriteString("hi")
		fd.Sync()
		time.Sleep(100*time.Millisecond)
		fd.WriteString("\n")
		fd.Sync()
		time.Sleep(100*time.Millisecond)
		fd.WriteString("world")
		fd.Sync()
		fd.WriteString("\n")
		fd.Sync()
		time.Sleep(100*time.Millisecond)
		fd.WriteString("bye\n")
		fd.Sync()
		fd.Close()
	}()

	doc := []byte("hi\nworld\nbye\n")
	buf := make([]byte, 6)
	rd, err := os.Open(fd.Name())
	g.Expect(err).ShouldNot(HaveOccurred())
	defer rd.Close()

	tailer := lineio.NewTailer(rd, buf)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	wg.Wait()

	var lines []lineio.Line
	err = tailer.Tail(ctx, 50*time.Millisecond, func(batch []lineio.Line) error {
		log.Printf("get lines: %+v", batch)
		for i := range batch {
			lines = append(lines, batch[i].Copy())
		}

		if len(lines) >= 3 {
			return lineio.ErrEndOfTail
		}

		return nil
	})

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(len(lines)).Should(Equal(3))
	g.Expect(lines[0].No).Should(Equal(1))
	g.Expect(lines[0].Raw).Should(BeEquivalentTo("hi"))
	g.Expect(lines[0].LineEnding).Should(BeEquivalentTo(2))
	g.Expect(doc[0:lines[0].LineEnding]).Should(BeEquivalentTo("hi"))

	g.Expect(lines[1].No).Should(Equal(2))
	g.Expect(lines[1].Raw).Should(BeEquivalentTo("world"))
	g.Expect(lines[1].LineEnding).Should(BeEquivalentTo(8))
	g.Expect(doc[lines[0].LineEnding+1 : lines[1].LineEnding]).Should(BeEquivalentTo("world"))

	g.Expect(lines[2].No).Should(Equal(3))
	g.Expect(lines[2].Raw).Should(BeEquivalentTo("bye"))
	g.Expect(lines[2].LineEnding).Should(BeEquivalentTo(12))
	g.Expect(doc[lines[1].LineEnding+1 : lines[2].LineEnding]).Should(BeEquivalentTo("bye"))
}
