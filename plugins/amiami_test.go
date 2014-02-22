package plugins

import (
	"testing"
)

func TestAmiAmiMatchUri(t *testing.T) {
	amiami := AmiAmi{}
	amiami.Setup(make(chan IRCMessage), PluginConf{})

	uris := []struct {
		uri     string
		matched bool
	}{
		{"http://amiami.com", false},
		{"http://www.amiami.com", false},
		{"https://amiami.com", false},
		{"https://www.amiami.com", false},
		{"http://www.amiami.com/top/detail/detail?gcode=MED-CD2-12882", true},
		{"http://www.amiami.com/top/detail/detail?scode=FIG-IPN-1380-S001&page=top", true},
		{"http://www.amiami.com/top/detail/detail?gcode=GAME-0010785&page=top", true},
		{"http://www.amiami.com/top/detail/detail?gcode=FIG-DOL-6882&page=top", true},
		{"http://www.amiami.com/top/detail/review?scode=FIG-DOL-6882&page=top", true},
		{"http://www.amiami.com/top/detail/detail?gcode=LTD-FIG-00194&page=top", true},
		{"http://www.amiami.com catscatscats /detail?gcode=test", false},
	}

	for _, test := range uris {
		result := amiami.match.MatchString(test.uri)
		if result != test.matched {
			t.Error(test.uri, "expected", test.matched, "but got", result)
		}
	}
}
