package plugins

import (
	"testing"
)

func YoutubeTestMatchUri(t *testing.T) {
	youtube := Youtube{}
	youtube.Setup(make(chan IRCMessage))

	uris := []struct {
		uri     string
		matched bool
	}{
		{"http://youtube.com", false},
		{"http://www.youtube.com", false},
		{"http://youtu.be", false},
		{"https://youtube.com", false},
		{"https://www.youtube.com", false},
		{"https://youtu.be", false},
		{"http://youtube.com/v/O2rGTXHvPCQ", true},
		{"http://youtu.be/v/O2rGTXHvPCQ", true},
		{"http://www.youtube.com/v/O2rGTXHvPCQ", true},
		{"https://youtube.com/v/O2rGTXHvPCQ", true},
		{"https://youtu.be/v/O2rGTXHvPCQ", true},
		{"https://www.youtube.com/v/O2rGTXHvPCQ", true},
		{"http://youtube.com/v/O2rGTXHvPCQ&hl=en_US&fs=1&", true},
		{"http://youtu.be/v/O2rGTXHvPCQ&hl=en_US&fs=1&", true},
		{"http://www.youtube.com/v/O2rGTXHvPCQ&hl=en_US&fs=1&", true},
		{"https://youtube.com/v/O2rGTXHvPCQ&hl=en_US&fs=1&", true},
		{"https://youtu.be/v/O2rGTXHvPCQ&hl=en_US&fs=1&", true},
		{"https://www.youtube.com/v/O2rGTXHvPCQ&hl=en_US&fs=1&", true},
		{"2013/12/30 12:17:43 :user!~user@127.0.0.1 PRIVMSG #channel:user everyone knows https://www.youtube.com/v/O2rGTXHvPCQ&hl=en_US&fs=1& is the one true irc client", true},
	}

	for _, test := range uris {
		result := youtube.match.MatchString(test.uri)
		if result != test.matched {
			t.Error(test.uri, "expected", test.matched, "but got", result)
		}
	}
}

