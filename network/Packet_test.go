package network

import (
	"testing"
)

var sourceStamps = map[string]bool{
	"15:42:13": true, "00:60:00": false, "01:17:31": true,
	"17;32:00": false, "abcdefgh": false, "-1:-0:18": false,
	"34:15:22": false, "  :  :  ": false, "06:18:20": true,
}

func TestTimeStampParsing(t *testing.T) {
	for s, v := range sourceStamps {
		if ts, err := ParseTimeStamp(s); err != nil {
			if v {
				t.Errorf("%s should have matched TimeStamp", s)
			} else {
				t.Log(s, "didn't match")
			}
		} else {
			t.Log(s, "matched", ts.String())
		}
	}
}

var sourceNames = map[string]bool{
	"joncena": true, " ekl19": false, "S€€D3R01": false,
	"\tbronks": false, "\nnewline": false, "-jon-connor18": false,
	"mickeal_\\phelps": false, "   ": false, "Springwater64": true,
}

func TestUsername(t *testing.T) {
	for s, v := range sourceNames {
		if ok, err := checkUsername(s); ok {
			if v {
				t.Logf("%s correctly matched", s)
				continue
			}
			t.Errorf("%s shouldn't have matched", s)
		} else {
			if v {
				t.Errorf("%s should have matched", s)
				t.Error(err)
				continue
			}
			t.Logf("%s correctly didn't match", s)
		}
	}
}
