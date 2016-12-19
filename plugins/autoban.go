package plugins

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"time"
	"unicode"
)

type AutobanMatches struct {
	Matcher *regexp.Regexp
	Time    string   `json:"time"`
	Regex   string   `json:"match"`
	Strip   []string `json:"strip"`
	Reason  string   `json:"reason"`
}

type Autoban struct {
	plugin
	autobanStats
	autoBans []AutobanMatches
	user     string
	self     string
}

type autobanStats struct {
	prior []*IRCMessage
}

type AutobanConf struct {
	Matches []AutobanMatches `json:"matchers"`
	User    string           `json:"notify_user"`
}

func (plug *Autoban) Setup(write chan IRCMessage, conf PluginConf) {
	plug.user = conf.Autoban.User
	plug.self = conf.UserName
	plug.write = write
	plug.event = make(chan IRCMessage, 1000)

	var buffer bytes.Buffer
	buffer.WriteString(`(?i:`)
	// Because range copies the elements, which is totally idiotic
	matches := conf.Autoban.Matches
	for i := 0; i < len(matches); i++ {
		buffer.WriteString(matches[i].Regex)
		if i != len(matches)-1 {
			buffer.WriteString(`|`)
		}
		matches[i].Matcher = regexp.MustCompile(matches[i].Regex)
	}
	buffer.WriteString(`)`)

	plug.match = regexp.MustCompile(buffer.String())

	plug.autoBans = conf.Autoban.Matches

	go plug.Action()
	return
}

func removeNongraphic(msg string) string {
	var buffer bytes.Buffer
	for _, char := range msg {
		if unicode.IsGraphic(char) && !unicode.IsSpace(char) {
			buffer.WriteRune(char)
		}
	}
	return buffer.String()
}

func removeColors(msg string) string {
	return regexp.MustCompile("\x03.").ReplaceAllLiteralString(msg, "")
}

func (plug Autoban) computeReasonAndTime(msg *IRCMessage, match bool, spam bool) (notify string, chanserv string) {
	time := ""
	reason := ":("
	if spam {
		time = "5"
		reason = "spam is bad and you should feel bad"
		notify = fmt.Sprintf(
			"Banning user `%s` with `%s` from `%s` for spam at `%s`",
			msg.User,
			msg.Mask,
			msg.Channel,
			msg.When)

		chanserv = fmt.Sprintf(
			"akick %s add *!*@%s !T %s %s | Laala b& '%s' for spam",
			msg.Channel,
			msg.Mask,
			time,
			reason,
			msg.User)
	}

	if match {
		for index, matcher := range plug.autoBans {
			cleaned := matcher.doCleanup(msg.Msg)
			if matcher.Matcher.MatchString(cleaned) {
				time = plug.autoBans[index].Time
				reason = plug.autoBans[index].Reason
			}
		}

		notify = fmt.Sprintf(
			"Banning user `%s` with `%s` from `%s` for `%s` at `%s`",
			msg.User,
			msg.Mask,
			msg.Channel,
			msg.Msg,
			msg.When)

		chanserv = fmt.Sprintf(
			"akick %s add *!*@%s !T %s %s | Laala b& '%s' for '%s'",
			msg.Channel,
			msg.Mask,
			time,
			reason,
			msg.User,
			msg.Msg)
	}

	return
}

func (plug Autoban) Ban(msg *IRCMessage, match bool, spam bool) {

	logMsg, banMsg := plug.computeReasonAndTime(msg, match, spam)

	log.Println(logMsg)
	if len(plug.user) > 0 {
		plug.write <- IRCMessage{
			Channel:   plug.user,
			Msg:       logMsg,
			User:      msg.User,
			When:      msg.When,
			Unlimited: true,
		}
	}

	if len(msg.Mask) < 3 {
		log.Println("msg.Mask too short to ban! %s", msg.Mask)
		return
	}

	log.Println(banMsg)
	plug.write <- IRCMessage{
		Channel:   "ChanServ",
		Msg:       banMsg,
		User:      msg.User,
		When:      msg.When,
		Unlimited: true,
	}
}

func (plug Autoban) Action() {
	for msg := range plug.event {
		go plug.Ban(&msg, true, false)
	}
}

func (matcher AutobanMatches) doCleanup(msg string) string {
	cleaned := msg
	for _, stripper := range matcher.Strip {
		switch stripper {
		case "colors":
			cleaned = removeColors(cleaned)
		case "nongraphic":
			cleaned = removeNongraphic(cleaned)
		}
	}
	return cleaned
}

func computeStats(msgs []*IRCMessage, last *IRCMessage) bool {
	count := 0
	time := time.Now().Add(time.Second * -5)
	for _, msg := range msgs {
		if msg.User == last.User && msg.When.After(time) {
			count += 1
		}
	}
	log.Println(last.User, count)
	if count >= 5 {
		return true
	}
	return false
}

func (plug *Autoban) Match(msg *IRCMessage) bool {
	if msg.Channel == plug.self {
		return false
	}

	plug.prior = append(plug.prior, msg)
	if len(plug.prior) > 110 {
		plug.prior = plug.prior[len(plug.prior)-100 : len(plug.prior)]
	}

	ban := computeStats(plug.prior, msg)
	if ban {
		plug.Ban(msg, false, true)
		return false
	}

	var cleaned string
	matched := false
	for _, matcher := range plug.autoBans {
		cleaned = matcher.doCleanup(msg.Msg)
		matched = plug.match.MatchString(cleaned)
		if matched {
			break
		}
	}

	if matched {
		plug.event <- *msg
	}

	return matched
}

func (plug Autoban) Event() chan IRCMessage {
	return plug.event
}
