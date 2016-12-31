package nagioscfg

/*
IO-related stuff for nagioscfg
Much of the stuff here is taken from Golangs encoding/json source and modified to the specific needs of this package.
See: https://golang.org/LICENSE
*/

import (
	"bufio"
	"bytes"
	"container/list"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// A ParseError is returned for parsing errors.
// The first line is 1.  The first column is 0.
type ParseError struct {
	Line   int   // Line where the error occurred
	Column int   // Column (rune index) where the error occurred
	Err    error // The actual error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Err)
}

// These are the errors that can be returned in ParseError.Error
var (
	ErrNoValue = errors.New("only key given where key/value expected")
	ErrUnknown = errors.New("unknown parsing error")
)

type Reader struct {
	Comment rune
	line    int
	column  int
	field   bytes.Buffer
	r       *bufio.Reader
}

type FileReader struct {
	*Reader
	f *os.File
}

type MultiReader []Reader

type MultiFileReader struct {
}

func _debug(args ...interface{}) {
	fmt.Println(args)
}

func NewReader(rr io.Reader) *Reader {
	return &Reader{
		Comment: '#',
		r:       bufio.NewReader(rr),
	}
}

//func NewMultiReader(rs ...io.Reader) *MultiReader {
//	mr := make(MultiReader, len(rs))
//	for i := range rs {
//		mr[i] = NewReader(rs[i])
//	}
//	return mr
//}

func NewFileReader(path string) *FileReader {
	file, err := os.Open(path)
	if err != nil {
		log.Error(err)
		return nil
	}
	fr := &FileReader{}
	fr.Reader = NewReader(file)
	fr.f = file
	return fr
}

func (fr *FileReader) Close() error {
	return fr.f.Close()
}

func (fr *FileReader) String() string {
	return fmt.Sprintf("FileReader: %s", fr.f.Name())
}

func (r *Reader) error(err error) error {
	return &ParseError{
		Line:   r.line,
		Column: r.column,
		Err:    err,
	}
}

// this is basically "dos2unix"
func (r *Reader) readRune() (rune, error) {
	r1, _, err := r.r.ReadRune()
	if r1 == '\r' {
		r1, _, err = r.r.ReadRune()
		if err == nil {
			if r1 != '\n' {
				r.r.UnreadRune()
				r1 = '\r'
			}
		}
	}
	r.column++
	return r1, err
}

// skip advances the reader until it reaches delim, ignoring everything it reads
func (r *Reader) skip(delim rune) error {
	for {
		r1, err := r.readRune()
		if err != nil {
			return err
		}
		if r1 == delim {
			return nil
		}
	}
}

func (r *Reader) parseFields() (haveField bool, delim rune, err error) {
	r.field.Reset() // clear buffer at each call

	r1, err := r.readRune()
	for err == nil && r1 != '\n' && unicode.IsSpace(r1) {
		r1, err = r.readRune()
	}
	if err == io.EOF && r.column != 0 {
		return true, 0, err
	}
	if err != nil {
		return false, 0, err
	}

	switch r1 {
	case '\n':
		fallthrough
	case '\t':
		fallthrough
	case ' ':
		fallthrough
	case '{':
		return false, r1, nil
	case '}':
		return true, r1, nil
	default:
		for {
			if !unicode.IsSpace(r1) {
				r.field.WriteRune(r1)
			}
			r1, err = r.readRune()
			//if err != nil || r1 == '{' || r1 == '}' || unicode.IsSpace(r1) {
			if err != nil || r1 == '{' || unicode.IsSpace(r1) {
				break
			}
			//if r1 == '\n' {
			//	_debug("End of line, returning")
			//	return true, r1, nil
			//}
		}
	}

	if err != nil {
		if err == io.EOF {
			return true, 0, err
		}
		return false, 0, err
	}

	return true, r1, nil
}

func (r *Reader) parseLine() (fields []string, state IoState, err error) {
	r.line++
	r.column = -1

	r1, _, err := r.r.ReadRune()
	if err != nil {
		return nil, IO_OBJ_OUT, err
	}
	if r.Comment != 0 && r1 == r.Comment {
		return nil, IO_OBJ_OUT, r.skip('\n')
	}
	r.r.UnreadRune()

	for {
		haveField, delim, err := r.parseFields()
		if haveField {
			if fields == nil {
				fields = make([]string, 0, 6) // 6 is a random guess at what is suitable
			}
			fields = append(fields, r.field.String())
		}
		if delim == '{' {
			return fields, IO_OBJ_BEGIN, nil
		} else if delim == '}' {
			return fields, IO_OBJ_END, nil
		} else if delim == '\n' {
			return fields, IO_OBJ_IN, nil
		} else if err == io.EOF {
			return fields, IO_OBJ_OUT, err
		} else if err != nil {
			return nil, IO_OBJ_OUT, err
		}
	}
}

// Read reads from a Nagios config stream and returns the next config object.
// Should be called repeatedly. Returns err = io.EOF when done (really? Does it?)
func (r *Reader) Read(setUUID bool, fileID string) (*CfgObj, error) {
	var fields []string
	var state IoState
	var err error
	var co *CfgObj

	for {
		fields, state, err = r.parseLine()
		if fields != nil {
			switch state {
			case IO_OBJ_BEGIN:
				ct := CfgName(fields[1]).Type()
				if ct == -1 {
					return nil, r.error(ErrUnknown)
				}
				if setUUID {
					co = NewCfgObjWithUUID(ct)
				} else {
					co = NewCfgObj(ct)
				}
				co.FileID = fileID
			case IO_OBJ_IN:
				fl := len(fields)
				//_debug(fields)
				if fl < 2 || co == nil {
					//return nil, r.error(ErrNoValue)
					continue
				}
				co.Add(fields[0], strings.Join(fields[1:fl], " "))
			case IO_OBJ_END:
				//fmt.Printf("Obj size: %d\n", co.size()) // approx avg turned out to be ~362 bytes per declaration for our services.cfg file
				return co, nil
			default:
				return nil, r.error(ErrUnknown)
			}
		}
		if err != nil {
			return nil, err
		}
	}

	// should not get here
	return nil, r.error(ErrUnknown)
}

// ReadAll calls Read repeatedly and returns all config objects it collects
//func (r *Reader) ReadAll(setUUID bool, fileID string) (CfgObjs, error) {
//	// Should make a version of this that approximates the number of entries based on the bytes size of the file and allocates near that number
//	objs := make(CfgObjs, 0, 64) // should find a way to calculate the approx number of entries from file/stream size, to avoid more re-alloc than needed and just hit the sweet spot at first try here
//	var obj *CfgObj
//	var err error
//	for {
//		obj, err = r.Read(setUUID, fileID)
//		if err == nil && obj != nil {
//			objs = append(objs, obj)
//		}
//		if err != nil {
//			if err != io.EOF {
//				return objs, err
//			} else {
//				break
//			}
//		}
//	}
//	return objs, nil
//}

func (r *Reader) ReadChan(setUUID bool, fileID string) <-chan *CfgObj {
	objchan := make(chan *CfgObj, 2) // making the channel buffered seems to make the function slightly faster
	go func() {
		for {
			obj, err := r.Read(setUUID, fileID)
			if err == nil && obj != nil {
				objchan <- obj
			}
			if err != nil {
				break
			}
		}
		close(objchan)
	}()
	return objchan
}

// ReadAllList does the same as ReadAll, but returns a list instead of a slice
func (r *Reader) ReadAllList(setUUID bool, fileID string) (*list.List, error) {
	l := list.New()
	for {
		obj, err := r.Read(setUUID, fileID)
		if err == nil && obj != nil {
			l.PushBack(obj)
		}
		if err != nil {
			if err != io.EOF {
				return l, err
			} else {
				break
			}
		}
	}
	return l, nil
}

func (r *Reader) ReadAllMap(fileID string) (CfgMap, error) {
	m := make(CfgMap)
	for {
		obj, err := r.Read(true, fileID)
		if err == nil && obj != nil {
			m[obj.UUID] = obj // might be better to use obj.UUID.String()
		}
		if err != nil {
			if err != io.EOF {
				return m, err
			} else {
				break
			}
		}
	}
	return m, nil
}

//func (mr *MultiReader) Close() error {
//	errcnt := 0
//	for i := range mr {
//		err := mr[i].Close()
//		if err != nil {
//			errcnt++
//			log.Error(err)
//		}
//	}
//	if errcnt > 0 {
//		return fmt.Errorf("MultiReader.Close(): Encountered %d errors closing readers", errcnt)
//	}
//	return nil
//}

func (mr *MultiReader) ReadChan(setUUID bool) {
}

func (co *CfgObj) PrintProps(w io.Writer, format string) {
	for k, v := range co.Props {
		fmt.Fprintf(w, format, k, v)
	}
}

func (co *CfgObj) PrintPropsSorted(w io.Writer, format string) {
	keypri := make(map[int]string)
	for k := range co.Props {
		keypri[CfgKeySortOrder[k][co.Type]] = k // should have error checking for non-existing keys/types
	}
	keys := make([]int, len(keypri))
	i := 0
	for k := range keypri {
		keys[i] = k
		i++
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Fprintf(w, format, keypri[k], co.Props[keypri[k]])
	}
}

// Print prints out a CfgObj in Nagios format
func (co *CfgObj) Print(w io.Writer) {
	prefix := strings.Repeat(" ", co.Indent)
	fstr := fmt.Sprintf("%s%s%d%s", prefix, "%-", co.Align, "s%s\n")
	co.generateComment() // this might fail, but don't care yet
	fmt.Fprintf(w, "%s\n", co.Comment)
	fmt.Fprintf(w, "define %s{\n", co.Type.String())
	//for k, v := range co.Props {
	//	fmt.Fprintf(w, fstr, k, v)
	//}
	co.PrintProps(w, fstr)
	//co.PrintPropsSorted(w, fstr)
	fmt.Fprintf(w, "%s}\n", prefix)
}

// PrintSorted prints out a CfgObj in Nagios format, sorted like Nagios/op5 would do
//func (co *CfgObj) PrintSorted(w io.Writer) {
//	fmt.Fprintf(w, "# Implement it yourself, Göran!\n")
//}

// Print writes a collection of CfgObj to a given stream
func (cos CfgObjs) Print(w io.Writer) {
	for i := range cos {
		cos[i].Print(w)
		fmt.Fprint(w, "\n")
	}
}

// PrintSorted writes a collection of CfgObj to a given stream, sorted by host_name/service_description
//func (cos CfgObjs) PrintSorted(w io.Writer) {
//	for i := range cos {
//		cos[i].PrintSorted(w)
//		fmt.Fprintf(w, "\n")
//	}
//}

func (cm CfgMap) Print(w io.Writer) {
	for k := range cm {
		cm[k].Print(w)
		fmt.Fprintf(w, "\n")
	}
}

func (cm CfgMap) WriteByFileID() error {
	var wg sync.WaitGroup
	fmap := cm.SplitByFileID()
	schan := make(chan error)

	for fname := range fmap {
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			fhnd, err := os.Create(filename)
			if err != nil {
				schan <- err
				return
			}
			defer fhnd.Close()
			w := bufio.NewWriter(fhnd)
			for i := range fmap[filename] {
				cm[fmap[filename][i]].Print(w)
			}
			w.Flush()
			schan <- nil
		}(fname)
	}

	go func() {
		wg.Wait()
		close(schan)
	}()

	var errcnt int
	for e := range schan {
		if e != nil {
			log.Error(e)
			errcnt++
		}
	}

	if errcnt > 0 {
		return fmt.Errorf("CfgMap.WriteByFileID(): Error writing to %d files", errcnt)
	}

	return nil
}

