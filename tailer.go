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

func (t *Tailer) Tail(ctx context.Context, backoff time.Duration, consume func([]Line) error) error {
	_, err := t.fd.Seek(t.StartingByte, io.SeekStart)
	if err != nil {
		return err
	}

	fileDataStart := t.StartingByte
	lineno := t.StartingLine - 1
	bufDataWriteIdx := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := t.fd.Read(t.buf[bufDataWriteIdx:])
			if n > 0 {
				bufDataLen := bufDataWriteIdx + n
				var lines []Line
				var lastLineEnding int = -1 // track lines within buf
				var lastLineNo int = 1
				for i := 0; i < bufDataLen; i++ {
					if t.buf[i] == '\n' {
						line := Line{
							No:         lineno + lastLineNo,
							LineStart:  fileDataStart,
							LineEnding: fileDataStart + int64(i),
							Raw:        t.buf[lastLineEnding+1 : i], // exclude \n
						}
						lines = append(lines, line)
						lastLineEnding = i
						lastLineNo++
					}
				}

				if len(lines) > 0 {
					if err := consume(lines); err != nil {
						if errors.Is(err, ErrEndOfTail) {
							return nil
						} else {
							return err
						}
					}
				} else {
					// no \n found and the buffer is filled.
					if bufDataLen == len(t.buf) {
						return ErrLineTooLong
					}
				}

				fileDataStart += int64(lastLineEnding) + 1
				lineno += len(lines)
				if lastLineEnding < bufDataLen-1 {
					if lastLineEnding > -1 {
						// a line is found so part of the buffer is consumed
						n = copy(t.buf, t.buf[lastLineEnding+1:bufDataLen])
						bufDataWriteIdx = n
					} else {
						// no line found so no buffer is consumed
						bufDataWriteIdx = bufDataLen
					}
				} else {
					// lastLineEnding == bufDataLen - 1
					// which means the whole buffer is consumed.
					bufDataWriteIdx = 0
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
