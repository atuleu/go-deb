package deb

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type ControlField struct {
	Name string
	Data []string
}

// A ControlFileLexer can be used to lex any kind of debian
// ControlFile formatted file
type ControlFileLexer struct {
	r        *bufio.Reader
	fields   chan ControlField
	errors   chan error
	action   lActionFn
	curField ControlField
}

func NewControlFileLexer(r io.Reader) *ControlFileLexer {
	return &ControlFileLexer{
		r:      bufio.NewReader(r),
		fields: make(chan ControlField, 2),
		errors: make(chan error, 3),
		action: lexEmptyLine,
	}
}

func (l *ControlFileLexer) Next() (ControlField, error) {
	for {
		select {
		case err := <-l.errors:
			if err != io.EOF {
				return ControlField{}, err
			}
		case f := <-l.fields:
			return f, nil
		default:
			if l.action == nil {
				return ControlField{}, io.EOF
			}
			l.action = l.action(l)
		}
	}
}

type lActionFn func(l *ControlFileLexer) lActionFn

func (l *ControlFileLexer) error(err error) lActionFn {
	//avoid deadlock on error channel, just drop the error
	if len(l.errors) < cap(l.errors) {
		l.errors <- err
	}
	return nil
}

func (l *ControlFileLexer) errorf(format string, args ...interface{}) lActionFn {
	return l.error(fmt.Errorf(format, args...))
}

func (l *ControlFileLexer) emitCurrent() {
	//we check that last line of field is not empty
	if len(l.curField.Data[len(l.curField.Data)-1]) == 0 {
		l.errorf("Invalid field %v, as it ends with an empty line", l.curField)
	}
	l.fields <- l.curField
}

func lexEmptyLine(l *ControlFileLexer) lActionFn {
	nextChar, err := l.r.Peek(1)
	if err != nil {
		return l.error(err)
	}

	if nextChar[0] != '\n' {
		return lexNewField
	} else {
		if _, err = l.r.ReadByte(); err != nil {
			return l.error(err)
		}
		return lexNewParagraph
	}
}

func lexNewParagraph(l *ControlFileLexer) lActionFn {
	//remove all empty lines
	for {
		nextChar, err := l.r.Peek(1)
		if err != nil {
			return l.error(err)
		}
		if nextChar[0] != '\n' {
			break
		}
		if _, err = l.r.ReadByte(); err != nil {
			return l.error(err)
		}
	}

	//TODO: should emit new paragraph

	return lexNewField
}

func lexNewField(l *ControlFileLexer) lActionFn {
	line, err := l.r.ReadString('\n')
	if err != nil && err != io.EOF {
		return l.error(err)
	}

	//check for a new fieldname
	matches := fieldNameRx.FindStringSubmatch(line)
	if matches == nil {
		return l.errorf("Got unexpected line `%s'", strings.TrimRight(line, "\n"))
	}

	l.curField = ControlField{
		Name: matches[1],
		Data: []string{
			strings.TrimSpace(strings.TrimPrefix(line, matches[0])),
		},
	}

	return lexContinuationLine
}

func lexContinuationLine(l *ControlFileLexer) lActionFn {
	nextChar, err := l.r.Peek(1)
	if err != nil && err != io.EOF {
		l.emitCurrent()
		return l.error(err)
	}

	if err == io.EOF || nextChar[0] != ' ' {
		// in that case this is not a continuation, we emit
		l.emitCurrent()
		// Now we should take care if this is a new paragraph or not
		return lexEmptyLine
	}

	line, err := l.r.ReadString('\n')
	if err != nil && err != io.EOF {
		return l.error(err)
	}

	//now we append the data, and we iterate
	l.curField.Data = append(l.curField.Data, strings.TrimSpace(line))
	return lexContinuationLine
}

var fieldNameRx = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9\-]*):`)
