package plugins

import (
	"testing"
)

func TestMatchUri(t *testing.T) {
	vimeo := Vimeo{}
	vimeo.Setup(make(chan IRCMessage))

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

func TestMatchTitle(t *testing.T) {
	vimeo := Vimeo{}
	vimeo.Setup(make(chan IRCMessage))

	html :=
		`
	<head><script src="http://a.vimeocdn.com/p/1.4.31/js/swfobject.v2.2.js"></script>
					<meta charset="utf-8">
	<meta name="viewport" content="width=1024,maximum-scale=1.0">
	<link rel="dns-prefetch" href="//player.vimeo.com">
	<link rel="dns-prefetch" href="http://av.vimeo.com">
			<link rel="dns-prefetch" href="//a.vimeocdn.com">
			<link rel="dns-prefetch" href="//b.vimeocdn.com">
	<meta property="fb:app_id" content="19884028963">
							<meta property="og:title" content="testestestest">
	`

	title, err := getFirstMatch(vimeo.title, &html)
	if err != nil {
		t.Error(err)
	}

	if *title != "testestestest" {
		t.Error(err)
	}
}

func TestMatchUser(t *testing.T) {
	vimeo := Vimeo{}
	vimeo.Setup(make(chan IRCMessage))

	html :=
		`

	<div itemprop="provider" itemscope itemtype="http://schema.org/Organization">
	<meta itemprop="name" content="Vimeo">
	<meta itemprop="url" content="https://vimeo.com">
	<meta itemprop="logo" content="http://a.vimeocdn.com/logo.svg">
	</div>
	<div itemprop="author" itemscope="" itemtype="http://schema.org/Person">
	<meta itemprop="name" content="test user">
	<a href="http://vimeo.com/ajkldfjsa" itemprop="url">
	<img class="portrait portrait_lg" src="http://b.vimeocdn.com/ps/406/566/4065663_75.jpg" alt="" itemprop="image">
	</a>
	</div>
	<div class="video_meta">
	<h1 itemprop="name">testestestset</h1>
	<div class="byline">
	from <a rel="author" href="/ajkldfjsa">test user</a> <a data-click-tracked="tracked" tabindex="-1" href="/plus" title="Learn more about Vimeo Plus" data-ga-event-click="button|plus_badge_click|plus"><span class="badge_plus">Plus</span></a>            <time data-title="Thursday, August 23, 2012 12:55 AM" data-timeago="1 year ago" datetime="2012-08-23T00:55:10-04:00" title="Thursday, August 23, 2012 12:55 AM">1 year ago</time>            

	<span class="meta">
	<span class="badge_rating safe">not yet rated</span>
	</span>
	</div>
	</div>
	`

	user, err := getFirstMatch(vimeo.user, &html)
	if err != nil {
		t.Error(err)
	}

	if *user != "test user" {
		t.Error(err)
	}
}
