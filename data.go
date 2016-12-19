package nagioscfg

/*
Defines the data structures that maps to Nagios config items
Odd, 2016-08-10 17:51
*/

import (
	//"io"
	"regexp"
)

//type WriteMap map[string]CfgMap // used to sort/write out according to FileID

//type CfgKey int

// UUID representation compliant with specification
// described in RFC 4122.
type UUID [16]byte

type CfgType int
type CfgName string
type IoState int
type CfgObjs []*CfgObj
type CfgMap map[UUID]*CfgObj

const (
	T_COMMAND CfgType = iota
	T_CONTACTGROUP
	T_CONTACT
	T_HOSTESCALATION
	T_HOSTGROUP
	T_HOST
	T_SERVICEESCALATION
	T_SERVICEGROUP
	T_SERVICE
	T_TIMEPERIOD
)

const (
	DEF_INDENT int    = 4
	DEF_ALIGN  int    = 32
	SEP_CMD    string = "!"
	SEP_LST    string = ","
)

const (
	IO_OBJ_OUT IoState = iota
	IO_OBJ_BEGIN
	IO_OBJ_IN
	IO_OBJ_END
)

var CfgTypes = [...]CfgName{
	"command",
	"contactgroup",
	"contact",
	"hostescalation",
	"hostgroup",
	"host",
	"serviceescalation",
	"servicegroup",
	"service",
	"timeperiod",
}

var CfgKeys = [...]string{
	"active_checks_enabled",         // 00
	"address",                       // 01
	"alias",                         // 02
	"can_submit_commands",           // 03
	"check_command",                 // 04
	"check_freshness",               // 05
	"check_interval",                // 06
	"check_period",                  // 07
	"command_line",                  // 08
	"command_name",                  // 09
	"contact_groups",                // 10
	"contact_name",                  // 11
	"contactgroup_name",             // 12
	"contactgroups",                 // 13
	"contacts",                      // 14
	"display_name",                  // 15
	"email",                         // 16
	"escalation_options",            // 17
	"escalation_period",             // 18
	"event_handler_enabled",         // 19
	"first_notification",            // 20
	"flap_detection_enabled",        // 21
	"flap_detection_options",        // 22
	"friday",                        // 23
	"host_name",                     // 24
	"host_notification_commands",    // 25
	"host_notification_options",     // 26
	"host_notification_period",      // 27
	"host_notifications_enabled",    // 28
	"hostgroup_members",             // 29
	"hostgroup_name",                // 30
	"hostgroups",                    // 31
	"icon_image",                    // 32
	"is_volatile",                   // 33
	"last_notification",             // 34
	"max_check_attempts",            // 35
	"monday",                        // 36
	"name",                          // 37
	"notes",                         // 38
	"notes_url",                     // 39
	"notification_interval",         // 40
	"notification_options",          // 41
	"notification_period",           // 42
	"notifications_enabled",         // 43
	"obsess",                        // 44
	"pager",                         // 45
	"parallelize_check",             // 46
	"parents",                       // 47
	"passive_checks_enabled",        // 48
	"process_perf_data",             // 49
	"register",                      // 50
	"retain_nonstatus_information",  // 51
	"retain_status_information",     // 52
	"retry_interval",                // 53
	"saturday",                      // 54
	"service_description",           // 55
	"service_notification_commands", // 56
	"service_notification_options",  // 57
	"service_notification_period",   // 58
	"service_notifications_enabled", // 59
	"servicegroup_name",             // 60
	"servicegroups",                 // 61
	"stalking_options",              // 62
	"statusmap_image",               // 63
	"sunday",                        // 64
	"thursday",                      // 65
	"timeperiod_name",               // 66
	"tuesday",                       // 67
	"use",                           // 68
	"wednesday",                     // 69
}

// Key order as defined here:
// https://assets.nagios.com/downloads/nagioscore/docs/nagioscore/3/en/objectdefinitions.html#service
var CfgKeyOrderService [...]int{
	24, // host_name
	30, //hostgroup_name
	55, // service_description
	15, // display_name
	61, // servicegroups
	33, // is_volatile
	04, // check_command
	// initial_state should come here
	35, // max_check_attempts
	06, // check_interval
	53, // retry_interval
	00, // active_checks_enabled
	48, // passive_checks_enabled
	07, // check_period
	// obsess_over_service should come here
	05, // check_freshness
	// 	freshness_threshold should come here
	// event_handler should come here
	19, // event_handler_enabled
	// 	low_flap_threshold should come here
	// high_flap_threshold should come here
	21. // flap_detection_enabled
	22, // flap_detection_options
	49, // process_perf_data
	52, // retain_status_information
	51, // retain_nonstatus_information
	40, // notification_interval
	// 	first_notification_delay should come here
	42, // notification_period
	41, // notification_options
	43, // notifications_enabled
	14, // contacts
	10, // contact_groups
	62, // stalking_options
	38, // notes
	39, // notes_url
	// 	action_url should come here
	32, // icon_image
	// 	icon_image_alt should come here
}

/*
type PropertyCollection interface {
	Add(key, val string) bool      // should only add if key does not yet exist. Return false if key exists
	Set(key, val string) bool      // adds or overwrites. Return true if key was overwritten
	Get(key string) (string, bool) // return val, success
	Del(key string) bool           // return true if key was present
	LongestKey() int
}
*/

//type CfgObjCollection interface {
//	Add(key string, val *CfgObj) bool
//	Set(key string, val *CfgObj) bool
//	Get(key string) (*CfgObj, bool)
//	Del(key string) *CfgObj
//	LongestKey() int
//	MatchKeys(rx *regexp.Regexp, keys ...string) CfgObjCollection
//	MatchAny(rx *regexp.Regexp)  CfgObjCollection
//}

//type Printer interface {
//	Print(w io.Writer)
//}

type CfgObj struct {
	Type    CfgType
	UUID    UUID
	Indent  int
	Align   int
	FileID  string
	Comment string
	Props   map[string]string
}

type CfgQuery struct {
	Keys []string
	RXs  []*regexp.Regexp
}

// Top level struct for managing collections of CfgObj
//type NagiosCfg struct {
//	Objs map[string]CfgMap // key by FileID
//}

//type CfgFile struct {
//	Path string
//	Objs CfgObjs
//}