//func readFileToCfgMap(fileName string) (CfgMap, error) {
//	file, err := os.Open(fileName)
//	if err != nil {
//		return nil, err
//	}
//	defer file.Close()
//	r := NewReader(file)
//	cmap, err := r.ReadAllMap(fileName)
//	if err != nil {
//		return nil, err
//	}
//	return cmap, nil
//}

//func writeFileFromCfgMap(fileName string, cmap CfgMap) error {
//	file, err := os.Create(fileName)
//	if err != nil {
//		return err
//	}
//	defer file.Close()
//	w := bufio.NewWriter(file)
//	//
//
//	w.Flush()
//	return nil
//}

//func readFileToCfgObjs(fileName string, setUUID bool) (CfgObjs, error) {
//	file, err := os.Open(fileName)
//	if err != nil {
//		return nil, err
//	}
//	defer file.Close()
//	r := NewReader(file)
//	objs, err := r.ReadAll(setUUID, fileName)
//	if err != nil {
//		return nil, err
//	}
//	return objs, nil
//}

//func writeFileFromCfgObjs(fileName string, objs CfgObjs) error {
//	file, err := os.Create(fileName)
//	if err != nil {
//		return err
//	}
//	defer file.Close()
//	w := bufio.NewWriter(file)
//	objs.Print(w)
//	w.Flush()
//	return nil
//}


//func ReadFile(fileName string, setUUID bool) (CfgObjs, error) {
//	file, err := os.Open(fileName)
//	if err != nil {
//		return nil, err
//	}
//	defer file.Close()
//	r := NewReader(file)
//	objs, err := r.ReadAll(setUUID, fileName)
//	if err != nil {
//		return nil, err
//	}
//	return objs, nil
//}

//func WriteFile(fileName string, objs CfgObjs) error {
//	file, err := os.Create(fileName)
//	if err != nil {
//		return err
//	}
//	defer file.Close()
//	w := bufio.NewWriter(file)
//	objs.Print(w)
//	w.Flush()
//	return nil
//}

//func NewCfgFile(path string) *CfgFile {
//	objs := make(CfgObjs, 0)
//	return &CfgFile{
//		Path: path,
//		Objs: objs,
//	}
//}

//func (cf *CfgFile) Read(setUUID bool) error {
//	objs, err := ReadFile(cf.Path, setUUID)
//	if err != nil {
//		return err
//	}
//	if objs != nil {
//		cf.Objs = objs
//	}
//	return nil
//}

//func (cf *CfgFile) Write() error {
//	return WriteFile(cf.Path, cf.Objs)
//}
