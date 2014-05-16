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
		{"www.youtube.com/v/O2rGTXHvPCQ", true},
		{"youtube.com/v/O2rGTXHvPCQ", true},
		{"youtu.be/O2rGTXHvPCQ", true},
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
		{"http://www.youtube.com/watch?v=YjvTy2u9qxE", "http://www.youtube.com/watch?v=YjvTy2u9qxE", false},
		{"test http://www.youtube.com/watch?v=YjvTy2u9qxE", "http://www.youtube.com/watch?v=YjvTy2u9qxE", false},
		{"test http://www.youtube.com/watch?v=YjvTy2u9qxE test", "http://www.youtube.com/watch?v=YjvTy2u9qxE", false},
		{"www.youtube.com/v/O2rGTXHvPCQ", "http://www.youtube.com/watch?v=O2rGTXHvPCQ", false},
		{"youtube.com/v/O2rGTXHvPCQ", "http://www.youtube.com/watch?v=O2rGTXHvPCQ", false},
		{"youtu.be/O2rGTXHvPCQ", "http://www.youtube.com/watch?v=O2rGTXHvPCQ", false},
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

func TestYoutubeMatchTitle(t *testing.T) {
	youtube := Youtube{}
	youtube.Setup(make(chan IRCMessage), PluginConf{})

	html := []struct {
		html, result string
		err          bool
	}{
		{"<title>Test - YouTube</title>", "Test", false},
		{"<title>Test - YouTube - YouTube</title>", "Test - YouTube", false},
		{"<title>Test - YouTube - Test - YouTube</title>", "Test - YouTube - Test", false},
		{"garbage<title>Test - YouTube</title>garbage", "Test", false},
		{"Test - YouTube</title>", "", true},
		{"<title>Test - YouTube</title><link rel=\"search\" type=\"application/opensearchdescription+xml\" href=\"http://www.youtube.com/opensearch?locale=en_US\" title=\"YouTube Video Search\"><link rel=\"shortcut icon\" href=\"http://s.ytimg.com/yts/img/favicon-yay.ico\" type=\"image/x-icon\">     <link rel=\"icon\" href=\"//s.ytimg.com/yts/img/favicon_32-yay.png\" sizes=\"32x32\"><link rel=\"canonical\" href=\"http://www.youtube.com/watch?v=vidya\"><link rel=\"alternate\" media=\"handheld\" href=\"http://m.youtube.com/watch?v=vidya\"><link rel=\"alternate\" media=\"only screen and (max-width: 640px)\" href=\"http://m.youtube.com/watch?v=vidya\"><link rel=\"shortlink\" href=\"http://youtu.be/vidya\">      <meta name=\"title\" content=\"Test\">", "Test", false},
	}

	for _, test := range html {
		result, err := GetFirstMatch(youtube.title, &test.html)
		errResult := err != nil
		if errResult != test.err {
			t.Error(test.html, "expected errResult to be", test.err, "but got", errResult, ":", err)
		}

		if result != nil && *result != test.result {
			t.Error(test.html, "expected", test.result, "but got", *result)
		}
	}

}

func TestYoutubeMatchUser(t *testing.T) {
	youtube := Youtube{}
	youtube.Setup(make(chan IRCMessage), PluginConf{})

	html := []struct {
		html, result string
		err          bool
	}{
		{"<a href=\"/user/AkiAkiSignal\" class=\"g-hovercard yt-uix-sessionlink yt-user-name \" data-sessionlink=\"feature=watch&amp;ei=7CAhU9X1OerKkwLz_YDYBw\" dir=\"ltr\" data-ytid=\"UCUxaYwnuATuhavEUdy3ELBQ\" data-name=\"watch\">Geoffrey Adams</a>", "Geoffrey Adams", false},
	}

	for _, test := range html {
		result, err := GetFirstMatch(youtube.user, &test.html)
		errResult := err != nil
		if errResult != test.err {
			t.Error(test.html, "expected errResult to be", test.err, "but got", errResult, ":", err)
		}

		if result != nil && *result != test.result {
			t.Error(test.html, "expected", test.result, "but got", *result)
		}
	}

}
