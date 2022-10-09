package lineio

import (
	"io"
	"os"
)

type Scanner struct {
	fd           *os.File
	buf          []byte
	lineStartIdx int64
	lineEndIdx   int64
	dataLen      int64
	err          error
	lastline     Line
}

func NewScanner(fd *os.File, buf []byte) Scanner {
	return Scanner{
		fd:           fd,
		buf:          buf,
		lineStartIdx: 0,
		lineEndIdx:   0,
		lastline: Line{
			LineEnding: -1,
		},
	}
}

func (s *Scanner) SetStart(startingByte int64, startingLine int) {
	s.lastline.LineStart = startingByte
	s.lastline.No = startingLine
}

func (s *Scanner) ResumeFromEOF() (int64, error) {
	return s.fd.Seek(0, io.SeekCurrent)
}

func (s *Scanner) Scan() bool {
	if s.dataLen == int64(len(s.buf)) && s.lineStartIdx == s.dataLen {
		s.dataLen = 0
		s.lineStartIdx = 0
	}

	n, err := s.fd.Read(s.buf[s.dataLen:])
	if n > 0 {
		s.dataLen += int64(n)
	} else if err != nil {
		s.err = err
		return false
	}

	if err != nil {
		s.err = err
	}

	for i := s.lineStartIdx; i < s.dataLen; i++ {
		if s.buf[i] == '\n' {
			s.lineEndIdx = i
			linelen := s.lineEndIdx - s.lineStartIdx + 1
			s.lastline.No++
			s.lastline.LineStart = s.lastline.LineEnding + 1
			s.lastline.LineEnding += linelen
			s.lastline.Raw = s.buf[s.lineStartIdx:s.lineEndIdx]
			s.lineStartIdx = s.lineEndIdx + 1

			return true
		}
	}

	if s.dataLen == int64(len(s.buf)) && s.lineStartIdx == 0 {
		// the whole buffer does not contain a newline.
		s.err = ErrLineTooLong
		return false
	}

	n = copy(s.buf, s.buf[s.lineStartIdx:s.dataLen])
	s.dataLen = int64(n)
	s.lineStartIdx = 0

	return s.Scan()
}

func (s Scanner) Line() Line {
	return s.lastline
}

func (s Scanner) Err() error {
	return s.err
}
