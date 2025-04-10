package slack

import (
	"fmt"
	llog "log"
	"os"

	"github.com/grafana/memo/cfg"
	"github.com/grafana/memo/parser"
	"github.com/grafana/memo/service"
	"github.com/grafana/memo/store"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	log "github.com/sirupsen/logrus"
)

// SlackService
type SlackService struct {
	// botToken xoxb-
	botToken string

	// appToken xapp-
	appToken string

	// parser takes the memo and extracts the values from it
	parser parser.Parser
	// store puts the memo in the defined store
	store store.Store

	// api client for talking to the slack API
	api *slack.Client
	// socket client connected to the slack websocket API
	socket *socketmode.Client

	// see https://github.com/nlopes/slack/issues/532
	// chanIdToNameCache
	chanIdToNameCache map[string]string
	// userIdToNameCache
	userIdToNameCache map[string]string
}

// Name returns the basic name of this service
func (s SlackService) Name() string {
	return "slack"
}

// chanIdToName gets and stores the channel name from the id
func (s *SlackService) chanIdToName(id string) string {
	name, ok := s.chanIdToNameCache[id]
	if ok {
		return name
	}
	g, err := s.api.GetConversationInfo(id, false)
	if err != nil {
		log.Debugf("GetChannelInfo error: %s", err.Error())
		return "null"
	}
	s.chanIdToNameCache[id] = g.Name
	return g.Name
}

// userIdToName gets and stores the user name from the id
func (s *SlackService) userIdToName(id string) string {
	name, ok := s.userIdToNameCache[id]
	if ok {
		return name
	}
	u, err := s.api.GetUserInfo(id)
	if err != nil {
		log.Errorf("GetUserInfo error, getting `%s`: %s (You probably don't have the `users:read` scope)", id, err.Error())
		return "null"
	}
	s.userIdToNameCache[id] = u.Name
	return u.Name
}

// handleMessage takes the slack message event and creates the memo, to pass
// to the store for storing the memo
func (s *SlackService) handleMessage(msg *slackevents.MessageEvent) error {
	ch := s.chanIdToName(msg.Channel)
	usr := s.userIdToName(msg.User)

	memo, err := s.parser.Parse(msg.Text)
	if err != nil {
		s.api.PostMessage(msg.Channel, slack.MsgOptionPostEphemeral(msg.User), slack.MsgOptionText(err.Error(), false))
		return err
	}

	// message is not for us, but not an error, just bail out
	if memo == nil {
		return nil
	}

	tags := []string{
		"author:" + usr,
		"chan:" + ch,
		"source: slack",
	}

	memo.BuildTags(tags)

	err = s.store.Save(*memo)
	if err != nil {
		s.api.PostMessage(msg.Channel, slack.MsgOptionPostEphemeral(msg.User), slack.MsgOptionText("memo failed: "+err.Error(), false))
		return err
	}

	s.api.PostMessage(msg.Channel, slack.MsgOptionPostEphemeral(msg.User), slack.MsgOptionText("Memo saved", false))
	return nil
}

// New creates a new instance of this service
func New(config cfg.Slack, parser parser.Parser, store store.Store) (service.Service, error) {
	s := SlackService{
		botToken: config.BotToken,
		appToken: config.AppToken,

		parser: parser,
		store:  store,

		chanIdToNameCache: make(map[string]string),
		userIdToNameCache: make(map[string]string),
	}

	s.api = slack.New(
		s.botToken,
		slack.OptionAppLevelToken(s.appToken),
	)

	s.socket = socketmode.New(
		s.api,
		socketmode.OptionDebug(false),
		socketmode.OptionLog(llog.New(os.Stdout, "slack", llog.Lshortfile|llog.LstdFlags)),
	)

	go func() {
		for evt := range s.socket.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				log.Info("Connecting to slack")
			case socketmode.EventTypeConnectionError:
				log.Errorf("Connection error: %v", evt)
			case socketmode.EventTypeConnected:
				log.Info("Socket connected")
			case socketmode.EventTypeDisconnect:
				log.Info("Socket disconnected")
			case socketmode.EventTypeIncomingError:
				log.Errorf("Connection error: %v", evt)
			case socketmode.EventTypeHello:
				log.Info("Received hello from slack, hi!")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					fmt.Printf("Ignored %+v\n", evt)

					continue
				}

				log.Debugf("Received event: %+v", eventsAPIEvent)

				s.socket.Ack(*evt.Request)
				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.MessageEvent:
						log.Debugf("Inner event: %+v", ev)
						s.handleMessage(ev)
					}
				}
			default:
				fmt.Fprintf(os.Stderr, "Unexpected event type received: %s\n", evt.Type)
			}
		}
	}()

	go func() {
		err := s.socket.Run()
		log.Fatalf("slack socket closed: %s", err.Error())
	}()

	return s, nil
}
