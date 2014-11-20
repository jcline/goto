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
		{"</a><div class=\"yt-user-info\"><a href=\"/channel/UCU8tGtIuuuL2flF2S_AXibw\" class=\" yt-uix-sessionlink     spf-link  g-hovercard\" data-name=\"\" data-ytid=\"UCU8tGtIuuuL2flF2S_AXibw\" data-sessionlink=\"ei=dTRuVLPnDMSk-AO8k4HoBg\">Collision</a></div><span id=\"watch7-subscription-container\"><span class=\" yt-uix-button-subscription-container\"><button class=\"yt-uix-button yt-uix-button-size-default yt-uix-button-default yt-uix-button-has-icon no-icon-markup yt-can-buffer yt-uix-button yt-uix-button-size-default yt-uix-subscription-button yt-uix-button-has-icon yt-uix-button-subscribe-branded\" type=\"button\" onclick=\";return false;\" aria-live=\"polite\" aria-busy=\"false\" data-sessionlink=\"itct=\" data-channel-external-id=\"UCU8tGtIuuuL2flF2S_AXibw\" data-href=\"https://accounts.google.com/ServiceLogin?uilel=3&amp;service=youtube&amp;hl=en&amp;passive=true&amp;continue=http%3A%2F%2Fwww.youtube.com%2Fsignin%3Fnext%3D%252Fchannel%252FUCU8tGtIuuuL2flF2S_AXibw%26hl%3Den%26app%3Ddesktop%26feature%3Dsubscribe%26continue_action%3DQUFFLUhqbHZLWmo1WUJiU2lrTnFWOEVETkQ5ZHU5d1ZmZ3xBQ3Jtc0tualphdU5TNDd2Zy1uS3dXRFVYQTJrTVBJQ0JBUjhPX2hlcVk3blZDVWwtTnZoT1VsWlBGTVdrelcyUXNKZUhQWVdyVnB3YU9OYklFZzhudzJmdXhoZk1mcUpBZ0dueVVYSE5jUTJOTGpKN3doNThUZEZmM0l1M01IZVdZODM0TWRBZHlLT2VtdExaRWJablQwT1pfbF9Fanl5S1N3WUluaElITzhZOUxSazlTMlpSbkhxT1N2c0hPSmtTUXN6bmU3dUVzaUloWi1kazIwaTY0YVAtR0tieC1OSllB%26action_handle_signin%3Dtrue\" data-style-type=\"branded\"><span class=\"yt-uix-button-content\"><span class=\"subscribe-label\" aria-label=\"Subscribe\">Subscribe</span><span class=\"subscribed-label\" aria-label=\"Unsubscribe\">Subscribed</span><span class=\"unsubscribe-label\" aria-label=\"Unsubscribe\">Unsubscribe</span> </span></button><button class=\"yt-uix-button yt-uix-button-size-default yt-uix-button-default yt-uix-button-empty yt-uix-button-has-icon yt-uix-subscription-preferences-button\" type=\"button\" onclick=\";return false;\" aria-live=\"polite\" aria-busy=\"false\" aria-role=\"button\" aria-label=\"Subscription preferences\" data-channel-external-id=\"UCU8tGtIuuuL2flF2S_AXibw\"><span class=\"yt-uix-button-icon-wrapper\"><span class=\"yt-uix-button-icon yt-uix-button-icon-subscription-preferences yt-sprite\"></span></span></button><span class=\"yt-subscription-button-subscriber-count-branded-horizontal\" title=\"324\" tabindex=\"0\">324</span>", "Collision", false},
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
