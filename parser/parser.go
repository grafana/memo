package parser

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/grafana/memo"
	"github.com/raintank/dur"

	log "github.com/sirupsen/logrus"
)

// Parser
type Parser struct {
	// re
	re *regexp.Regexp

	// for mocking times in tests
	clock clock.Clock
}

// Parse takes a message and returns a memo with the fields extracted
func (p *Parser) Parse(message string) (*memo.Memo, error) {
	message = strings.TrimSpace(message)

	if len(message) == 0 {
		return nil, memo.ErrEmpty
	}

	m := memo.Memo{}

	// does not detect "isForUs" validly
	ok, err := p.isForUs(message)

	// regex match fail
	if err != nil {
		return nil, err
	}

	// not for us
	if !ok {
		return nil, nil
	}

	words := strings.Fields(message)
	if len(words) == 0 {
		return nil, memo.ErrEmpty
	}

	// [1:] strips out the "memo" trigger
	words, ts := p.extractTimestamp(words[1:])

	m.Date = ts
	m.Desc = strings.Join(words, " ")

	pos := len(words) - 1 // pos of the last word that is not a tag
	for strings.Contains(words[pos], ":") {
		pos--
		if pos < 0 {
			return &m, nil
		}
	}

	extraTags := words[pos+1:]
	m.BuildTags(extraTags)

	m.Desc = strings.Join(words[:pos+1], " ")

	return &m, nil
}

// isForUs returns if this message has been identified as a memo
func (p *Parser) isForUs(message string) (bool, error) {
	out := p.re.FindStringSubmatch(message)
	if len(out) == 0 {
		if strings.HasPrefix(message, "memo:") || strings.HasPrefix(message, "mrbot:") || strings.HasPrefix(message, "memobot:") {
			log.Debugf("A user seems to direct a message `%q` to us, but we don't understand it. so sending help message back", message)
			return false, errors.New("message could not be understood")
		}

		// we're in a channel. don't spam in it. the message was probably not meant for us.
		log.Tracef("Received message `%q`, not for us. ignoring", message)
		return false, nil
	}

	return true, nil
}

// SetClock allows injection of a benbjohnson/clock clock.Clock
// interface, for mocking within tests.
func (p *Parser) SetClock(clock clock.Clock) {
	p.clock = clock
}

// extractTimestamp takes a timestamp at the start of the memo
// (after memo phrase itself) written in RFC3339 format or time
// strings compatible with [https://pkg.go.dev/github.com/raintank/dur#ParseDuration]
// We make use of benbjohnson/clock to enable mocking of time for tests
func (p *Parser) extractTimestamp(words []string) ([]string, time.Time) {
	// parse time offset out of message (if applicable) and set timestamp
	ts := p.clock.Now().Add(-25 * time.Second)
	dur, err := dur.ParseDuration(words[0])
	if err == nil {
		ts = p.clock.Now().Add(-time.Duration(dur) * time.Second)
		words = words[1:]
	} else {
		parsed, err := time.Parse(time.RFC3339, words[0])
		if err == nil {
			ts = parsed
			words = words[1:]
		}
	}

	return words, ts
}

// New returns a new instance of Parser
func New() Parser {
	return Parser{
		re:    regexp.MustCompile("^memo(?::)? (.*)"),
		clock: clock.New(),
	}
}
