package bulkio

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

	offset := t.StartingByte
	lineno := t.StartingLine - 1

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, err := t.fd.Seek(0, io.SeekCurrent)
			if err != nil {
				return err
			}

			n, err := io.ReadFull(t.fd, t.buf)
			if n > 0 {
				var lines []Line
				var lastLineEnding int = -1 // track lines within buf
				var lastLineNo int = 1
				for i := 0; i < n; i++ {
					if t.buf[i] == '\n' {
						line := Line{
							No:         lineno + lastLineNo,
							LineEnding: offset + int64(i),
							Raw:        t.buf[lastLineEnding+1 : i], // remove \n
						}
						lines = append(lines, line)
						lastLineEnding = i
						lastLineNo++
					}
				}

				offset += int64(lastLineEnding) + 1 // last line start
				lineno += len(lines)
				_, err := t.fd.Seek(offset, io.SeekStart) // unread last partial line
				if err != nil {
					return err
				}

				if len(lines) == 0 {
					time.Sleep(backoff)
					continue
				}

				if err := consume(lines); err != nil {
					if errors.Is(err, ErrEndOfTail) {
						return nil
					} else {
						return err
					}
				}
			} else {
				if err != io.EOF {
					return err
				}

				time.Sleep(backoff)
			}
		}
	}
}
