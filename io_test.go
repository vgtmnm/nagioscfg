/*
   Copyright 2017 Odd Eivind Ebbesen

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package nagioscfg

import (
	"fmt"
	"io/ioutil"
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
	host_name localhost1
	          check_command check_gris
    }
	
define command {
	command_name gris
	gris_fest roligt
	   command_line $USER1$/dont_buy_lynx
}

# Bla bla, some comment crap
# I'm really too tired now
  Just fucking up the format without comment signs....

define service{
	service_description Disk usage /my/ass
	contact_groups toilet
host_name grevling
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
	co.Print(os.Stdout, true)
}

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
		co.Print(os.Stdout, true)
	}
}

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
}

func TestReadAllMapMulti(t *testing.T) {
	files := []string{
		"/tmp/ncfg-testwritebyfileid_0.cfg",
		"/tmp/ncfg-testwritebyfileid_1.cfg",
		"/tmp/ncfg-testwritebyfileid_2.cfg",
	}
	mfr := NewMultiFileReader(files...)
	defer mfr.Close()
	cm, err := mfr.ReadAllMap()
	if err != nil {
		t.Error(err)
	}
	t.Log(cm.Dump())
}


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

func BenchmarkPrintObjProps(b *testing.B) {
	path := "../op5_automation/cfg/etc/services-mini.cfg"
	fr := NewFileReader(path)
	cm, err := fr.ReadAllMap(path)
	if err != nil {
		b.Fatal(err)
	}
	fr.Close()

	fstr := "%s %s"
	for i := 0; i <= b.N; i++ {
		for k := range cm {
			cm[k].PrintProps(ioutil.Discard, fstr)
		}
	}
}

func BenchmarkPrintObjPropsSorted(b *testing.B) {
	path := "../op5_automation/cfg/etc/services-mini.cfg"
	fr := NewFileReader(path)
	cm, err := fr.ReadAllMap(path)
	if err != nil {
		b.Fatal(err)
	}
	fr.Close()

	fstr := "%s %s"
	for i := 0; i <= b.N; i++ {
		for k := range cm {
			cm[k].PrintPropsSorted(ioutil.Discard, fstr)
		}
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
	err = cmap.WriteByFileID(true)
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


func getLocalObjs() (CfgMap, error) {
	prefix := "../op5_automation/cfg/etc/"
	_f := func(f string) string {
		return fmt.Sprintf("%s%s", prefix, f)
	}
	files := []string{
		_f("checkcommands.cfg"),
		_f("contactgroups.cfg"),
		_f("contacts.cfg"),
		_f("eventhandlers.cfg"),
		_f("hostdependencies.cfg"),
		_f("hostescalations.cfg"),
		_f("hostgroups.cfg"),
		_f("hosts.cfg"),
		_f("misccommands.cfg"),
		_f("servicedependencies.cfg"),
		_f("serviceescalations.cfg"),
		_f("servicegroups.cfg"),
		_f("services.cfg"),
		_f("timeperiods.cfg"),
		_f("vgt_dps/amq_cluster_dummies.cfg"),
		_f("vgt_dps/aws_availability.cfg"),
		_f("vgt_dps/aws_mb.cfg"),
		_f("vgt_dps/aws_shared.cfg"),
		_f("vgt_dps/deep_pings_ageo_prod.cfg"),
		_f("vgt_dps/deep_pings_avtfleet.cfg"),
		_f("vgt_dps/deep_pings_avtfleet_cn.cfg"),
		_f("vgt_dps/deep_pings_avtgot2.cfg"),
		_f("vgt_dps/deep_pings_avtgot2_qa.cfg"),
		_f("vgt_dps/deep_pings_aws_daimler_mbiot.cfg"),
		_f("vgt_dps/deep_pings_caretrack.cfg"),
		_f("vgt_dps/deep_pings_dfol_prod.cfg"),
		_f("vgt_dps/deep_pings_dfol_qa.cfg"),
		_f("vgt_dps/deep_pings_dfol_swr8.cfg"),
		_f("vgt_dps/deep_pings_dug_prod.cfg"),
		_f("vgt_dps/deep_pings_geofence.cfg"),
		_f("vgt_dps/deep_pings_jlr_cn.cfg"),
		_f("vgt_dps/deep_pings_jlr_cn_qa1.cfg"),
		_f("vgt_dps/deep_pings_jlr_cn_zone-a.cfg"),
		_f("vgt_dps/deep_pings_jlr_cn_zone-b.cfg"),
		_f("vgt_dps/deep_pings_jlr_preprod.cfg"),
		_f("vgt_dps/deep_pings_jlr_prod.cfg"),
		_f("vgt_dps/deep_pings_jlr_prod_LB.cfg"),
		_f("vgt_dps/deep_pings_nissan_preprod.cfg"),
		_f("vgt_dps/deep_pings_nissan_prod.cfg"),
		_f("vgt_dps/deep_pings_openportal_prod.cfg"),
		_f("vgt_dps/deep_pings_opus_got.cfg"),
		_f("vgt_dps/deep_pings_opus_gso.cfg"),
		_f("vgt_dps/deep_pings_opus_tjn.cfg"),
		_f("vgt_dps/deep_pings_preprod.cfg"),
		_f("vgt_dps/deep_pings_sirun.cfg"),
		_f("vgt_dps/deep_pings_uptime.cfg"),
		_f("vgt_dps/deep_pings_uptime_qa.cfg"),
		_f("vgt_dps/deep_pings_vlink_prod.cfg"),
		_f("vgt_dps/deep_pings_vlink_val.cfg"),
		_f("vgt_dps/deep_pings_voc.cfg"),
		_f("vgt_dps/deep_pings_voc_cn.cfg"),
		_f("vgt_dps/deep_pings_voc_cn_qa.cfg"),
		_f("vgt_dps/deep_pings_voc_eu_qa.cfg"),
		_f("vgt_dps/deep_pings_voc_na.cfg"),
		_f("vgt_dps/deep_pings_voc_na_qa.cfg"),
		_f("vgt_dps/ssl_cert_dummy.cfg"),
	}
	mfr := NewMultiFileReader(files...)
	defer mfr.Close()

	return mfr.ReadAllMap()
}

//func BenchmarkInvertMap(b *testing.B) {
//	objs, err := getLocalObjs()
//	if err != nil {
//		b.Fatal(err)
//	}
//	for i := 0; i < b.N; i++ {
//	}
//}

//func BenchmarkInvertCmpSlice(b *testing.B) {
//	objs, err := getLocalObjs()
//	if err != nil {
//		b.Fatal(err)
//	}
//	for i := 0; i < b.N; i++ {
//	}
//}

func TestCfgObjMarshalJSON(t *testing.T) {
	co := NewCfgObjWithUUID(T_SERVICE)
	co.Add("use", "linux-prod")
	co.Add("host_name", "localhost")
	co.Add("service_description", "JSONTest")
	co.Add("check_command", "check_json_output!param1!param2")
	co.FileID = "/opt/monitor/etc/services.cfg"

	jval, err := co.MarshalJSON()
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", jval)
}

func TestCfgObjUnmarshalJSON(t *testing.T) {
	jbytes := []byte(`{"uuid":"3d3f8292-713b-11e7-aa16-0800279d8583","fileid":"/opt/monitor/etc/services.cfg","type":8,"props":{"host_name":"localhost","service_description":"JSON Test","check_command":"check_json_output!param1!param2","use":"linux-prod"}}`)
	co := &CfgObj{}
	err := co.UnmarshalJSON(jbytes)
	if err != nil {
		t.Error(err)
	}
	co.Print(os.Stdout, true)
	fmt.Printf("Obj UUID   : %s\n", co.UUID)
	fmt.Printf("Obj FileID : %s\n", co.FileID)
}

func TestCfgMapMarshalJSON(t *testing.T) {
	str_r := strings.NewReader(cfgobjstr)
	rdr := NewReader(str_r)
	m, err := rdr.ReadAllMap("/dev/null")
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("=== Map: ===\n%s\n", m.Dump())
	}
	// do actual json shit
	jval, err := m.MarshalJSON()
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", jval)
}

func TestCfgMapUnmarshalJSON(t *testing.T) {
	jbytes := []byte(`{"0e03153b-7182-11e7-ba59-0800279d8583":{"uuid":"0e03153b-7182-11e7-ba59-0800279d8583","fileid":"/dev/null","type":8,"props":{"service_description":"Disk usage /my/ass","contact_groups":"toilet","host_name":"grevling"}},"0e03130a-7182-11e7-ba59-0800279d8583":{"uuid":"0e03130a-7182-11e7-ba59-0800279d8583","fileid":"/dev/null","type":8,"props":{"service_description":"A service name with spaces","host_name":"localhost1","check_command":"check_gris"}},"0e0314cb-7182-11e7-ba59-0800279d8583":{"uuid":"0e0314cb-7182-11e7-ba59-0800279d8583","fileid":"/dev/null","type":0,"props":{"command_line":"$USER1$/dont_buy_lynx","command_name":"gris"}}}`)
	cm := make(CfgMap)
	err := cm.UnmarshalJSON(jbytes)
	if err != nil {
		t.Error(err)
	}
	cm.Print(os.Stdout, true)
}

func TestNcfgMarshalJSON(t *testing.T) {
	path := "../op5_automation/cfg/etc/services-mini.cfg"
	fr := NewFileReader(path)
	defer fr.Close()
	cm, err := fr.ReadAllMap(path)
	if err != nil {
		t.Error(err)
	}
	ncfg := NewNagiosCfg()
	ncfg.Config = cm

	jbuf, err := ncfg.MarshalJSON()
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", jbuf)
}

func TestNcfgUnmarshalJSON(t *testing.T) {
	//jbytes := []byte(`{"sessionid":"02e67b59-7193-11e7-82f9-0800279d8583","date":"2017-07-26T01:43:08.08799836+02:00","version":"2017-07-26","cfg":{"02e67853-7193-11e7-82f9-0800279d8583":{"uuid":"02e67853-7193-11e7-82f9-0800279d8583","fileid":"../op5_automation/cfg/etc/services-mini.cfg","type":8,"props":{"check_command":"check_snmpif_traffic_v2!wcar_supervision!224!1000mbit!70!90","servicegroups":"VGT_Infrastructure_Services","use":"linux-prod","host_name":"vgt-cn-sha-lb-02","service_description":"Interface 224 Traffic"}},"02e678f5-7193-11e7-82f9-0800279d8583":{"uuid":"02e678f5-7193-11e7-82f9-0800279d8583","fileid":"../op5_automation/cfg/etc/services-mini.cfg","type":8,"props":{"use":"linux-prod","host_name":"vgt-cn-sha-lb-02","service_description":"PING","check_command":"check_ping!100,20%!500,60%","servicegroups":"VGT_Infrastructure_Services"}},"02e67951-7193-11e7-82f9-0800279d8583":{"uuid":"02e67951-7193-11e7-82f9-0800279d8583","fileid":"../op5_automation/cfg/etc/services-mini.cfg","type":8,"props":{"check_command":"vgt_check_f5_psu!wcar_supervision!5","servicegroups":"PROD_VOC_CN_Services,VGT_Infrastructure_Services","contact_groups":"wcar_jour_got_sms,wcar_network","use":"linux-prod","host_name":"vgt-cn-sha-lb-02","service_description":"PSU Status"}}}}`)
}
