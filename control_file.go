package deb

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"net/mail"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ControlField struct {
	Name string
	Data []string
}

func (c ControlField) String() string {
	return fmt.Sprintf("%s: %#q", c.Name, c.Data)
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

func IsNewParagraph(c ControlField) bool {
	return c.Name == "" && c.Data == nil
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
	if IsNewParagraph(l.curField) == false &&
		len(l.curField.Data[len(l.curField.Data)-1]) == 0 {
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

	// This field will be such that IsNewParagraph() is true
	l.curField = ControlField{
		Name: "",
		Data: nil,
	}

	l.emitCurrent()

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

type controlFieldParser func(ControlField, interface{}) error

func getFieldIDFromTag(v interface{}, tag string) (int, error) {
	vType := reflect.TypeOf(v).Elem()
	if vType.Kind() != reflect.Struct {
		return -1, fmt.Errorf("%v is not a struct", vType)
	}
	resTag := -1
	resName := -1
	for i := 0; i < vType.NumField(); i = i + 1 {
		sf := vType.Field(i)
		sfTag := sf.Tag.Get("field")
		if sfTag == tag {
			resTag = i
			continue
		}

		if sf.Name == tag {
			resName = i
			continue
		}
	}
	res := -1
	var err error = nil
	if resTag == -1 && resName == -1 {
		err = fmt.Errorf(`Could not find field in %v with tag field:"%s"`, vType, tag)
	}
	if resTag != -1 {
		res = resTag
	} else {
		res = resName
	}

	return res, err
}

func setField(toModify interface{}, name string, v interface{}) error {
	mType := reflect.TypeOf(toModify)
	if mType.Kind() != reflect.Ptr {
		return fmt.Errorf("%v is not a pointer", mType)
	}
	id, err := getFieldIDFromTag(toModify, name)
	if err != nil {
		return err
	}

	mValue := reflect.Indirect(reflect.ValueOf(toModify)).Field(id)
	if mValue.CanSet() == false {
		return fmt.Errorf("Could not set field %s in %v", name, mType)
	}
	vType := reflect.TypeOf(v)
	if vType.AssignableTo(mValue.Type()) == false {
		return fmt.Errorf("Could not set field %s in %v, %v is not assignable to %v", name, mType, vType, mValue.Type())
	}
	mValue.Set(reflect.ValueOf(v))
	return nil
}

func expectSingleLine(f ControlField) error {
	if len(f.Data) != 1 {
		return fmt.Errorf("expected a single line field")
	}
	return nil
}

func expectMultiLine(f ControlField) error {
	if len(f.Data) <= 1 || len(f.Data[0]) != 0 {
		return fmt.Errorf("expected a multi-line field, first line empty")
	}
	return nil
}

func parseChangesFormat(f ControlField, v interface{}) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}
	ver, err := ParseVersion(f.Data[0])
	if err != nil {
		return err
	}

	if ver.Epoch != 0 || ver.DebianRevision != "0" {
		return fmt.Errorf("it should have no epoch or debian revision")
	}

	return setField(v, "Format", *ver)
}

func parseDate(f ControlField, v interface{}) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}
	elems := strings.Split(f.Data[0], " ")
	offset := elems[len(elems)-1]
	if len(offset) != 5 || (offset[0] != '+' && offset[0] != '-') {
		return fmt.Errorf("invalid UTC offset `%s'", offset)
	}
	offsetHours, err := strconv.Atoi(offset[0:3])
	if err != nil {
		return fmt.Errorf("invalid UTC offset `%s'", offset)
	}
	offsetMinutes, err := strconv.Atoi(offset[3:5])
	if err != nil {
		return fmt.Errorf("invalid UTC offset `%s'", offset)
	}
	if offsetMinutes < 0 || offsetMinutes > 59 {
		return fmt.Errorf("invalid UTC offset `%s'", offset)
	}
	if offset[0] == '-' {
		offsetMinutes = -offsetMinutes
	}

	// parse the offseted time
	date, err := time.Parse("Mon, 02 Jan 2006 15:04:05",
		strings.Join(elems[0:len(elems)-1], " "))
	if err != nil {
		return err
	}

	//remove the extracted offset
	date = date.Add(-time.Duration(offsetHours)*time.Hour - time.Duration(offsetMinutes)*time.Minute)
	return setField(v, "Date", date)
}

