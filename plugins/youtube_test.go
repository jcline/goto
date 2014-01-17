package plugins

import (
	"testing"
)

func TestYoutubeMatchUri(t *testing.T) {
	youtube := Youtube{}
	youtube.Setup(make(chan IRCMessage), PluginConf{})

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
		{"http://youtube.com/channel/UCeBMccz-PDZf6OB4aV6a3eA?feature=g-high", false},
		{"http://www.youtube.com/channel/UCeBMccz-PDZf6OB4aV6a3eA?feature=g-high", false},
		{"http://youtu.be/channel/UCeBMccz-PDZf6OB4aV6a3eA?feature=g-high", false},
		{"https://youtube.com/channel/UCeBMccz-PDZf6OB4aV6a3eA?feature=g-high", false},
		{"https://www.youtube.com/channel/UCeBMccz-PDZf6OB4aV6a3eA?feature=g-high", false},
		{"https://youtu.be/channel/UCeBMccz-PDZf6OB4aV6a3eA?feature=g-high", false},
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
		{"https://www.youtube.com/watch?feature=player_detailpage&v=WiDgeNBsMEA", true},
		{"https://www.youtube.com/watch?v=E-5cB2rUnvk", true},
	}

	for _, test := range uris {
		result := youtube.match.MatchString(test.uri)
		if result != test.matched {
			t.Error(test.uri, "expected", test.matched, "but got", result)
		}
	}
}

func TestYoutubeFindUri(t *testing.T) {
	youtube := Youtube{}
	youtube.Setup(make(chan IRCMessage), PluginConf{})

	uris := []struct {
		uri    string
		result string
		err    bool
	}{
		{"https://www.youtube.com/v/O2rGTXHvPCQ&hl=en_US&fs=1&", "http://www.youtube.com/watch?v=O2rGTXHvPCQ&hl=en_US&fs=1&", false},
		{"https://www.youtube.com/watch?v=-5wpm-gesOY", "http://www.youtube.com/watch?v=-5wpm-gesOY", false},
		{"https://youtu.be/-5wpm-gesOY", "http://www.youtube.com/watch?v=-5wpm-gesOY", false},
		{"http://www.youtube.com/v/O2rGTXHvPCQ&hl=en_US&fs=1&", "http://www.youtube.com/watch?v=O2rGTXHvPCQ&hl=en_US&fs=1&", false},
		{"http://www.youtube.com/watch?v=-5wpm-gesOY", "http://www.youtube.com/watch?v=-5wpm-gesOY", false},
		{"http://youtu.be/-5wpm-gesOY", "http://www.youtube.com/watch?v=-5wpm-gesOY", false},
		{"https://www.youtube.com/watch?feature=player_detailpage&v=WiDgeNBsMEA", "http://www.youtube.com/watch?v=WiDgeNBsMEA", false},
		{"https://www.youtube.com/watch?v=E-5cB2rUnvk", "http://www.youtube.com/watch?v=E-5cB2rUnvk", false},
		{"cat", "", true},
	}

	for _, test := range uris {
		result, err := youtube.FindUri(&test.uri)
		errResult := err != nil
		if errResult != test.err {
			t.Error(test.uri, "expected errResult to be", test.err, "but got", errResult, ":", err)
		}

		if result != nil && *result != test.result {
			t.Error(test.uri, "expected", test.result, "but got", *result)
		}
	}
}
