package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPollIDGenerator struct {
	mock.Mock
}

func (m *MockPollIDGenerator) NewId() string {
	return `1234567890abcdefghij`
}

func TestParseInput(t *testing.T) {
	for name, test := range map[string]struct {
		Input            string
		ExpectedQuestion string
		ExpectedOptions  []string
	}{
		"Normal test": {
			Input:            `/matterpoll "A" "B" "C"`,
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
		},
		"Trim whitespace": {
			Input:            `/matterpoll   "A" "B" "C"`,
			ExpectedQuestion: "A",
			ExpectedOptions:  []string{"B", "C"},
		},
		"No options": {
			Input:            `/matterpoll  `,
			ExpectedQuestion: "",
			ExpectedOptions:  []string{},
		},
	} {
		t.Run(name, func(t *testing.T) {

			q, o := ParseInput(test.Input)
			assert.Equal(t, test.ExpectedQuestion, q)
			assert.Equal(t, test.ExpectedOptions, o)
		})
	}
}

func TestPluginExecuteCommand(t *testing.T) {
	assert := assert.New(t)

	siteURL := `https://example.org/`
	expectedPoll := Poll{Question: `Question`,
		Options: []*Option{
			{Answer: `Answer 1`},
			{Answer: `Answer 2`},
		},
	}

	idGen := new(MockPollIDGenerator)
	api := &plugintest.API{}
	api.On(`KVSet`, idGen.NewId(), expectedPoll.Encode()).Return(nil)
	defer api.AssertExpectations(t)
	p := &MatterpollPlugin{
		idGen: idGen,
	}
	p.SetAPI(api)

	r, err := p.ExecuteCommand(nil, &model.CommandArgs{
		Command: `/matterpoll "Question" "Answer 1" "Answer 2"`,
		SiteURL: siteURL,
	})

	assert.Nil(err)
	assert.NotNil(r)
	assert.Equal(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, r.ResponseType)
	assert.Equal(model.POST_DEFAULT, r.Type)
	assert.Equal(RESPONSE_USERNAME, r.Username)
	assert.Equal(RESPONSE_ICON_URL, r.IconURL)
	assert.Equal([]*model.SlackAttachment{{
		AuthorName: `Matterpoll`,
		Text:       `Question`,
		Actions: []*model.PostAction{
			{Name: `Answer 1`},
			{Name: `Answer 2`},
			{Name: `End Poll`, Integration: &model.PostActionIntegration{
				URL: fmt.Sprintf(`%s/plugins/%s/polls/%s/end`, siteURL, PluginId, p.idGen.NewId()),
			}},
		},
	}}, r.Attachments)
}

func TestPluginExecuteCommandHelp(t *testing.T) {
	api := &plugintest.API{}
	p := &MatterpollPlugin{}
	p.SetAPI(api)

	r, err := p.ExecuteCommand(nil, &model.CommandArgs{
		Command: `/matterpoll`,
	})

	assert.Nil(t, err)
	assertHelpResponse(t, r)
}

func TestPluginExecuteOneArgument(t *testing.T) {
	api := &plugintest.API{}
	p := &MatterpollPlugin{}
	p.SetAPI(api)

	r, err := p.ExecuteCommand(nil, &model.CommandArgs{
		Command: `/matterpoll "abcd"`,
	})
	assert.Nil(t, err)
	assertHelpResponse(t, r)
}

func assertHelpResponse(t *testing.T, r *model.CommandResponse) {
	assert := assert.New(t)

	assert.NotNil(r)
	assert.Equal(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, r.ResponseType)
	assert.Equal(`Matterpoll`, r.Username)
	assert.Equal(`We need input. Try `+"`"+`/matterpoll "Question" "Answer 1" "Answer 2"`+"`", r.Text)
	assert.Nil(r.Attachments)
}

func TestPluginOnActivate(t *testing.T) {
	api := &plugintest.API{}
	api.On("RegisterCommand", &model.Command{
		Trigger:          `matterpoll`,
		DisplayName:      `Matterpoll`,
		Description:      `Polling feature by https://github.com/matterpoll/matterpoll`,
		AutoComplete:     true,
		AutoCompleteDesc: `Create a poll`,
		AutoCompleteHint: `[Question] [Answer 1] [Answer 2]...`,
	}).Return(nil)
	defer api.AssertExpectations(t)
	p := &MatterpollPlugin{}
	p.SetAPI(api)

	err := p.OnActivate()
	assert.Nil(t, err)
}

func TestServeHTTP(t *testing.T) {
	for name, test := range map[string]struct {
		RequestURL         string
		KV                 bool
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"InvalidRequestURL": {
			RequestURL:         "/not_found",
			KV:                 false,
			ExpectedStatusCode: http.StatusNotFound,
			ExpectedHeader:     http.Header{},
		},
		"ValidEndPollRequest": {
			RequestURL:         fmt.Sprintf("/polls/%s/end", new(MockPollIDGenerator).NewId()),
			KV:                 true,
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader: map[string][]string{
				"Content-Type": []string{"application/json"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			if test.KV {
				api.On(`KVDelete`, new(MockPollIDGenerator).NewId()).Return(nil)
				defer api.AssertExpectations(t)
			}
			p := MatterpollPlugin{}
			p.SetAPI(api)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", test.RequestURL, nil)
			p.ServeHTTP(nil, w, r)
			assert.Equal(t, test.ExpectedStatusCode, w.Result().StatusCode)
			assert.Equal(t, test.ExpectedHeader, w.Result().Header)
		})
	}
}
