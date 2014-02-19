package plugins

import (
	"testing"
)

func TestVimeoMatchUri(t *testing.T) {
	vimeo := Vimeo{}
	vimeo.Setup(make(chan IRCMessage), PluginConf{})

	uris := []struct {
		uri     string
		matched bool
	}{
		{"http://vimeo.com", false},
		{"http://vimeo.com/", false},
		{"http://vimeo.com/48055765", true},
		{"https://vimeo.com", false},
		{"https://vimeo.com/", false},
		{"https://vimeo.com/48055765", true},
	}

	for _, test := range uris {
		result := vimeo.match.MatchString(test.uri)
		if result != test.matched {
			t.Error(test.uri, "expected", test.matched, "but got", result)
		}
	}
}

