package plugins

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/jcline/DamerauLevenshteinDistance"
	//"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type entries []entry
type result struct {
	Entries entries `xml:"entry"`
}

type entry struct {
	Title    string `xml:"title"`
	Id       int    `xml:"id"`
	distance int
	search   string
	computed bool
}

type MalConf struct {
	User      string `json:"user"`
	Password  string `json:"password"`
	UserAgent string `json:"user_agent"`
}

func (r entries) Len() int {
	return len(r)
}

func (r entries) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r entries) Less(i, j int) bool {
	if !r[i].computed {
		r[i].distance = DamerauLevenshteinDistance.Distance(r[i].search, r[i].Title)
		r[i].computed = true
	}
	if !r[j].computed {
		r[j].distance = DamerauLevenshteinDistance.Distance(r[j].search, r[j].Title)
		r[j].computed = true
	}

	return r[i].distance < r[j].distance
}

type Mal struct {
	plugin
	spoiler, title, typeMatch *regexp.Regexp
	searchType, terms         *string
}

func (plug *Mal) Setup(write chan IRCMessage, conf PluginConf) {
	plug.write = write
	plug.match = regexp.MustCompile(`^!(?:anime|manga) (.{1,75})`)
	plug.spoiler = regexp.MustCompile(`(?i)(.*spoil.*)`)
	plug.title = regexp.MustCompile(`.*<title>(.+)</title>.*`)
	plug.typeMatch = regexp.MustCompile(`^!(anime|manga) .+`)
	plug.event = make(chan IRCMessage, 1000)

	malScrapeAndSend(plug, conf.Mal)
	return
}

func (plug *Mal) FindUri(candidate *string) (uri *string, err error) {
	terms, err := GetFirstMatch(plug.match, candidate)
	if err != nil {
		uri = nil
		return
	}

	plug.searchType, err = GetFirstMatch(plug.typeMatch, candidate)
	if err != nil {
		uri = nil
		return
	}

	full := "http://myanimelist.net/api/" + *plug.searchType + "/search.xml?q=" + url.QueryEscape(*terms)
	plug.terms = terms
	uri = &full
	fmt.Println(plug)
	return
}

func (plug Mal) Write(msg *IRCMessage, body *string) (err error) {
	fmt.Println(plug)
	if len(*body) < 10 {
		plug.write <- IRCMessage{Channel: msg.Channel,
			Msg:  "┐('～`；)┌    http://myanimelist.net/" + *plug.searchType + ".php?q=" + url.QueryEscape(*plug.terms),
			User: msg.User, When: msg.When}
		err = errors.New("No results")
		return
	}

	fmt.Printf("%v\n", *body)
	//unescaped := html.UnescapeString(*body)
	//fmt.Printf("%v\n", unescaped)
	var r result
	err = xml.Unmarshal([]byte(*body), &r)
	if err != nil {
		plug.write <- IRCMessage{Channel: msg.Channel,
			Msg:  "┐('～`；)┌    http://myanimelist.net/" + *plug.searchType + ".php?q=" + url.QueryEscape(*plug.terms),
			User: msg.User, When: msg.When}
		return
	}
	fmt.Printf("%v\n", r)

	var resultString = ""
	var nsfw = false
	reference, _ := GetFirstMatch(plug.match, &msg.Msg)

	for _, e := range r.Entries {
		//r[i].Title = html.UnescapeString(r[i].Title)
		e.search = *reference
		e.computed = false
	}
	sort.Sort(r.Entries)

	length := 2
	if len(r.Entries) < length {
		length = len(r.Entries)
	}
	for count, result := range r.Entries {
		if *plug.searchType == "anime" {
			/*
				class := result.Classification
				if class != "" {
					if strings.Contains(class, "Rx") ||
						strings.Contains(class, "R+") ||
						strings.Contains(class, "Hentai") {
						nsfw = true
					}
					class = " [Rating " + class + "]"
					nsfw = true
				} else {
					nsfw = true
				}
			*/
			nsfw = true

			resultString += result.Title + /*class + */ " http://myanimelist.net/" + *plug.searchType + "/" + strconv.Itoa(result.Id) + "  "
		} else {
			resultString += result.Title + " http://myanimelist.net/" + *plug.searchType + "/" + strconv.Itoa(result.Id) + "  "
			nsfw = true
		}
		if count >= length {
			break
		}
	}

	if nsfw {
		resultString = "NSFW " + resultString
	}

	if len(r.Entries) > 3 {
		resultString += "More: " + "http://myanimelist.net/" + *plug.searchType + ".php?q=" + url.QueryEscape(*plug.terms)
	}

	plug.write <- IRCMessage{Channel: msg.Channel, Msg: resultString, User: msg.User, When: msg.When}
	return
}

func (plug Mal) Match() *regexp.Regexp {
	return plug.match
}

func (plug Mal) Event() chan IRCMessage {
	return plug.event
}

