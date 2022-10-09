package lineio

import "errors"

var ErrEndOfTail error = errors.New("end of tail")
var ErrLineTooLong error = errors.New("a line doesnot fit in the buffer")

type Line struct {
	// No is the number of current line, starting from 1
	No int
	// LineStart is the start offset of current line in the File, in number of bytes, staring from 0
	LineStart int64
	// LineEnding is the ending offset of current line in the File, in number of bytes, staring from 0, pointing at current \n
	LineEnding int64
	// Raw holds the line content, excluding \n
	Raw []byte
}
