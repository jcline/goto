package plugins

import (
	"html"
	"regexp"
)

type Vimeo struct {
	plugin
	spoiler, title, user *regexp.Regexp
}

func (plug *Vimeo) Setup(write chan IRCMessage) {
	plug.write = write
	plug.match = regexp.MustCompile(`(?:https?://|)(?:www\.|)(vimeo.com/\S+)`)
	plug.spoiler = regexp.MustCompile(`(?i)(.*spoil.*)`)
	//plug.title = regexp.MustCompile(`.*<title>(.+)(?: on Vimeo){1}</title>.*`)
	plug.title = regexp.MustCompile(`<[^>]*meta[^>]*property="og:title"[^>]*content="(.+)"[^>]*>`)
	//plug.user = regexp.MustCompile(`.*<div[^>]+(?: +itemtype="http://schema.org/Person" +|[^>]+?| +itemprop="author" +){3,}>(?s:.*?)<[^>]*meta[^>]*itemprop="name"[^>]*content="(.+)"[^>]*>.*`)
	//plug.user = regexp.MustCompile(`<[^>]*meta[^>]*itemprop="name"[^>]*content="(.+)"[^>]*>`)
	plug.user = regexp.MustCompile(`<a rel="author" href="/[^>]+?">(.+?)</a>`)
	plug.event = make(chan IRCMessage, 1000)
	scrapeAndSend(plug)
	return
}

func (plug *Vimeo) FindUri(candidate *string) (uri *string, err error) {
	uri, err = getFirstMatch(plug.match, candidate)
	if err != nil {
		uri = nil
		return
	}
	full := "http://" + *uri
	uri = &full
	return
}

func (plug Vimeo) Write(msg *IRCMessage, body *string) (err error) {
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
		plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
			Msg: "[Vimeo] " + html.UnescapeString(*title+" uploaded by "+*user)}
	} else {
		plug.write <- IRCMessage{Channel: msg.Channel, User: msg.User, When: msg.When,
			Msg: "[Vimeo] [[Title omitted due to possible spoilers]] uploaded by " + *user}
	}

	return
}

func (plug Vimeo) Match() *regexp.Regexp {
	return plug.match
}

func (plug Vimeo) Event() chan IRCMessage {
	return plug.event
}
