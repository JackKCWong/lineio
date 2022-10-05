package lineio 

import (
	"context"
	"errors"
	"io"
	"time"
)

type Tailer struct {
	fd           io.ReadSeeker
	buf          []byte
	StartingByte int64
	StartingLine int
}

type Line struct {
	// No is the number of current line, starting from 1
	No int
	// LineEnding is the ending offset of current line in the File, in number of bytes, staring from 0, pointing at current \n
	LineEnding int64
	// Raw holds the line content, excluding \n
	Raw []byte
}

func (l Line) Copy() Line {
	raw := make([]byte, len(l.Raw))
	copy(raw, l.Raw)
	return Line{
		No:         l.No,
		LineEnding: l.LineEnding,
		Raw:        raw,
	}
}

func NewTailer(fd io.ReadSeeker, buf []byte) *Tailer {
	return &Tailer{
		fd:           fd,
		buf:          buf,
		StartingByte: 0,
		StartingLine: 1,
	}
}

var ErrEndOfTail error = errors.New("end of tail")

func (t *Tailer) Tail(ctx context.Context, backoff time.Duration, consume func([]Line) error) error {
	_, err := t.fd.Seek(t.StartingByte, io.SeekStart)
	if err != nil {
		return err
	}

	fileDataStart := t.StartingByte
	lineno := t.StartingLine - 1
	bufDataStart := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := t.fd.Read(t.buf[bufDataStart:])
			if n > 0 {
				bufDataEnd := bufDataStart + n
				var lines []Line
				var lastLineEnding int = -1 // track lines within buf
				var lastLineNo int = 1
				for i := 0; i < bufDataEnd; i++ {
					if t.buf[i] == '\n' {
						line := Line{
							No:         lineno + lastLineNo,
							LineEnding: fileDataStart + int64(i),
							Raw:        t.buf[lastLineEnding+1 : i], // remove \n
						}
						lines = append(lines, line)
						lastLineEnding = i
						lastLineNo++
					}
				}

				fileDataStart += int64(lastLineEnding) + 1 // last line start
				lineno += len(lines)
				if len(lines) > 0 {
					if err := consume(lines); err != nil {
						if errors.Is(err, ErrEndOfTail) {
							return nil
						} else {
							return err
						}
					}
				}

				if lastLineEnding < bufDataEnd-1 {
					n := copy(t.buf, t.buf[lastLineEnding+1:bufDataEnd])
					bufDataStart = n
				} else {
					bufDataStart = 0
				}
			}

			if err == io.EOF {
				time.Sleep(backoff)
				_, err := t.fd.Seek(0, io.SeekCurrent) // reset EOF
				if err != nil {
					return err
				}
			}
		}
	}
}
