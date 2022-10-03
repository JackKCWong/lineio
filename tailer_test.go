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

	rd := bytes.NewReader([]byte("hello\nworld\nbye\n"))

	tailer := bulkio.NewTailer(rd, 1024)
	var lineCount int

	err := tailer.TailN(context.Background(), 100*time.Millisecond, func(lines []bulkio.Line) error {
		lineCount = len(lines)
		g.Expect(lineCount).Should(Equal(3))

		g.Expect(lines[0].No).Should(Equal(1))
		g.Expect(lines[0].Raw).Should(BeEquivalentTo("hello\n"))
		g.Expect(lines[0].Offset).Should(BeEquivalentTo(6))

		g.Expect(lines[1].No).Should(Equal(2))
		g.Expect(lines[1].Raw).Should(BeEquivalentTo("world\n"))
		g.Expect(lines[1].Offset).Should(BeEquivalentTo(12))

		g.Expect(lines[2].No).Should(Equal(3))
		g.Expect(lines[2].Raw).Should(BeEquivalentTo("bye\n"))
		g.Expect(lines[2].Offset).Should(BeEquivalentTo(16))
		return bulkio.ErrEndOfTail
	})

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(lineCount).Should(Equal(3)) // make sure lines are read.
}
