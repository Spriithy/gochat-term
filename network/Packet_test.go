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