func parseArchitecture(f ControlField, v interface{}) error {
	archs := []Architecture{}
	for _, l := range f.Data {
		for _, as := range strings.Split(l, " ") {
			a := Architecture(as)
			_, ok := ArchitectureList[a]
			if ok == false {
				return fmt.Errorf("unknown architecture %s", a)
			}
			archs = append(archs, a)
		}
	}
	return setField(v, "Architectures", archs)
}

func parseDistribution(f ControlField, v interface{}) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}

	d := strings.TrimSpace(f.Data[0])
	if strings.Contains(d, " ") {
		return fmt.Errorf("does not contains a single distribution")
	}

	return setField(v, "Distribution", Codename(d))
}

func parseMaintainer(f ControlField, v interface{}) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}

	mail, err := mail.ParseAddress(f.Data[0])
	if err != nil {
		return err
	}

	return setField(v, "Maintainer", mail)
}

func parseChanges(f ControlField, v interface{}) error {
	if err := expectMultiLine(f); err != nil {
		return err
	}
	return setField(v, "Changes", strings.Join(f.Data[1:], "\n"))
}

func parseDescription(f ControlField, v interface{}) error {
	if err := expectMultiLine(f); err != nil {
		return err
	}
	return setField(v, "Description", strings.Join(f.Data[1:], "\n"))
}

func parseFileList(f ControlField) ([]FileReference, error) {
	files := []FileReference{}
	if err := expectMultiLine(f); err != nil {
		return nil, err
	}

	for _, line := range f.Data[1:] {
		data := strings.Split(line, " ")
		if len(data) != 3 && len(data) != 5 {
			return nil, fmt.Errorf("invalid line `%s' (%d elements), expected `checksum size [section priority] name'", line, len(data))
		}

		file := FileReference{
			Name: data[len(data)-1],
		}

		_, err := fmt.Sscanf(data[1], "%d", &file.Size)
		if err != nil {
			return nil, err
		}
		file.Checksum, err = hex.DecodeString(data[0])
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func parseSha1(f ControlField, v interface{}) error {
	files, err := parseFileList(f)
	if err != nil {
		return err
	}
	return setField(v, "ChecksumsSha1", files)
}

func parseSha256(f ControlField, v interface{}) error {
	files, err := parseFileList(f)
	if err != nil {
		return err
	}
	return setField(v, "ChecksumsSha256", files)
}

func parseFiles(f ControlField, v interface{}) error {
	files, err := parseFileList(f)
	if err != nil {
		return err
	}

	return setField(v, "Files", files)
}

func parseSource(f ControlField, v interface{}) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}

	s := strings.TrimSpace(f.Data[0])
	if strings.Contains(s, " ") {
		return fmt.Errorf("multiple source name")
	}

	return setField(v, "Source", s)
}

func parseVersion(f ControlField, v interface{}) error {
	if err := expectSingleLine(f); err != nil {
		return err
	}
	ver, err := ParseVersion(f.Data[0])
	if err != nil {
		return err
	}
	return setField(v, "Version", *ver)
}

func parseBinary(f ControlField, v interface{}) error {
	res := []string{}
	for _, line := range f.Data {
		res = append(res, strings.Split(line, " ")...)
	}
	return setField(v, "Binary", res)
}

type controlFileParser struct {
	l        *ControlFileLexer
	fMapper  map[string]controlFieldParser
	required []string
}

func (p *controlFileParser) parse(v interface{}) error {
	parsedField := make(map[string]bool)
	for {
		f, err := p.l.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if IsNewParagraph(f) {
			return fmt.Errorf("expect a single paragraph")
		}

		fn, ok := p.fMapper[f.Name]
		if ok == false {
			return fmt.Errorf("unexpected field %s:%v", f.Name, f.Data)
		}
		parsedField[f.Name] = true
		if fn == nil {
			continue
		}

		err = fn(f, v)
		if err != nil {
			return fmt.Errorf("invalid field %s:%v: %s", f.Name, f.Data, err)
		}
	}

	missing := make([]string, 0)
	for _, fName := range p.required {
		if _, ok := parsedField[fName]; ok == false {
			missing = append(missing, fName)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required field %v", missing)
	}

	return nil
}
