package plugin

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/undefinedlabs/go-mpatch"

	root "github.com/matterpoll/matterpoll"
	"github.com/matterpoll/matterpoll/server/poll"
	"github.com/matterpoll/matterpoll/server/store/mockstore"
	"github.com/matterpoll/matterpoll/server/utils/testutils"
)

func TestPluginExecuteCommand(t *testing.T) {
	const trigger = "poll"
	const helpText = "To create a poll with the answer options \"Yes\" and \"No\" type `/poll \"Question\"`\n" +
		"You can customize the options by typing `/poll \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"`\n" +
		"Poll Settings provider further customization, e.g. `/poll \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --progress --anonymous`. The available Poll Settings are:\n" +
		"- `--anonymous`: Don't show who voted for what when the poll ends\n" +
		"- `--anonymous-creator`: Don't show author of the poll\n" +
		"- `--progress`: During the poll, show how many votes each answer option got\n" +
		"- `--public-add-option`: Allow all users to add additional options\n" +
		"- `--votes=X`: Allow users to vote for X options. Default is 1. If X is 0, users have an unlimited amount of votes."
	triggerID := model.NewId()
	rootID := model.NewId()

	createPollDialog := model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       fmt.Sprintf("/plugins/%s/api/v1/polls/create", root.Manifest.Id),
		Dialog: model.Dialog{
			CallbackId: rootID,
			Title:      "Create Poll",
			IconURL:    fmt.Sprintf(responseIconURL, testutils.GetSiteURL(), root.Manifest.Id),
			Elements: []model.DialogElement{{
				DisplayName: "Question",
				Name:        "question",
				Type:        "text",
				SubType:     "text",
			}, {
				DisplayName: "Option 1",
				Name:        "option1",
				Type:        "text",
				SubType:     "text",
			}, {
				DisplayName: "Option 2",
				Name:        "option2",
				Type:        "text",
				SubType:     "text",
			}, {
				DisplayName: "Option 3",
				Name:        "option3",
				Type:        "text",
				SubType:     "text",
				Optional:    true,
			}, {
				DisplayName: "Number of Votes",
				Name:        "setting-multi",
				Type:        "text",
				SubType:     "number",
				Default:     "1",
				HelpText:    "The number of options that a user can vote on. 0 means that users can vote for all options even after adding options.",
				Optional:    false,
			}, {
				DisplayName: "Anonymous",
				Name:        "setting-anonymous",
				Type:        "bool",
				Placeholder: "Don't show who voted for what when the poll ends",
				Default:     "true",
				Optional:    true,
			}, {
				DisplayName: "Anonymous creator",
				Name:        "setting-anonymous-creator",
				Type:        "bool",
				Placeholder: "Don't show author of the poll",
				Default:     "false",
				Optional:    true,
			}, {
				DisplayName: "Progress",
				Name:        "setting-progress",
				Type:        "bool",
				Placeholder: "During the poll, show how many votes each answer option got",
				Default:     "false",
				Optional:    true,
			}, {
				DisplayName: "Public Add Option",
				Name:        "setting-public-add-option",
				Type:        "bool",
				Placeholder: "Allow all users to add additional options",
				Default:     "true",
				Optional:    true,
			}},
			SubmitLabel: "Create",
		},
	}

	converter := func(userID string) (string, *model.AppError) {
		switch userID {
		case "userID1":
			return "@jhDoe", nil
		default:
			return "", &model.AppError{}
		}
	}

	for name, test := range map[string]struct {
		SetupAPI     func(*plugintest.API) *plugintest.API
		SetupStore   func(*mockstore.Store) *mockstore.Store
		Command      string
		ExpectedText string
		ShouldError  bool
	}{
		"No argument": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("OpenInteractiveDialog", createPollDialog).Return(nil)
				return api
			},
			SetupStore:   func(store *mockstore.Store) *mockstore.Store { return store },
			Command:      fmt.Sprintf("/%s", trigger),
			ExpectedText: "",
		},
		"No argument, OpenInteractiveDialog fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("OpenInteractiveDialog", createPollDialog).Return(&model.AppError{})
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			SetupStore:   func(store *mockstore.Store) *mockstore.Store { return store },
			Command:      fmt.Sprintf("/%s", trigger),
			ExpectedText: commandErrorGeneric.Other,
		},
		"Help text": {
			SetupAPI:     func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:   func(store *mockstore.Store) *mockstore.Store { return store },
			Command:      fmt.Sprintf("/%s help", trigger),
			ExpectedText: helpText,
		},
		"Two arguments": {
			SetupAPI:    func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:  func(store *mockstore.Store) *mockstore.Store { return store },
			Command:     fmt.Sprintf("/%s \"Question\" \"Just one option\"", trigger),
			ShouldError: true,
		},
		"Just question": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 3)...).Return()

				post := &model.Post{
					UserId:    testutils.GetBotUserID(),
					ChannelId: "channelID1",
					RootId:    rootID,
					Type:      MatterpollPostType,
					Props: model.StringInterface{
						"poll_id": testutils.GetPollID(),
					},
				}
				actions := testutils.GetPollTwoOptions().ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe")
				model.ParseSlackAttachment(post, actions)

				rPost := post.Clone()
				rPost.Id = "postID1"

				api.On("CreatePost", post).Return(rPost, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Insert", testutils.GetPollTwoOptions()).Return(nil)
				return store
			},
			Command: fmt.Sprintf("/%s \"Question\"", trigger),
		},
		"Just question and setting anonymous creator": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 3)...).Return()

				post := &model.Post{
					UserId:    testutils.GetBotUserID(),
					ChannelId: "channelID1",
					RootId:    rootID,
					Type:      MatterpollPostType,
					Props: model.StringInterface{
						"poll_id": testutils.GetPollID(),
					},
				}
				poll := testutils.GetPollTwoOptionsWithSettings(poll.Settings{AnonymousCreator: true, MaxVotes: 1})
				actions := poll.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "")
				model.ParseSlackAttachment(post, actions)

				rPost := post.Clone()
				rPost.Id = "postID1"

				api.On("CreatePost", post).Return(rPost, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollTwoOptionsWithSettings(poll.Settings{AnonymousCreator: true, MaxVotes: 1})
				store.PollStore.On("Insert", poll).Return(nil)
				return store
			},
			Command: fmt.Sprintf("/%s \"Question\" --anonymous-creator", trigger),
		},
		"Just question, CreatePost fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)

				post := &model.Post{
					UserId:    testutils.GetBotUserID(),
					ChannelId: "channelID1",
					RootId:    rootID,
					Type:      MatterpollPostType,
					Props: model.StringInterface{
						"poll_id": testutils.GetPollID(),
					},
				}
				actions := testutils.GetPollTwoOptions().ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe")
				model.ParseSlackAttachment(post, actions)
				api.On("CreatePost", post).Return(nil, &model.AppError{})
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			SetupStore:   func(store *mockstore.Store) *mockstore.Store { return store },
			Command:      fmt.Sprintf("/%s \"Question\"", trigger),
			ExpectedText: commandErrorGeneric.Other,
		},
		"With 4 arguments": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 3)...).Return()

				post := &model.Post{
					UserId:    testutils.GetBotUserID(),
					ChannelId: "channelID1",
					RootId:    rootID,
					Type:      MatterpollPostType,
					Props: model.StringInterface{
						"poll_id": testutils.GetPollID(),
					},
				}
				actions := testutils.GetPoll().ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe")
				model.ParseSlackAttachment(post, actions)

				rPost := post.Clone()
				rPost.Id = "postID1"

				api.On("CreatePost", post).Return(rPost, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				store.PollStore.On("Insert", testutils.GetPoll()).Return(nil)
				return store
			},
			Command: fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
		},
		"With 4 arguments and setting progress": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe", Username: "jhDoe"}, nil)
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 3)...).Return()

				post := &model.Post{
					UserId:    testutils.GetBotUserID(),
					ChannelId: "channelID1",
					RootId:    rootID,
					Type:      MatterpollPostType,
					Props: model.StringInterface{
						"poll_id": testutils.GetPollID(),
					},
				}
				poll := testutils.GetPollWithSettings(poll.Settings{Progress: true, MaxVotes: 1})
				actions := poll.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe")
				model.ParseSlackAttachment(post, actions)
				post.AddProp("card", poll.ToCard(testutils.GetBundle(), converter))

				rPost := post.Clone()
				rPost.Id = "postID1"

				api.On("CreatePost", post).Return(rPost, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithSettings(poll.Settings{Progress: true, MaxVotes: 1})
				store.PollStore.On("Insert", poll).Return(nil)
				return store
			},
			Command: fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --progress", trigger),
		},
		"With 4 arguments and multi setting": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 3)...).Return()

				post := &model.Post{
					UserId:    testutils.GetBotUserID(),
					ChannelId: "channelID1",
					RootId:    rootID,
					Type:      MatterpollPostType,
					Props: model.StringInterface{
						"poll_id": testutils.GetPollID(),
					},
				}
				poll := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 3})
				actions := poll.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe")
				model.ParseSlackAttachment(post, actions)

				rPost := post.Clone()
				rPost.Id = "postID1"

				api.On("CreatePost", post).Return(rPost, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 3})
				store.PollStore.On("Insert", poll).Return(nil)
				return store
			},
			Command: fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --votes=3", trigger),
		},
		"With 4 arguments and multi setting (--votes=0)": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 3)...).Return()

				post := &model.Post{
					UserId:    testutils.GetBotUserID(),
					ChannelId: "channelID1",
					RootId:    rootID,
					Type:      MatterpollPostType,
					Props: model.StringInterface{
						"poll_id": testutils.GetPollID(),
					},
				}
				poll := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 0})
				actions := poll.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe")
				model.ParseSlackAttachment(post, actions)

				rPost := post.Clone()
				rPost.Id = "postID1"

				api.On("CreatePost", post).Return(rPost, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithSettings(poll.Settings{MaxVotes: 0})
				store.PollStore.On("Insert", poll).Return(nil)
				return store
			},
			Command: fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --votes=0", trigger),
		},
		"With 4 arguments and setting anonymous and progress": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe", Username: "jhDoe"}, nil)
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 3)...).Return()

				post := &model.Post{
					UserId:    testutils.GetBotUserID(),
					ChannelId: "channelID1",
					RootId:    rootID,
					Type:      MatterpollPostType,
					Props: model.StringInterface{
						"poll_id": testutils.GetPollID(),
					},
				}
				poll := testutils.GetPollWithSettings(poll.Settings{Progress: true, Anonymous: true, MaxVotes: 1})
				actions := poll.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe")
				model.ParseSlackAttachment(post, actions)
				post.AddProp("card", poll.ToCard(testutils.GetBundle(), converter))

				rPost := post.Clone()
				rPost.Id = "postID1"

				api.On("CreatePost", post).Return(rPost, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPollWithSettings(poll.Settings{Progress: true, Anonymous: true, MaxVotes: 1})
				store.PollStore.On("Insert", poll).Return(nil)
				return store
			},
			Command: fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --anonymous --progress", trigger),
		},
		"Store.Save fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(&model.User{FirstName: "John", LastName: "Doe"}, nil)
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return()

				post := &model.Post{
					UserId:    testutils.GetBotUserID(),
					ChannelId: "channelID1",
					RootId:    rootID,
					Type:      MatterpollPostType,
					Props: model.StringInterface{
						"poll_id": testutils.GetPollID(),
					},
				}
				poll := testutils.GetPoll()
				actions := poll.ToPostActions(testutils.GetBundle(), root.Manifest.Id, "John Doe")
				model.ParseSlackAttachment(post, actions)

				rPost := post.Clone()
				rPost.Id = "postID1"

				api.On("CreatePost", post).Return(rPost, nil)
				return api
			},
			SetupStore: func(store *mockstore.Store) *mockstore.Store {
				poll := testutils.GetPoll()
				poll.PostID = "postID1"
				store.PollStore.On("Insert", poll).Return(errors.New(""))
				return store
			},
			Command:      fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
			ExpectedText: commandErrorGeneric.Other,
		},
		"GetUser fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetUser", "userID1").Return(nil, &model.AppError{})
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 3)...).Return()
				return api
			},
			SetupStore:   func(store *mockstore.Store) *mockstore.Store { return store },
			Command:      fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\"", trigger),
			ExpectedText: commandErrorGeneric.Other,
		},
		"Invalid setting": {
			SetupAPI:    func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:  func(store *mockstore.Store) *mockstore.Store { return store },
			Command:     fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --unknownOption", trigger),
			ShouldError: true,
		},
		"Invalid multi setting, ": {
			SetupAPI:    func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:  func(store *mockstore.Store) *mockstore.Store { return store },
			Command:     fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --votes=4", trigger),
			ShouldError: true,
		},
		"Invalid multi setting, invalid number": {
			SetupAPI:    func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:  func(store *mockstore.Store) *mockstore.Store { return store },
			Command:     fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --votes=-1", trigger),
			ShouldError: true,
		},
		"Invalid multi setting, exceed number": {
			SetupAPI:    func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:  func(store *mockstore.Store) *mockstore.Store { return store },
			Command:     fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --votes=4", trigger),
			ShouldError: true,
		},
		"Invalid multi setting, not number": {
			SetupAPI:    func(api *plugintest.API) *plugintest.API { return api },
			SetupStore:  func(store *mockstore.Store) *mockstore.Store { return store },
			Command:     fmt.Sprintf("/%s \"Question\" \"Answer 1\" \"Answer 2\" \"Answer 3\" --votes=abc", trigger),
			ShouldError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			api.On("GetUser", "userID1").Return(&model.User{Username: "user1"}, nil)
			if test.ExpectedText != "" {
				ephemeralPost := &model.Post{
					ChannelId: "channelID1",
					RootId:    rootID,
					UserId:    testutils.GetBotUserID(),
					Message:   test.ExpectedText,
				}
				api.On("SendEphemeralPost", "userID1", ephemeralPost).Return(nil)
			}
			defer api.AssertExpectations(t)
			store := test.SetupStore(&mockstore.Store{})
			defer store.AssertExpectations(t)
			p := setupTestPlugin(t, api, store)
			p.configuration.Trigger = trigger
			p.configuration.DefaultSettings = map[string]bool{"anonymous": true, "publicAddOption": true}

			patch1, _ := mpatch.PatchMethod(model.GetMillis, func() int64 { return 1234567890 })
			patch2, _ := mpatch.PatchMethod(model.NewId, testutils.GetPollID)
			defer func() { require.NoError(t, patch1.Unpatch()) }()
			defer func() { require.NoError(t, patch2.Unpatch()) }()

			r, err := p.ExecuteCommand(nil, &model.CommandArgs{
				Command:   test.Command,
				UserId:    "userID1",
				ChannelId: "channelID1",
				RootId:    rootID,
				TriggerId: triggerID,
			})

			assert.Equal(&model.CommandResponse{}, r)
			if test.ShouldError {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}
		})
	}
}
