package main

import (
	"encoding/json"
	"fmt"
	"log"
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

type MatterpollPlugin struct {
	plugin.MattermostPlugin
	idGen PollIDGenerator
}

type PollIDGenerator interface {
	String() string
}

func (p *MatterpollPlugin) OnActivate() error {
	p.idGen = xid.New()
	return p.API.RegisterCommand(&model.Command{
		Trigger:          `matterpoll`,
		AutoComplete:     true,
		AutoCompleteDesc: `Create a poll`,
		AutoCompleteHint: `[Question] [Answer 1] [Answer 2]...`,
	})
}

func (p *MatterpollPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch {
	case endPollRoute.MatchString(r.URL.Path):
		p.handleEndPoll(w, r)
	default:
		//fmt.Fprintf(w, "Hello, world!")
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (p *MatterpollPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	input := ParseInput(args.Command)
	if len(input) < 2 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Username:     `Matterpoll`,
			Text:         `We need input. Try ` + "`" + `/matterpoll "Question" "Answer 1" "Answer 2"` + "`",
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
			URL: fmt.Sprintf(`%s/plugins/%s/polls/%s/end`, args.SiteURL, PluginId, p.idGen.String()),
		},
	})
	log.Printf("attachList: %#+v\n", attachList)

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Username:     `Matterpoll`,
		Attachments: []*model.SlackAttachment{{
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
