package bulkio_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/JackKCWong/go-bulkio"
	. "github.com/onsi/gomega"
)

func TestSmokeTailN(t *testing.T) {
	g := NewGomegaWithT(t)

	doc := []byte("hello\nworld\nbye\n")
	rd := bytes.NewReader(doc)

	tailer := bulkio.NewTailer(rd, 11)
	// ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	// defer cancel()

	var lines []bulkio.Line
	var count int
	err := tailer.TailN(context.Background(), 100*time.Millisecond, func(batch []bulkio.Line) error {
		for i := range batch {
			lines = append(lines, batch[i].Copy())
		}

		count++
		if count == 2 {
			return bulkio.ErrEndOfTail
		}

		return nil
	})

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(len(lines)).Should(Equal(3))
	g.Expect(lines[0].No).Should(Equal(1))
	g.Expect(lines[0].Raw).Should(BeEquivalentTo("hello"))
	g.Expect(lines[0].LineEnding).Should(BeEquivalentTo(5))
	g.Expect(doc[0:lines[0].LineEnding]).Should(BeEquivalentTo("hello"))

	g.Expect(lines[1].No).Should(Equal(2))
	g.Expect(lines[1].Raw).Should(BeEquivalentTo("world"))
	g.Expect(lines[1].LineEnding).Should(BeEquivalentTo(11))
	g.Expect(doc[lines[0].LineEnding+1 : lines[1].LineEnding]).Should(BeEquivalentTo("world"))

	g.Expect(lines[2].No).Should(Equal(3))
	g.Expect(lines[2].Raw).Should(BeEquivalentTo("bye"))
	g.Expect(lines[2].LineEnding).Should(BeEquivalentTo(15))
	g.Expect(doc[lines[1].LineEnding+1 : lines[2].LineEnding]).Should(BeEquivalentTo("bye"))
}
