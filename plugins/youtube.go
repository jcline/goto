package plugins

import (
	"html"
	"regexp"
)

type Youtube struct {
	plugin
	spoiler, title, user *regexp.Regexp
}


func (plug Youtube) Setup() (res Plugin) {
	plug.match = regexp.MustCompile(`(?:https?://|)(?:www\.|)(youtu(?:\.be|be\.com)/\S+)`)
	plug.spoiler = regexp.MustCompile(`(?i)(.*spoil.*)`)
	plug.title = regexp.MustCompile(`.*<title>(.+)(?: - YouTube){1}</title>.*`)
	plug.user = regexp.MustCompile(`.*<a[^>]+feature=watch[^>]+class="[^"]+yt-user-name[^>]+>([^<]+)</a>.*`)
	plug.event = make(chan IRCMessage, 1000)
	res = plug
	return
}

func (plug Youtube) FindUri(candidate *string) (uri *string, err error) {
	uri, err = getFirstMatch(plug.match, candidate)
	if err != nil {
		uri = nil
		return
	}
	full := "http://" + *uri
	uri = &full
	return
}

func (plug Youtube) Write(msg *IRCMessage, body *string) (outMsg *IRCMessage, err error) {
	outMsg = nil
	title, err := getFirstMatch(plug.title, body)
	if err != nil {
		return
	}

	user, err := getFirstMatch(plug.user, body)
	if err != nil {
		return
	}

	_, notFound := getFirstMatch(plug.spoiler, title)
	if notFound != nil {
		 outMsg = &IRCMessage{msg.Channel, "[YouTube] " + html.UnescapeString(*title + " uploaded by " + *user), msg.User, msg.When}
	} else {
		 outMsg = &IRCMessage{msg.Channel, "[YouTube] [[Title omitted due to possible spoilers]] uploaded by " + *user,
													msg.User, msg.When}
	}

	return
}

func (plug Youtube) Match() *regexp.Regexp {
	return plug.match
}

func (plug Youtube) Event() chan IRCMessage {
	return plug.event
}
