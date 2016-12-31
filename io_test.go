package nagioscfg

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

var cfgobjstr string = `# some comment
define service {
	  service_description    A service name with spaces
# embedded comment
	  a_key                  Some value
		singlekey
    }
	
define command {
	command_name gris
	gris_fest roligt
}

# Bla bla, some comment crap
# I'm really too tired now

define service{
	service_description Disk usage /my/ass
	contact_group toilet
}
`

func TestRead(t *testing.T) {
	str_r := strings.NewReader(cfgobjstr)
	rdr := NewReader(str_r)
	co, err := rdr.Read(false, "/dev/null")
	if err != nil {
		t.Fatal(err)
	}
	if co == nil {
		t.Fatal("CfgObj is nil")
	}
	co.AutoAlign()
	co.Print(os.Stdout)
}

//func TestReadAll(t *testing.T) {
//	//t.Skip("Not implemented yet")
//	str_r := strings.NewReader(cfgobjstr)
//	rdr := NewReader(str_r)
//	cos, err := rdr.ReadAll(false, "/dev/null")
//	if err != nil {
//		t.Error(err)
//	} else {
//		cos.AutoAlign()
//		cos.Print(os.Stdout)
//	}
//}

func TestReadAllMap(t *testing.T) {
	str_r := strings.NewReader(cfgobjstr)
	rdr := NewReader(str_r)
	m, err := rdr.ReadAllMap("/dev/null")
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("=== Map: ===\n%s\n", m.Dump())
	}
}

// Test how we can use UUID as a map key and use the string representation back and forth to retrieve the entry
func TestUUIDMapKeys(t *testing.T) {
	str_r := strings.NewReader(cfgobjstr)
	rdr := NewReader(str_r)
	m, err := rdr.ReadAllMap("/dev/null")
	if err != nil {
		t.Error(err)
	}
	strkeys := make([]string, 0, 4)
	for k := range m {
		strkeys = append(strkeys, m[k].UUID.String())
	}
	t.Log(strkeys)

	for i := range strkeys {
		u, err := UUIDFromString(strkeys[i])
		if err != nil {
			t.Error(err)
		}
		co, found := m.Get(u.String())
		if !found {
			t.Errorf("Could not find map entry for key %q", u)
			continue
		}
		co.Print(os.Stdout)
	}
}

//func TestReadFile(t *testing.T) {
//	path := "../op5_automation/cfg/etc/services.cfg"
//	objs, err := ReadFile(path, false)
//	if err != nil {
//		t.Error(err)
//	}
//	t.Log("Number of objets read: ", len(objs))
//}

func TestReadFileChan(t *testing.T) {
	path := "../op5_automation/cfg/etc/services-mini.cfg"
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	r := NewReader(file)
	ochan := r.ReadChan(false, path)
	for o := range ochan {
		name, _ := o.GetUniqueCheckName()
		t.Logf("Read config object from channel: %q", name)
	}
	//for o, ok := <-ochan; ok; o, ok = <-ochan {
	//	if ok {
	//		name, _ := o.GetUniqueCheckName()
	//		t.Log("Read one config object from channel:", name)
	//	} else {
	//		t.Error("Channel closed")
	//	}
	//}
}

//func BenchmarkReadFile(b *testing.B) {
//	path := "../op5_automation/cfg/etc/services-mini.cfg"
//	for i := 0; i <= b.N; i++ {
//		ReadFile(path, false)
//	}
//}

func TestReadMultiFileChan(t *testing.T) {
	files := []string{
		"/tmp/ncfg-testwritebyfileid_0.cfg",
		"/tmp/ncfg-testwritebyfileid_1.cfg",
		"/tmp/ncfg-testwritebyfileid_2.cfg",
	}
	mfr := NewMultiFileReader(files...)
	ochan := mfr.ReadChan(true)
	expobjnum := 3
	objcnt := 0
	for o := range ochan {
		name, _ := o.GetUniqueCheckName()
		t.Logf("%s %q", name, o.FileID)
		objcnt++
	}
	mfr.Close()
	if objcnt != expobjnum {
		t.Errorf("Expected to read %d objects from channel, but got %d", expobjnum, objcnt)
	}
}

func BenchmarkReadFileChan(b *testing.B) {
	path := "../op5_automation/cfg/etc/services-mini.cfg"
	for i := 0; i <= b.N; i++ {
		file, err := os.Open(path)
		if err != nil {
			b.Fatal(err)
		}
		r := NewReader(file)
		ochan := r.ReadChan(false, path)
		for o := range ochan {
			if o == nil {
				b.Error("Got empty object")
			}
		}
		file.Close()
	}
}

func TestWriteByFileID(t *testing.T) {
	path := "../op5_automation/cfg/etc/services-mini.cfg"
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	r := NewReader(file)
	ochan := r.ReadChan(true, path)
	cmap := make(CfgMap)
	i := 0
	for o := range ochan {
		o.FileID = fmt.Sprintf("/tmp/ncfg-testwritebyfileid_%d.cfg", i)
		i++
		cmap[o.UUID] = o
	}
	//t.Log("\n", cmap.Dump())
	err = cmap.WriteByFileID()
	if err != nil {
		t.Error(err)
	}
}

func TestNewFileReader(t *testing.T) {
	//path := "../op5_automation/cfg/etc/services-mini.cfg"
	path := "/tmp/a.txt"
	fr := NewFileReader(path)
	if fr == nil {
		t.Fatalf("Failed to create new FileReader from path %q", path)
	}
	t.Logf("%+v", fr)
	err := fr.Close()
	if err != nil {
		t.Error(err)
	}
}

func TestNewMultiFileReader(t *testing.T) {
	files := []string{"/tmp/a.txt", "/tmp/b.txt", "/tmp/c.txt"}
	mfr := NewMultiFileReader(files...)
	t.Logf("MFR length: %d", len(mfr))
	for i := range mfr {
		t.Logf("MFR entry #%d: %s", i, mfr[i])
	}
	err := mfr.Close()
	if err != nil {
		t.Error(err)
	}
}

//func BenchmarkReadFileSetUUID(b *testing.B) {
//	path := "../op5_automation/cfg/etc/services-mini.cfg"
//	for i := 0; i <= b.N; i++ {
//		ReadFile(path, true)
//	}
//}

//func TestObjReadFile(t *testing.T) {
//	path := "../op5_automation/cfg/etc/services.cfg"
//	cf := NewCfgFile(path)
//	err := cf.Read(false)
//	if err != nil {
//		t.Error(err)
//	}
//	t.Log("Number of objets read: ", len(cf.Objs))
//}

//func TestWriteFile(t *testing.T) {
//	src := "../op5_automation/cfg/etc/services.cfg"
//	dst := "/tmp/services.cfg"
//	objs, err := ReadFile(src, false)
//	if err != nil {
//		t.Error(err)
//	}
//	t.Log("Number of objets read: ", len(objs))
//	err = WriteFile(dst, objs)
//	if err != nil {
//		t.Error(err)
//	}
//}

//func TestObjWriteFile(t *testing.T) {
//	src := "../op5_automation/cfg/etc/services.cfg"
//	dst := "/tmp/services.cfg"
//	cf := NewCfgFile(src)
//	err := cf.Read(false)
//	if err != nil {
//		t.Error(err)
//	}
//	t.Log("Number of objets read: ", len(cf.Objs))
//	cf.Path = dst
//	err = cf.Write()
//	if err != nil {
//		t.Error(err)
//	}
//}
