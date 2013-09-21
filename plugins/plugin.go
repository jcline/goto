package plugins

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"
)

type IRCMessage struct {
	Channel string
	Msg     string
	User    string
	When    time.Time
}

type Plugin interface {
	Match() *regexp.Regexp
	Event() chan IRCMessage
	Setup(chan IRCMessage)
}

type scrapePlugin interface {
	FindUri(*string) (*string, error)
	Write(*IRCMessage, *string) error
	Event() chan IRCMessage
}

type plugin struct {
	match *regexp.Regexp
	event chan IRCMessage
	write chan IRCMessage
}

func getMatch(re *regexp.Regexp, matchee *string) (match [][]string, err error) {
	match = nil
	match = re.FindAllStringSubmatch(*matchee, -1)
	if len(match) < 1 {
		err = errors.New("Could not match")
		return
	}
	return
}

func getFirstMatch(re *regexp.Regexp, matchee *string) (match *string, err error) {
	match = nil
	matches, err := getMatch(re, matchee)
	if err != nil {
		return
	}

	if len(matches[0]) < 2 {
		err = errors.New("Could not match")
		return
	}
	match = &matches[0][1]
	return
}

func scrapeAndSend(plug scrapePlugin) {
	var f = func(msg IRCMessage) {
		uri, err := plug.FindUri(&msg.Msg)
		if err != nil {
			log.Println(err)
			return
		}

		resp, err := http.Get(*uri)
		if err != nil {
			log.Println(err)
			return
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			log.Println(err)
			return
		}
		body := string(bodyBytes)

		err = plug.Write(&msg, &body)
		if err != nil {
			log.Println(err)
			return
		}
	}

	go func() {
		for msg := range plug.Event() {
			go f(msg)
		}
	}()
}
