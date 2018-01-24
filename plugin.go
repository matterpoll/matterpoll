package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/rpcplugin"
	"github.com/rs/xid"
)

var (
	endPollRoute = regexp.MustCompile(`/polls/([0-9a-z]+)/end`)
)

type MatterpollPlugin struct {
	idGen PollIDGenerator
}

type PollIDGenerator interface {
	String() string
}

func (p *MatterpollPlugin) OnActivate(api plugin.API) error {
	p.idGen = xid.New()

	return api.RegisterCommand(&model.Command{
		DisplayName:      `Matterpoll`,
		Trigger:          `matterpoll`,
		AutoComplete:     true,
		AutoCompleteDesc: `Create a poll`,
		AutoCompleteHint: `[Question] [Answer 1] [Answer 2]...`,
	})
}

func (p *MatterpollPlugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case endPollRoute.MatchString(r.URL.Path):
		p.handleEndPoll(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (p *MatterpollPlugin) handleEndPoll(w http.ResponseWriter, r *http.Request) {
	matches := endPollRoute.FindAllStringSubmatch(r.URL.Path, 1)
	id := matches[0][1]

	resp := &model.PostActionIntegrationResponse{
		Update: &model.Post{
			Message: fmt.Sprintf(`Poll #%s is done.`, id),
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

func (p *MatterpollPlugin) ExecuteCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	input := ParseInput(args.Command)
	if len(input) == 0 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Username:     `Matterpoll`,
			Text:         `We need input. Try /matterpoll "Question" "Answer 1" "Answer 2"`,
		}, nil
	}

	attachList := []*model.PostAction{}
	for index := 1; index < len(input); index++ {
		attachList = append(attachList, &model.PostAction{
			Name: input[index],
		})
	}
	attachList = append(attachList, &model.PostAction{
		Name: `End Poll`,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/matterpoll/polls/%s/end", args.SiteURL, p.idGen.String()),
		},
	})

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Username:     `Matterpoll`,
		Attachments: []*model.SlackAttachment{&model.SlackAttachment{
			AuthorName: `Matterpoll`,
			Text:       input[0],
			Actions:    attachList,
		},
		},
	}, nil
}

func ParseInput(input string) []string {
	o := strings.TrimRight(strings.TrimLeft(strings.TrimSpace(strings.TrimPrefix(input, "/matterpoll")), "\""), "\"")
	if o == "" {
		return []string{}
	}
	return strings.Split(o, "\" \"")
}

func main() {
	rpcplugin.Main(&MatterpollPlugin{})
}
