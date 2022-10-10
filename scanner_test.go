package lineio_test

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/JackKCWong/lineio"
	. "github.com/onsi/gomega"
)

func TestSmokeScan(t *testing.T) {
	g := NewGomegaWithT(t)

	fd, err := ioutil.TempFile(os.TempDir(), "TestSmokeTail")
	g.Expect(err).ShouldNot(HaveOccurred())
	defer fd.Close()

	log.Printf("%s created", fd.Name())

	buf := make([]byte, 6)
	rd, err := os.Open(fd.Name())
	g.Expect(err).ShouldNot(HaveOccurred())
	defer rd.Close()

	scanner := lineio.NewScanner(rd, buf)

	fd.WriteString("hi")
	fd.Sync()
	g.Expect(scanner.Scan()).Should(Equal(false))
	g.Expect(scanner.Err()).Should(Equal(io.EOF))

	fd.WriteString("\n")
	fd.Sync()
	ok, err := scanner.ResumeFromEOF()
	g.Expect(ok).Should(BeTrue())
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(scanner.Scan()).Should(Equal(true))
	g.Expect(scanner.Line()).Should(Equal(lineio.Line{
		No:         1,
		LineStart:  0,
		LineEnding: 2,
		Raw:        []byte("hi"),
	}))
	g.Expect(err).ShouldNot(HaveOccurred())

	fd.WriteString("world")
	fd.Sync()
	g.Expect(scanner.Scan()).Should(Equal(false))
	g.Expect(scanner.Err()).Should(Equal(io.EOF))

	fd.WriteString("\n")
	fd.Sync()
	ok, err = scanner.ResumeFromEOF()
	g.Expect(ok).Should(BeTrue())
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(scanner.Scan()).Should(Equal(true))
	g.Expect(scanner.Line()).Should(Equal(lineio.Line{
		No:         2,
		LineStart:  3,
		LineEnding: 8,
		Raw:        []byte("world"),
	}))
	g.Expect(scanner.Err()).ShouldNot(HaveOccurred())

	fd.WriteString("bye\nsu")
	fd.Sync()
	ok, err = scanner.ResumeFromEOF()
	g.Expect(ok).Should(BeTrue())
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(scanner.Scan()).Should(Equal(true))
	g.Expect(scanner.Line()).Should(Equal(lineio.Line{
		No:         3,
		LineStart:  9,
		LineEnding: 12,
		Raw:        []byte("bye"),
	}))
	g.Expect(scanner.Err()).ShouldNot(HaveOccurred())

	fd.WriteString("per looooooooooong line\n")
	fd.Sync()
	ok, err = scanner.ResumeFromEOF()
	g.Expect(ok).Should(BeTrue())
	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(scanner.Scan()).Should(Equal(false))
	g.Expect(scanner.Err()).Should(Equal(lineio.ErrLineTooLong))
	g.Expect(buf).Should(BeEquivalentTo("super "))
}
