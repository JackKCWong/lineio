package bulkio

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"text/scanner"
	"time"
)

type Tailer struct {
	fd           io.ReadSeeker
	bufsize      int
	StartingByte int64
	StartingLine int
}

type Line struct {
	// No is the number of current line, starting from 1
	No int
	// Offset is the ending offset of current line, in number of bytes, staring from 0
	Offset int64
	// Raw holds the line content, including \n
	Raw []byte
}

func NewTailer(fd io.ReadSeeker, bufsize int) *Tailer {
	return &Tailer{
		fd:           fd,
		bufsize:      bufsize,
		StartingLine: 1,
	}
}

var ErrEndOfTail error = errors.New("end of tail")

func (t *Tailer) Tail(ctx context.Context, backoff time.Duration, consume func(Line) error) error {
	_, err := t.fd.Seek(t.StartingByte, io.SeekStart)
	if err != nil {
		return err
	}

	rd := bufio.NewReaderSize(t.fd, t.bufsize)
	offset := t.StartingByte
	lineno := t.StartingLine

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// reset EOF
			_, err := t.fd.Seek(0, io.SeekCurrent)
			if err != nil {
				return err
			}

			line, err := rd.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					_, err := t.fd.Seek(offset, io.SeekStart) // unread last partial line
					if err != nil {
						return err
					}

					time.Sleep(backoff)
					continue
				} else {
					return err
				}
			}

			l := Line{
				No:     lineno,
				Offset: offset,
				Raw:    line, // keep \n
			}

			if err := consume(l); err != nil {
				if errors.Is(err, ErrEndOfTail) {
					return nil
				} else {
					return err
				}
			}

			offset += int64(len(line) + 1)
			lineno += 1
		}
	}
}

func (t *Tailer) TailN(ctx context.Context, backoff time.Duration, consume func([]Line) error) error {
	_, err := t.fd.Seek(t.StartingByte, io.SeekStart)
	if err != nil {
		return err
	}

	buf := make([]byte, t.bufsize)
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

			n, err := io.ReadFull(t.fd, buf)
			if n > 0 {
				var s scanner.Scanner
				s.Init(bytes.NewReader(buf[0:n]))
				s.Whitespace ^= 1<<'\t' | 1<<'\n' // don't skip tabs and new lines

				var lines []Line
				var lastPosOffset int // track lines within buf
				for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
					if tok == '\n' {
						pos := s.Pos()
						line := Line{
							No:     lineno + (pos.Line - 1), // \n is counted as next line start
							Offset: offset + int64(pos.Offset),
							Raw:    buf[lastPosOffset:pos.Offset],
						}
						lines = append(lines, line)
						lastPosOffset = pos.Offset // Next line
					}
				}

				if len(lines) == 0 {
					_, err := t.fd.Seek(offset, io.SeekStart) // unread last partial line
					if err != nil {
						return err
					}

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
				offset += int64(lastPosOffset) // last line start
				lineno += len(lines)
			} else {
				if err != io.EOF {
					return err
				}

				time.Sleep(backoff)
			}
		}
	}
}
