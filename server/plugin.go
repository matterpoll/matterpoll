package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

var (
	endPollRoute = regexp.MustCompile(`/polls/([0-9a-z]+)/end`)
	voteRoute    = regexp.MustCompile(`/polls/([0-9a-z]+)/vote`)
)

const (
	responseIconURL     = `https://www.mattermost.org/wp-content/uploads/2016/04/icon.png`
	responseUsername    = `Matterpoll`
	commandGenericError = `Something went bad. Please try again later.`
)

type MatterpollPlugin struct {
	plugin.MattermostPlugin
	idGen IDGenerator
}

func (p *MatterpollPlugin) OnActivate() error {
	p.idGen = &PollIDGenerator{}
	return p.API.RegisterCommand(getCommand())
}

func (p *MatterpollPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch {
	case endPollRoute.MatchString(r.URL.Path):
		p.handleEndPoll(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (p *MatterpollPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	q, o := ParseInput(args.Command)
	if len(o) < 1 || q == "" {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, `We need input. Try `+"`"+`/matterpoll "Question" "Answer 1" "Answer 2"`+"`", nil), nil
	}
	poll := NewPoll(q, o)
	pollID := p.idGen.NewID()
	err := p.API.KVSet(pollID, poll.Encode())
	if err != nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, commandGenericError, nil), nil
	}
	return poll.ToCommandResponse(args.SiteURL, pollID), nil
}

func ParseInput(input string) (string, []string) {
	o := strings.TrimRight(strings.TrimLeft(strings.TrimSpace(strings.TrimPrefix(input, "/matterpoll")), "\""), "\"")
	if o == "" {
		return "", []string{}
	}
	s := strings.Split(o, "\" \"")
	return s[0], s[1:]
}

func (p *MatterpollPlugin) handleEndPoll(w http.ResponseWriter, r *http.Request) {
	resp := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: `Poll is done.`,
		},
	}
	id := endPollRoute.FindAllStringSubmatch(r.URL.Path, 1)[0][1]
	// TODO: Error handling
	_ = p.API.KVDelete(id)
	b, _ := json.Marshal(resp)

	w.Header().Set(`Content-Type`, `application/json`)
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func getCommandResponse(responseType, text string, attachments []*model.SlackAttachment) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     responseUsername,
		IconURL:      responseIconURL,
		Type:         model.POST_DEFAULT,
		Attachments:  attachments,
	}
}

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          `matterpoll`,
		DisplayName:      `Matterpoll`,
		Description:      `Polling feature by https://github.com/matterpoll/matterpoll`,
		AutoComplete:     true,
		AutoCompleteDesc: `Create a poll`,
		AutoCompleteHint: `[Question] [Answer 1] [Answer 2]...`,
	}
}
