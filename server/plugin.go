package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/rs/xid"
)

var (
	endPollRoute = regexp.MustCompile(`/polls/([0-9a-z]+)/end`)
)

const (
	RESPONSE_ICON_URL = `https://www.mattermost.org/wp-content/uploads/2016/04/icon.png`
	RESPONSE_USERNAME = `Matterpoll`
)

type MatterpollPlugin struct {
	plugin.MattermostPlugin
	idGen PollIDGenerator
}

type PollIDGenerator interface {
	String() string
}

func (p *MatterpollPlugin) OnActivate() error {
	p.idGen = xid.New()
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
	input := ParseInput(args.Command)
	if len(input) < 2 {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, `We need input. Try `+"`"+`/matterpoll "Question" "Answer 1" "Answer 2"`+"`", nil), nil
	}
	actions := []*model.PostAction{}
	for index := 1; index < len(input); index++ {
		actions = append(actions, &model.PostAction{
			Name: input[index],
		})
	}
	actions = append(actions, &model.PostAction{
		Name: `End Poll`,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf(`%s/plugins/%s/polls/%s/end`, args.SiteURL, PluginId, p.idGen.String()),
		},
	})

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, ``, []*model.SlackAttachment{{
		AuthorName: `Matterpoll`,
		Text:       input[0],
		Actions:    actions,
	},
	}), nil
}

func ParseInput(input string) []string {
	o := strings.TrimRight(strings.TrimLeft(strings.TrimSpace(strings.TrimPrefix(input, "/matterpoll")), "\""), "\"")
	if o == "" {
		return []string{}
	}
	return strings.Split(o, "\" \"")
}

func (p *MatterpollPlugin) handleEndPoll(w http.ResponseWriter, r *http.Request) {
	resp := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: fmt.Sprintf(`Poll is done.`),
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `Cannot encode end poll response: %v`, err)
		return
	}
	w.Header().Set(`Content-Type`, `application/json`)
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func getCommandResponse(responseType, text string, attachments []*model.SlackAttachment) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     RESPONSE_USERNAME,
		IconURL:      RESPONSE_ICON_URL,
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
