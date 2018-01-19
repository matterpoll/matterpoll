package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestParseInput(t *testing.T) {
	assert := assert.New(t)

	assert.Equal([]string{"A", "B", "C"}, ParseInput(`/matterpoll "A" "B" "C"`))
	assert.Equal([]string{"A", "B", "C"}, ParseInput(`/matterpoll  "A" "B" "C"`))
	assert.Equal([]string{}, ParseInput("/matterpoll "))
}

func TestPluginExecuteCommand(t *testing.T) {
	assert := assert.New(t)

	idGen := new(MockPollIDGenerator)
	p := &MatterpollPlugin{
		idGen: idGen,
	}

	r, err := p.ExecuteCommand(&model.CommandArgs{
		Command: `/matterpoll "Question" "Answer 1" "Answer 2"`,
		SiteURL: `http://localhost:8065`,
	})

	assert.Nil(err)
	assert.NotNil(r)
	assert.Equal(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, r.ResponseType)
	assert.Equal(`Matterpoll`, r.Username)
	assert.Equal([]*model.SlackAttachment{&model.SlackAttachment{
		AuthorName: `Matterpoll`,
		Text:       `Question`,
		Actions: []*model.PostAction{
			&model.PostAction{Name: `Answer 1`},
			&model.PostAction{Name: `Answer 2`},
			&model.PostAction{
				Name: `End Poll`,
				Integration: &model.PostActionIntegration{
					URL: `http://localhost:8065/plugins/matterpoll/polls/1234567890abcdefghij/end`,
				},
			},
		},
	}}, r.Attachments)
}

func TestPluginExecuteCommandHelp(t *testing.T) {
	assert := assert.New(t)
	p := &MatterpollPlugin{}

	r, err := p.ExecuteCommand(&model.CommandArgs{
		Command: `/matterpoll`,
	})

	assert.Nil(err)
	assert.NotNil(r)
	assert.Equal(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, r.ResponseType)
	assert.Equal(`Matterpoll`, r.Username)
	assert.Equal(`We need input. Try /matterpoll "Question" "Answer 1" "Answer 2"`, r.Text)
	assert.Nil(r.Attachments)
}

func TestPluginOnActivate(t *testing.T) {
	api := &plugintest.API{}
	api.On("RegisterCommand", &model.Command{
		DisplayName:      `Matterpoll`,
		Trigger:          `matterpoll`,
		AutoComplete:     true,
		AutoCompleteDesc: `Create a poll`,
		AutoCompleteHint: `[Question] [Answer 1] [Answer 2]...`,
	}).Return(nil)
	//defer api.AssertExpectations(t)

	p := &MatterpollPlugin{}
	p.OnActivate(api)
}

func TestServeHTTP(t *testing.T) {
	for name, test := range map[string]struct {
		RequestURL         string
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"InvalidRequestURL": {
			RequestURL:         "/not_found",
			ExpectedStatusCode: http.StatusNotFound,
			ExpectedHeader:     http.Header{},
		},
		"ValidEndPollRequest": {
			RequestURL:         "/polls/1234567890abcdefghij/end",
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader: map[string][]string{
				"Content-Type": []string{"application/json"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := MatterpollPlugin{}
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", test.RequestURL, nil)
			p.ServeHTTP(w, r)
			assert.Equal(t, test.ExpectedStatusCode, w.Result().StatusCode)
			assert.Equal(t, test.ExpectedHeader, w.Result().Header)
		})
	}
}

type MockPollIDGenerator struct {
	mock.Mock
}

func (m *MockPollIDGenerator) String() string {
	return `1234567890abcdefghij`
}
