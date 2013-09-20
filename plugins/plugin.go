package plugins

import (
	"errors"
	"log"
	"io/ioutil"
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

func getFirstMatch(re *regexp.Regexp, matchee *string) (*string, error) {
	match := re.FindAllStringSubmatch(*matchee, -1)
	if len(match) < 1 {
		return nil, errors.New("Could not match")
	}
	if len(match[0]) < 2 {
		return nil, errors.New("Could not match")
	}
	return &match[0][1], nil
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

