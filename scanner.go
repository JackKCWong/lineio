package lineio

import (
	"io"
	"os"
)

type Scanner struct {
	fd           *os.File
	buf          []byte
	lineStartIdx int
	lineEndIdx   int
	dataLen      int
	err          error
	eof          error
	lastline     Line
}

type ScannerOption func(*Scanner)

func WithStartPos(startingByte int64, startingLine int) ScannerOption {
	return func(s *Scanner) {
		s.lastline.LineStart = startingByte
		s.lastline.No = startingLine
	}
}

func NewScanner(fd *os.File, buf []byte, opts ...ScannerOption) Scanner {
	s := Scanner{
		fd:           fd,
		buf:          buf,
		lineStartIdx: 0,
		lineEndIdx:   0,
		lastline: Line{
			LineEnding: -1,
		},
	}

	for i := range opts {
		opts[i](&s)
	}

	return s
}

func (s *Scanner) ResumeFromEOF() (bool, error) {
	if s.err != nil && s.err != io.EOF {
		return false, s.err
	}

	n, err := s.fd.Seek(0, io.SeekCurrent)
	if err != nil {
		return false, err
	}

	fi, err := s.fd.Stat()
	if err != nil {
		return false, err
	}

	if n < fi.Size() {
		s.err = nil
		s.eof = nil
		return true, nil
	}

	return false, nil
}

func (s *Scanner) Scan() bool {
	if s.err != nil {
		return false
	}

	if s.dataLen == len(s.buf) && s.lineStartIdx == s.dataLen {
		s.dataLen = 0
		s.lineStartIdx = 0
	}

	if s.dataLen < len(s.buf) && s.eof == nil {
		n, err := s.fd.Read(s.buf[s.dataLen:])
		if n > 0 {
			s.dataLen += n
		}

		if err != nil {
			if err == io.EOF {
				s.eof = err
				// continue to read from buffer
			} else {
				s.err = err
				return false
			}
		}
	}

	for i := s.lineStartIdx; i < s.dataLen; i++ {
		if s.buf[i] == '\n' {
			s.lineEndIdx = i
			linelen := s.lineEndIdx - s.lineStartIdx + 1
			s.lastline.No++
			s.lastline.LineStart = s.lastline.LineEnding + 1
			s.lastline.LineEnding += int64(linelen)
			s.lastline.Raw = s.buf[s.lineStartIdx:s.lineEndIdx]
			s.lineStartIdx = s.lineEndIdx + 1

			return true
		}
	}

	if s.dataLen == len(s.buf) && s.lineStartIdx == 0 {
		// the whole buffer does not contain a newline.
		s.err = ErrLineTooLong
		return false
	}

	if s.eof != nil {
		// both the buffer and file are consumed
		s.err = io.EOF
		return false
	}

	n := copy(s.buf, s.buf[s.lineStartIdx:s.dataLen])
	s.dataLen = n
	s.lineStartIdx = 0

	return s.Scan()
}

func (s Scanner) Line() Line {
	return s.lastline
}

func (s Scanner) Err() error {
	return s.err
}
