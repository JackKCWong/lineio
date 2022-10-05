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

	doc := []byte("hi\nworld\nbye\n")
	buf := make([]byte, 6)
	rd := bytes.NewReader(doc)

	tailer := bulkio.NewTailer(rd, buf)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var lines []bulkio.Line
	err := tailer.Tail(ctx, 100*time.Millisecond, func(batch []bulkio.Line) error {
		for i := range batch {
			lines = append(lines, batch[i].Copy())
		}

		if len(lines) >= 3 {
			return bulkio.ErrEndOfTail
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