func malScrapeAndSend(plug scrapePlugin, conf MalConf) {
	var f = func(msg IRCMessage) {
		uri, err := plug.FindUri(&msg.Msg)
		if err != nil {
			log.Println(err)
			return
		}

		client := &http.Client{}
		request, err := http.NewRequest("GET", *uri, nil)
		request.SetBasicAuth(conf.User, conf.Password)
		request.Header.Set("User-Agent", conf.UserAgent)
		resp, err := client.Do(request)
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

		err = plug.Write(&msg, malEmploysShittyProgrammers(body))
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

var HTML_ENTITIES = map[string]string{
	"&nbsp;":     "&#160;",
	"&iexcl;":    "¡",
	"&cent;":     "¢",
	"&pound;":    "£",
	"&curren;":   "¤",
	"&yen;":      "¥",
	"&brvbar;":   "¦",
	"&sect;":     "§",
	"&uml;":      "¨",
	"&copy;":     "©",
	"&ordf;":     "ª",
	"&laquo;":    "«",
	"&not;":      "¬",
	"&shy;":      "&#173;",
	"&reg;":      "®",
	"&macr;":     "¯",
	"&deg;":      "°",
	"&plusmn;":   "±",
	"&sup2;":     "²",
	"&sup3;":     "³",
	"&acute;":    "´",
	"&micro;":    "µ",
	"&para;":     "¶",
	"&middot;":   "·",
	"&cedil;":    "¸",
	"&sup1;":     "¹",
	"&ordm;":     "º",
	"&raquo;":    "»",
	"&frac14;":   "¼",
	"&frac12;":   "½",
	"&frac34;":   "¾",
	"&iquest;":   "¿",
	"&Agrave;":   "À",
	"&Aacute;":   "Á",
	"&Acirc;":    "Â",
	"&Atilde;":   "Ã",
	"&Auml;":     "Ä",
	"&Aring;":    "Å",
	"&AElig;":    "Æ",
	"&Ccedil;":   "Ç",
	"&Egrave;":   "È",
	"&Eacute;":   "É",
	"&Ecirc;":    "Ê",
	"&Euml;":     "Ë",
	"&Igrave;":   "Ì",
	"&Iacute;":   "Í",
	"&Icirc;":    "Î",
	"&Iuml;":     "Ï",
	"&ETH;":      "Ð",
	"&Ntilde;":   "Ñ",
	"&Ograve;":   "Ò",
	"&Oacute;":   "Ó",
	"&Ocirc;":    "Ô",
	"&Otilde;":   "Õ",
	"&Ouml;":     "Ö",
	"&times;":    "×",
	"&Oslash;":   "Ø",
	"&Ugrave;":   "Ù",
	"&Uacute;":   "Ú",
	"&Ucirc;":    "Û",
	"&Uuml;":     "Ü",
	"&Yacute;":   "Ý",
	"&THORN;":    "Þ",
	"&szlig;":    "ß",
	"&agrave;":   "à",
	"&aacute;":   "á",
	"&acirc;":    "â",
	"&atilde;":   "ã",
	"&auml;":     "ä",
	"&aring;":    "å",
	"&aelig;":    "æ",
	"&ccedil;":   "ç",
	"&egrave;":   "è",
	"&eacute;":   "é",
	"&ecirc;":    "ê",
	"&euml;":     "ë",
	"&igrave;":   "ì",
	"&iacute;":   "í",
	"&icirc;":    "î",
	"&iuml;":     "ï",
	"&eth;":      "ð",
	"&ntilde;":   "ñ",
	"&ograve;":   "ò",
	"&oacute;":   "ó",
	"&ocirc;":    "ô",
	"&otilde;":   "õ",
	"&ouml;":     "ö",
	"&divide;":   "÷",
	"&oslash;":   "ø",
	"&ugrave;":   "ù",
	"&uacute;":   "ú",
	"&ucirc;":    "û",
	"&uuml;":     "ü",
	"&yacute;":   "ý",
	"&thorn;":    "þ",
	"&yuml;":     "ÿ",
	"&OElig;":    "Œ",
	"&oelig;":    "œ",
	"&Scaron;":   "Š",
	"&scaron;":   "š",
	"&Yuml;":     "Ÿ",
	"&fnof;":     "ƒ",
	"&circ;":     "ˆ",
	"&tilde;":    "˜",
	"&Alpha;":    "Α",
	"&Beta;":     "Β",
	"&Gamma;":    "Γ",
	"&Delta;":    "Δ",
	"&Epsilon;":  "Ε",
	"&Zeta;":     "Ζ",
	"&Eta;":      "Η",
	"&Theta;":    "Θ",
	"&Iota;":     "Ι",
	"&Kappa;":    "Κ",
	"&Lambda;":   "Λ",
	"&Mu;":       "Μ",
	"&Nu;":       "Ν",
	"&Xi;":       "Ξ",
	"&Omicron;":  "Ο",
	"&Pi;":       "Π",
	"&Rho;":      "Ρ",
	"&Sigma;":    "Σ",
	"&Tau;":      "Τ",
	"&Upsilon;":  "Υ",
	"&Phi;":      "Φ",
	"&Chi;":      "Χ",
	"&Psi;":      "Ψ",
	"&Omega;":    "Ω",
	"&alpha;":    "α",
	"&beta;":     "β",
	"&gamma;":    "γ",
	"&delta;":    "δ",
	"&epsilon;":  "ε",
	"&zeta;":     "ζ",
	"&eta;":      "η",
	"&theta;":    "θ",
	"&iota;":     "ι",
	"&kappa;":    "κ",
	"&lambda;":   "λ",
	"&mu;":       "μ",
	"&nu;":       "ν",
	"&xi;":       "ξ",
	"&omicron;":  "ο",
	"&pi;":       "π",
	"&rho;":      "ρ",
	"&sigmaf;":   "ς",
	"&sigma;":    "σ",
	"&tau;":      "τ",
	"&upsilon;":  "υ",
	"&phi;":      "φ",
	"&chi;":      "χ",
	"&psi;":      "ψ",
	"&omega;":    "ω",
	"&thetasym;": "ϑ",
	"&upsih;":    "ϒ",
	"&piv;":      "ϖ",
	"&ensp;":     "&#8194;",
	"&emsp;":     "&#8195;",
	"&thinsp;":   "&#8201;",
	"&zwnj;":     "&#8204;",
	"&zwj;":      "&#8205;",
	"&lrm;":      "&#8206;",
	"&rlm;":      "&#8207;",
	"&ndash;":    "–",
	"&mdash;":    "—",
	"&lsquo;":    "‘",
	"&rsquo;":    "’",
	"&sbquo;":    "‚",
	"&ldquo;":    "“",
	"&rdquo;":    "”",
	"&bdquo;":    "„",
	"&dagger;":   "†",
	"&Dagger;":   "‡",
	"&bull;":     "•",
	"&hellip;":   "…",
	"&permil;":   "‰",
	"&prime;":    "′",
	"&Prime;":    "″",
	"&lsaquo;":   "‹",
	"&rsaquo;":   "›",
	"&oline;":    "‾",
	"&frasl;":    "⁄",
	"&euro;":     "€",
	"&image;":    "&#8465;",
	"&weierp;":   "&#8472;",
	"&real;":     "&#8476;",
	"&trade;":    "&#8482;",
	"&alefsym;":  "ℵ",
	"&larr;":     "←",
	"&uarr;":     "↑",
	"&rarr;":     "→",
	"&darr;":     "↓",
	"&harr;":     "↔",
	"&crarr;":    "↵",
	"&lArr;":     "⇐",
	"&uArr;":     "⇑",
	"&rArr;":     "⇒",
	"&dArr;":     "⇓",
	"&hArr;":     "⇔",
	"&forall;":   "∀",
	"&part;":     "∂",
	"&exist;":    "∃",
	"&empty;":    "∅",
	"&nabla;":    "∇",
	"&isin;":     "∈",
	"&notin;":    "∉",
	"&ni;":       "∋",
	"&prod;":     "∏",
	"&sum;":      "∑",
	"&minus;":    "−",
	"&lowast;":   "∗",
	"&radic;":    "√",
	"&prop;":     "∝",
	"&infin;":    "∞",
	"&ang;":      "∠",
	"&and;":      "∧",
	"&or;":       "∨",
	"&cap;":      "∩",
	"&cup;":      "∪",
	"&int;":      "∫",
	"&there4;":   "∴",
	"&sim;":      "∼",
	"&cong;":     "≅",
	"&asymp;":    "≈",
	"&ne;":       "≠",
	"&equiv;":    "≡",
	"&le;":       "≤",
	"&ge;":       "≥",
	"&sub;":      "⊂",
	"&sup;":      "⊃",
	"&nsub;":     "⊄",
	"&sube;":     "⊆",
	"&supe;":     "⊇",
	"&oplus;":    "⊕",
	"&otimes;":   "⊗",
	"&perp;":     "⊥",
	"&sdot;":     "⋅",
	"&vellip;":   "⋮",
	"&lceil;":    "⌈",
	"&rceil;":    "⌉",
	"&lfloor;":   "⌊",
	"&rfloor;":   "⌋",
	"&lang;":     "&#9001;",
	"&rang;":     "&#9002;",
	"&loz;":      "◊",
	"&spades;":   "♠",
	"&clubs;":    "♣",
	"&hearts;":   "♥",
	"&diams;":    "♦",
}

func malEmploysShittyProgrammers(badlyEscapedXml string) (correctlyEscapedXml *string) {
	for k, v := range HTML_ENTITIES {
		badlyEscapedXml = strings.Replace(badlyEscapedXml, k, v, -1)
	}
	correctlyEscapedXml = &badlyEscapedXml

	return
}
