package main_test

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	main "github.com/matterpoll/matterpoll"
	"github.com/stretchr/testify/assert"
)

func TestParseInput(t *testing.T) {
	assert := assert.New(t)

	assert.Equal([]string{"A", "B", "C"}, main.ParseInput(`/matterpoll "A" "B" "C"`))
	assert.Equal([]string{"A", "B", "C"}, main.ParseInput(`/matterpoll  "A" "B" "C"`))
	assert.Equal([]string{}, main.ParseInput("/matterpoll "))
}

func TestPluginExecuteCommand(t *testing.T) {
	assert := assert.New(t)
	p := &main.MatterpollPlugin{}

	r, err := p.ExecuteCommand(&model.CommandArgs{
		Command: `/matterpoll "Question" "Answer 1" "Answer 2"`,
	})

	assert.Nil(err)
	assert.NotNil(r)
	assert.Equal(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, r.ResponseType)
	assert.Equal(`Matterpoll`, r.Username)
	assert.Equal([]*model.SlackAttachment{&model.SlackAttachment{
		AuthorName: `Matterpoll`,
		Text:       `Question`,
		Actions:    []*model.PostAction{&model.PostAction{Name: `Answer 1`}, &model.PostAction{Name: `Answer 2`}},
	}}, r.Attachments)
}

func TestPluginExecuteCommandHelp(t *testing.T) {
	assert := assert.New(t)
	p := &main.MatterpollPlugin{}

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

	p := &main.MatterpollPlugin{}
	p.OnActivate(api)
}
