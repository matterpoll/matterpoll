// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/store/sqlstore"
	"github.com/mattermost/mattermost-server/utils"
)

func TestCreateChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam

	channel := &model.Channel{DisplayName: "Test API Name", Name: GenerateTestChannelName(), Type: model.CHANNEL_OPEN, TeamId: team.Id}
	private := &model.Channel{DisplayName: "Test API Name", Name: GenerateTestChannelName(), Type: model.CHANNEL_PRIVATE, TeamId: team.Id}

	rchannel, resp := Client.CreateChannel(channel)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	if rchannel.Name != channel.Name {
		t.Fatal("names did not match")
	}

	if rchannel.DisplayName != channel.DisplayName {
		t.Fatal("display names did not match")
	}

	if rchannel.TeamId != channel.TeamId {
		t.Fatal("team ids did not match")
	}

	rprivate, resp := Client.CreateChannel(private)
	CheckNoError(t, resp)

	if rprivate.Name != private.Name {
		t.Fatal("names did not match")
	}

	if rprivate.Type != model.CHANNEL_PRIVATE {
		t.Fatal("wrong channel type")
	}

	if rprivate.CreatorId != th.BasicUser.Id {
		t.Fatal("wrong creator id")
	}

	_, resp = Client.CreateChannel(channel)
	CheckErrorMessage(t, resp, "store.sql_channel.save_channel.exists.app_error")
	CheckBadRequestStatus(t, resp)

	direct := &model.Channel{DisplayName: "Test API Name", Name: GenerateTestChannelName(), Type: model.CHANNEL_DIRECT, TeamId: team.Id}
	_, resp = Client.CreateChannel(direct)
	CheckErrorMessage(t, resp, "api.channel.create_channel.direct_channel.app_error")
	CheckBadRequestStatus(t, resp)

	Client.Logout()
	_, resp = Client.CreateChannel(channel)
	CheckUnauthorizedStatus(t, resp)

	userNotOnTeam := th.CreateUser()
	Client.Login(userNotOnTeam.Email, userNotOnTeam.Password)

	_, resp = Client.CreateChannel(channel)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.CreateChannel(private)
	CheckForbiddenStatus(t, resp)

	th.LoginBasic()

	// Check permissions with policy config changes
	isLicensed := utils.IsLicensed()
	license := utils.License()
	restrictPublicChannel := *th.App.Config().TeamSettings.RestrictPublicChannelCreation
	restrictPrivateChannel := *th.App.Config().TeamSettings.RestrictPrivateChannelCreation
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictPublicChannelCreation = restrictPublicChannel })
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictPrivateChannelCreation = restrictPrivateChannel })
		utils.SetIsLicensed(isLicensed)
		utils.SetLicense(license)
		th.App.SetDefaultRolesBasedOnConfig()
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictPublicChannelCreation = model.PERMISSIONS_ALL })
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictPrivateChannelCreation = model.PERMISSIONS_ALL })
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	channel.Name = GenerateTestChannelName()
	_, resp = Client.CreateChannel(channel)
	CheckNoError(t, resp)

	private.Name = GenerateTestChannelName()
	_, resp = Client.CreateChannel(private)
	CheckNoError(t, resp)

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPublicChannelCreation = model.PERMISSIONS_TEAM_ADMIN
	})
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelCreation = model.PERMISSIONS_TEAM_ADMIN
	})
	th.App.SetDefaultRolesBasedOnConfig()

	_, resp = Client.CreateChannel(channel)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.CreateChannel(private)
	CheckForbiddenStatus(t, resp)

	th.LoginTeamAdmin()

	channel.Name = GenerateTestChannelName()
	_, resp = Client.CreateChannel(channel)
	CheckNoError(t, resp)

	private.Name = GenerateTestChannelName()
	_, resp = Client.CreateChannel(private)
	CheckNoError(t, resp)

	channel.Name = GenerateTestChannelName()
	_, resp = th.SystemAdminClient.CreateChannel(channel)
	CheckNoError(t, resp)

	private.Name = GenerateTestChannelName()
	_, resp = th.SystemAdminClient.CreateChannel(private)
	CheckNoError(t, resp)

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPublicChannelCreation = model.PERMISSIONS_SYSTEM_ADMIN
	})
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelCreation = model.PERMISSIONS_SYSTEM_ADMIN
	})
	th.App.SetDefaultRolesBasedOnConfig()

	th.LoginBasic()

	_, resp = Client.CreateChannel(channel)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.CreateChannel(private)
	CheckForbiddenStatus(t, resp)

	th.LoginTeamAdmin()

	_, resp = Client.CreateChannel(channel)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.CreateChannel(private)
	CheckForbiddenStatus(t, resp)

	channel.Name = GenerateTestChannelName()
	_, resp = th.SystemAdminClient.CreateChannel(channel)
	CheckNoError(t, resp)

	private.Name = GenerateTestChannelName()
	_, resp = th.SystemAdminClient.CreateChannel(private)
	CheckNoError(t, resp)

	// Check that if unlicensed the policy restriction is not enforced.
	utils.SetIsLicensed(false)
	utils.SetLicense(nil)
	th.App.SetDefaultRolesBasedOnConfig()

	channel.Name = GenerateTestChannelName()
	_, resp = Client.CreateChannel(channel)
	CheckNoError(t, resp)

	private.Name = GenerateTestChannelName()
	_, resp = Client.CreateChannel(private)
	CheckNoError(t, resp)

	if r, err := Client.DoApiPost("/channels", "garbage"); err == nil {
		t.Fatal("should have errored")
	} else {
		if r.StatusCode != http.StatusBadRequest {
			t.Log("actual: " + strconv.Itoa(r.StatusCode))
			t.Log("expected: " + strconv.Itoa(http.StatusBadRequest))
			t.Fatal("wrong status code")
		}
	}
}

func TestUpdateChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam

	channel := &model.Channel{DisplayName: "Test API Name", Name: GenerateTestChannelName(), Type: model.CHANNEL_OPEN, TeamId: team.Id}
	private := &model.Channel{DisplayName: "Test API Name", Name: GenerateTestChannelName(), Type: model.CHANNEL_PRIVATE, TeamId: team.Id}

	channel, resp := Client.CreateChannel(channel)
	private, resp = Client.CreateChannel(private)

	//Update a open channel
	channel.DisplayName = "My new display name"
	channel.Header = "My fancy header"
	channel.Purpose = "Mattermost ftw!"

	newChannel, resp := Client.UpdateChannel(channel)
	CheckNoError(t, resp)

	if newChannel.DisplayName != channel.DisplayName {
		t.Fatal("Update failed for DisplayName")
	}

	if newChannel.Header != channel.Header {
		t.Fatal("Update failed for Header")
	}

	if newChannel.Purpose != channel.Purpose {
		t.Fatal("Update failed for Purpose")
	}

	//Update a private channel
	private.DisplayName = "My new display name for private channel"
	private.Header = "My fancy private header"
	private.Purpose = "Mattermost ftw! in private mode"

	newPrivateChannel, resp := Client.UpdateChannel(private)
	CheckNoError(t, resp)

	if newPrivateChannel.DisplayName != private.DisplayName {
		t.Fatal("Update failed for DisplayName in private channel")
	}

	if newPrivateChannel.Header != private.Header {
		t.Fatal("Update failed for Header in private channel")
	}

	if newPrivateChannel.Purpose != private.Purpose {
		t.Fatal("Update failed for Purpose in private channel")
	}

	//Non existing channel
	channel1 := &model.Channel{DisplayName: "Test API Name for apiv4", Name: GenerateTestChannelName(), Type: model.CHANNEL_OPEN, TeamId: team.Id}
	_, resp = Client.UpdateChannel(channel1)
	CheckNotFoundStatus(t, resp)

	//Try to update with not logged user
	Client.Logout()
	_, resp = Client.UpdateChannel(channel)
	CheckUnauthorizedStatus(t, resp)

	//Try to update using another user
	user := th.CreateUser()
	Client.Login(user.Email, user.Password)

	channel.DisplayName = "Should not update"
	_, resp = Client.UpdateChannel(channel)
	CheckNotFoundStatus(t, resp)

}

func TestPatchChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	patch := &model.ChannelPatch{
		Name:        new(string),
		DisplayName: new(string),
		Header:      new(string),
		Purpose:     new(string),
	}
	*patch.Name = model.NewId()
	*patch.DisplayName = model.NewId()
	*patch.Header = model.NewId()
	*patch.Purpose = model.NewId()

	channel, resp := Client.PatchChannel(th.BasicChannel.Id, patch)
	CheckNoError(t, resp)

	if *patch.Name != channel.Name {
		t.Fatal("do not match")
	} else if *patch.DisplayName != channel.DisplayName {
		t.Fatal("do not match")
	} else if *patch.Header != channel.Header {
		t.Fatal("do not match")
	} else if *patch.Purpose != channel.Purpose {
		t.Fatal("do not match")
	}

	patch.Name = nil
	oldName := channel.Name
	channel, resp = Client.PatchChannel(th.BasicChannel.Id, patch)
	CheckNoError(t, resp)

	if channel.Name != oldName {
		t.Fatal("should not have updated")
	}

	_, resp = Client.PatchChannel("junk", patch)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.PatchChannel(model.NewId(), patch)
	CheckNotFoundStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.PatchChannel(th.BasicChannel.Id, patch)
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.PatchChannel(th.BasicChannel.Id, patch)
	CheckNoError(t, resp)

	_, resp = th.SystemAdminClient.PatchChannel(th.BasicPrivateChannel.Id, patch)
	CheckNoError(t, resp)
}

func TestCreateDirectChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	user1 := th.BasicUser
	user2 := th.BasicUser2
	user3 := th.CreateUser()

	dm, resp := Client.CreateDirectChannel(user1.Id, user2.Id)
	CheckNoError(t, resp)

	channelName := ""
	if user2.Id > user1.Id {
		channelName = user1.Id + "__" + user2.Id
	} else {
		channelName = user2.Id + "__" + user1.Id
	}

	if dm.Name != channelName {
		t.Fatal("dm name didn't match")
	}

	_, resp = Client.CreateDirectChannel("junk", user2.Id)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.CreateDirectChannel(user1.Id, model.NewId())
	CheckBadRequestStatus(t, resp)

	_, resp = Client.CreateDirectChannel(model.NewId(), user1.Id)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.CreateDirectChannel(model.NewId(), user2.Id)
	CheckForbiddenStatus(t, resp)

	if r, err := Client.DoApiPost("/channels/direct", "garbage"); err == nil {
		t.Fatal("should have errored")
	} else {
		if r.StatusCode != http.StatusBadRequest {
			t.Log("actual: " + strconv.Itoa(r.StatusCode))
			t.Log("expected: " + strconv.Itoa(http.StatusBadRequest))
			t.Fatal("wrong status code")
		}
	}

	Client.Logout()
	_, resp = Client.CreateDirectChannel(model.NewId(), user2.Id)
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.CreateDirectChannel(user3.Id, user2.Id)
	CheckNoError(t, resp)
}

func TestCreateGroupChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	user := th.BasicUser
	user2 := th.BasicUser2
	user3 := th.CreateUser()

	userIds := []string{user.Id, user2.Id, user3.Id}

	rgc, resp := Client.CreateGroupChannel(userIds)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	if rgc == nil {
		t.Fatal("should have created a group channel")
	}

	if rgc.Type != model.CHANNEL_GROUP {
		t.Fatal("should have created a channel of group type")
	}

	m, _ := th.App.GetChannelMembersPage(rgc.Id, 0, 10)
	if len(*m) != 3 {
		t.Fatal("should have 3 channel members")
	}

	// saving duplicate group channel
	rgc2, resp := Client.CreateGroupChannel([]string{user3.Id, user2.Id})
	CheckNoError(t, resp)

	if rgc.Id != rgc2.Id {
		t.Fatal("should have returned existing channel")
	}

	m2, _ := th.App.GetChannelMembersPage(rgc2.Id, 0, 10)
	if !reflect.DeepEqual(*m, *m2) {
		t.Fatal("should be equal")
	}

	rgc, resp = Client.CreateGroupChannel([]string{user2.Id})
	CheckBadRequestStatus(t, resp)

	user4 := th.CreateUser()
	user5 := th.CreateUser()
	user6 := th.CreateUser()
	user7 := th.CreateUser()
	user8 := th.CreateUser()
	user9 := th.CreateUser()

	rgc, resp = Client.CreateGroupChannel([]string{user.Id, user2.Id, user3.Id, user4.Id, user5.Id, user6.Id, user7.Id, user8.Id, user9.Id})
	CheckBadRequestStatus(t, resp)

	if rgc != nil {
		t.Fatal("should return nil")
	}

	_, resp = Client.CreateGroupChannel([]string{user.Id, user2.Id, user3.Id, GenerateTestId()})
	CheckBadRequestStatus(t, resp)

	_, resp = Client.CreateGroupChannel([]string{user.Id, user2.Id, user3.Id, "junk"})
	CheckBadRequestStatus(t, resp)

	Client.Logout()

	_, resp = Client.CreateGroupChannel(userIds)
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.CreateGroupChannel(userIds)
	CheckNoError(t, resp)
}

func TestGetChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	channel, resp := Client.GetChannel(th.BasicChannel.Id, "")
	CheckNoError(t, resp)

	if channel.Id != th.BasicChannel.Id {
		t.Fatal("ids did not match")
	}

	Client.RemoveUserFromChannel(th.BasicChannel.Id, th.BasicUser.Id)
	_, resp = Client.GetChannel(th.BasicChannel.Id, "")
	CheckNoError(t, resp)

	channel, resp = Client.GetChannel(th.BasicPrivateChannel.Id, "")
	CheckNoError(t, resp)

	if channel.Id != th.BasicPrivateChannel.Id {
		t.Fatal("ids did not match")
	}

	Client.RemoveUserFromChannel(th.BasicPrivateChannel.Id, th.BasicUser.Id)
	_, resp = Client.GetChannel(th.BasicPrivateChannel.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetChannel(model.NewId(), "")
	CheckNotFoundStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetChannel(th.BasicChannel.Id, "")
	CheckUnauthorizedStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetChannel(th.BasicChannel.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetChannel(th.BasicChannel.Id, "")
	CheckNoError(t, resp)

	_, resp = th.SystemAdminClient.GetChannel(th.BasicPrivateChannel.Id, "")
	CheckNoError(t, resp)

	_, resp = th.SystemAdminClient.GetChannel(th.BasicUser.Id, "")
	CheckNotFoundStatus(t, resp)
}

func TestGetDeletedChannelsForTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam

	channels, resp := Client.GetDeletedChannelsForTeam(team.Id, 0, 100, "")
	CheckForbiddenStatus(t, resp)

	th.LoginTeamAdmin()

	channels, resp = Client.GetDeletedChannelsForTeam(team.Id, 0, 100, "")
	CheckNoError(t, resp)
	numInitialChannelsForTeam := len(channels)

	// create and delete public channel
	publicChannel1 := th.CreatePublicChannel()
	Client.DeleteChannel(publicChannel1.Id)

	channels, resp = Client.GetDeletedChannelsForTeam(team.Id, 0, 100, "")
	CheckNoError(t, resp)
	if len(channels) != numInitialChannelsForTeam+1 {
		t.Fatal("should be 1 deleted channel")
	}

	publicChannel2 := th.CreatePublicChannel()
	Client.DeleteChannel(publicChannel2.Id)

	channels, resp = Client.GetDeletedChannelsForTeam(team.Id, 0, 100, "")
	CheckNoError(t, resp)
	if len(channels) != numInitialChannelsForTeam+2 {
		t.Fatal("should be 2 deleted channels")
	}

	channels, resp = Client.GetDeletedChannelsForTeam(team.Id, 0, 1, "")
	CheckNoError(t, resp)
	if len(channels) != 1 {
		t.Fatal("should be one channel per page")
	}

	channels, resp = Client.GetDeletedChannelsForTeam(team.Id, 1, 1, "")
	CheckNoError(t, resp)
	if len(channels) != 1 {
		t.Fatal("should be one channel per page")
	}
}

func TestGetPublicChannelsForTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam
	publicChannel1 := th.BasicChannel
	publicChannel2 := th.BasicChannel2

	channels, resp := Client.GetPublicChannelsForTeam(team.Id, 0, 100, "")
	CheckNoError(t, resp)
	if len(channels) != 4 {
		t.Fatal("wrong length")
	}

	for i, c := range channels {
		if c.Type != model.CHANNEL_OPEN {
			t.Fatal("should include open channel only")
		}

		// only check the created 2 public channels
		if i < 2 && !(c.DisplayName == publicChannel1.DisplayName || c.DisplayName == publicChannel2.DisplayName) {
			t.Logf("channel %v: %v", i, c.DisplayName)
			t.Fatal("should match public channel display name only")
		}
	}

	privateChannel := th.CreatePrivateChannel()
	channels, resp = Client.GetPublicChannelsForTeam(team.Id, 0, 100, "")
	CheckNoError(t, resp)
	if len(channels) != 4 {
		t.Fatal("wrong length")
	}

	for _, c := range channels {
		if c.Type != model.CHANNEL_OPEN {
			t.Fatal("should not include private channel")
		}

		if c.DisplayName == privateChannel.DisplayName {
			t.Fatal("should not match private channel display name")
		}
	}

	channels, resp = Client.GetPublicChannelsForTeam(team.Id, 0, 1, "")
	CheckNoError(t, resp)
	if len(channels) != 1 {
		t.Fatal("should be one channel per page")
	}

	channels, resp = Client.GetPublicChannelsForTeam(team.Id, 1, 1, "")
	CheckNoError(t, resp)
	if len(channels) != 1 {
		t.Fatal("should be one channel per page")
	}

	channels, resp = Client.GetPublicChannelsForTeam(team.Id, 10000, 100, "")
	CheckNoError(t, resp)
	if len(channels) != 0 {
		t.Fatal("should be no channel")
	}

	_, resp = Client.GetPublicChannelsForTeam("junk", 0, 100, "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetPublicChannelsForTeam(model.NewId(), 0, 100, "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetPublicChannelsForTeam(team.Id, 0, 100, "")
	CheckUnauthorizedStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetPublicChannelsForTeam(team.Id, 0, 100, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetPublicChannelsForTeam(team.Id, 0, 100, "")
	CheckNoError(t, resp)
}

func TestGetPublicChannelsByIdsForTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	teamId := th.BasicTeam.Id
	input := []string{th.BasicChannel.Id}
	output := []string{th.BasicChannel.DisplayName}

	channels, resp := Client.GetPublicChannelsByIdsForTeam(teamId, input)
	CheckNoError(t, resp)

	if len(channels) != 1 {
		t.Fatal("should return 1 channel")
	}

	if (channels)[0].DisplayName != output[0] {
		t.Fatal("missing channel")
	}

	input = append(input, GenerateTestId())
	input = append(input, th.BasicChannel2.Id)
	input = append(input, th.BasicPrivateChannel.Id)
	output = append(output, th.BasicChannel2.DisplayName)
	sort.Strings(output)

	channels, resp = Client.GetPublicChannelsByIdsForTeam(teamId, input)
	CheckNoError(t, resp)

	if len(channels) != 2 {
		t.Fatal("should return 2 channels")
	}

	for i, c := range channels {
		if c.DisplayName != output[i] {
			t.Fatal("missing channel")
		}
	}

	_, resp = Client.GetPublicChannelsByIdsForTeam(GenerateTestId(), input)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetPublicChannelsByIdsForTeam(teamId, []string{})
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetPublicChannelsByIdsForTeam(teamId, []string{"junk"})
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetPublicChannelsByIdsForTeam(teamId, []string{GenerateTestId()})
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetPublicChannelsByIdsForTeam(teamId, []string{th.BasicPrivateChannel.Id})
	CheckNotFoundStatus(t, resp)

	Client.Logout()

	_, resp = Client.GetPublicChannelsByIdsForTeam(teamId, input)
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.GetPublicChannelsByIdsForTeam(teamId, input)
	CheckNoError(t, resp)
}

func TestGetChannelsForTeamForUser(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	channels, resp := Client.GetChannelsForTeamForUser(th.BasicTeam.Id, th.BasicUser.Id, "")
	CheckNoError(t, resp)

	found := make([]bool, 3)
	for _, c := range channels {
		if c.Id == th.BasicChannel.Id {
			found[0] = true
		} else if c.Id == th.BasicChannel2.Id {
			found[1] = true
		} else if c.Id == th.BasicPrivateChannel.Id {
			found[2] = true
		}

		if c.TeamId != th.BasicTeam.Id && c.TeamId != "" {
			t.Fatal("wrong team")
		}
	}

	for _, f := range found {
		if !f {
			t.Fatal("missing a channel")
		}
	}

	channels, resp = Client.GetChannelsForTeamForUser(th.BasicTeam.Id, th.BasicUser.Id, resp.Etag)
	CheckEtag(t, channels, resp)

	_, resp = Client.GetChannelsForTeamForUser(th.BasicTeam.Id, "junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelsForTeamForUser("junk", th.BasicUser.Id, "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelsForTeamForUser(th.BasicTeam.Id, th.BasicUser2.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetChannelsForTeamForUser(model.NewId(), th.BasicUser.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetChannelsForTeamForUser(th.BasicTeam.Id, th.BasicUser.Id, "")
	CheckNoError(t, resp)
}

func TestSearchChannels(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	search := &model.ChannelSearch{Term: th.BasicChannel.Name}

	channels, resp := Client.SearchChannels(th.BasicTeam.Id, search)
	CheckNoError(t, resp)

	found := false
	for _, c := range channels {
		if c.Type != model.CHANNEL_OPEN {
			t.Fatal("should only return public channels")
		}

		if c.Id == th.BasicChannel.Id {
			found = true
		}
	}

	if !found {
		t.Fatal("didn't find channel")
	}

	search.Term = th.BasicPrivateChannel.Name
	channels, resp = Client.SearchChannels(th.BasicTeam.Id, search)
	CheckNoError(t, resp)

	found = false
	for _, c := range channels {
		if c.Id == th.BasicPrivateChannel.Id {
			found = true
		}
	}

	if found {
		t.Fatal("shouldn't find private channel")
	}

	search.Term = ""
	_, resp = Client.SearchChannels(th.BasicTeam.Id, search)
	CheckNoError(t, resp)

	search.Term = th.BasicChannel.Name
	_, resp = Client.SearchChannels(model.NewId(), search)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.SearchChannels("junk", search)
	CheckBadRequestStatus(t, resp)

	_, resp = th.SystemAdminClient.SearchChannels(th.BasicTeam.Id, search)
	CheckNoError(t, resp)
}

func TestDeleteChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam
	user := th.BasicUser
	user2 := th.BasicUser2

	// successful delete of public channel
	publicChannel1 := th.CreatePublicChannel()
	pass, resp := Client.DeleteChannel(publicChannel1.Id)
	CheckNoError(t, resp)

	if !pass {
		t.Fatal("should have passed")
	}

	if ch, err := th.App.GetChannel(publicChannel1.Id); err == nil && ch.DeleteAt == 0 {
		t.Fatal("should have failed to get deleted channel")
	} else if err := th.App.JoinChannel(ch, user2.Id); err == nil {
		t.Fatal("should have failed to join deleted channel")
	}

	post1 := &model.Post{ChannelId: publicChannel1.Id, Message: "a" + GenerateTestId() + "a"}
	if _, err := Client.CreatePost(post1); err == nil {
		t.Fatal("should have failed to post to deleted channel")
	}

	// successful delete of private channel
	privateChannel2 := th.CreatePrivateChannel()
	_, resp = Client.DeleteChannel(privateChannel2.Id)
	CheckNoError(t, resp)

	// successful delete of channel with multiple members
	publicChannel3 := th.CreatePublicChannel()
	th.App.AddUserToChannel(user2, publicChannel3)
	_, resp = Client.DeleteChannel(publicChannel3.Id)
	CheckNoError(t, resp)

	// successful delete by TeamAdmin of channel created by user
	publicChannel4 := th.CreatePublicChannel()
	th.LoginTeamAdmin()
	_, resp = Client.DeleteChannel(publicChannel4.Id)
	CheckNoError(t, resp)

	// default channel cannot be deleted.
	defaultChannel, _ := th.App.GetChannelByName(model.DEFAULT_CHANNEL, team.Id)
	pass, resp = Client.DeleteChannel(defaultChannel.Id)
	CheckBadRequestStatus(t, resp)

	if pass {
		t.Fatal("should have failed")
	}

	// check system admin can delete a channel without any appropriate team or channel membership.
	sdTeam := th.CreateTeamWithClient(Client)
	sdPublicChannel := &model.Channel{
		DisplayName: "dn_" + model.NewId(),
		Name:        GenerateTestChannelName(),
		Type:        model.CHANNEL_OPEN,
		TeamId:      sdTeam.Id,
	}
	sdPublicChannel, resp = Client.CreateChannel(sdPublicChannel)
	CheckNoError(t, resp)
	_, resp = th.SystemAdminClient.DeleteChannel(sdPublicChannel.Id)
	CheckNoError(t, resp)

	sdPrivateChannel := &model.Channel{
		DisplayName: "dn_" + model.NewId(),
		Name:        GenerateTestChannelName(),
		Type:        model.CHANNEL_PRIVATE,
		TeamId:      sdTeam.Id,
	}
	sdPrivateChannel, resp = Client.CreateChannel(sdPrivateChannel)
	CheckNoError(t, resp)
	_, resp = th.SystemAdminClient.DeleteChannel(sdPrivateChannel.Id)
	CheckNoError(t, resp)

	th.LoginBasic()
	publicChannel5 := th.CreatePublicChannel()
	Client.Logout()

	Client.Login(user2.Id, user2.Password)
	_, resp = Client.DeleteChannel(publicChannel5.Id)
	CheckUnauthorizedStatus(t, resp)

	_, resp = Client.DeleteChannel("junk")
	CheckUnauthorizedStatus(t, resp)

	Client.Logout()
	_, resp = Client.DeleteChannel(GenerateTestId())
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.DeleteChannel(publicChannel5.Id)
	CheckNoError(t, resp)

	th.InitBasic().InitSystemAdmin()

	isLicensed := utils.IsLicensed()
	license := utils.License()
	restrictPublicChannel := *th.App.Config().TeamSettings.RestrictPublicChannelManagement
	restrictPrivateChannel := *th.App.Config().TeamSettings.RestrictPrivateChannelManagement
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictPublicChannelManagement = restrictPublicChannel })
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictPrivateChannelManagement = restrictPrivateChannel })
		utils.SetIsLicensed(isLicensed)
		utils.SetLicense(license)
		th.App.SetDefaultRolesBasedOnConfig()
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictPublicChannelManagement = model.PERMISSIONS_ALL })
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictPrivateChannelManagement = model.PERMISSIONS_ALL })
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	Client = th.Client
	team = th.BasicTeam
	user = th.BasicUser
	user2 = th.BasicUser2

	// channels created by SystemAdmin
	publicChannel6 := th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_OPEN)
	privateChannel7 := th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_PRIVATE)
	th.App.AddUserToChannel(user, publicChannel6)
	th.App.AddUserToChannel(user, privateChannel7)
	th.App.AddUserToChannel(user2, privateChannel7)

	// successful delete by user
	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckNoError(t, resp)

	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckNoError(t, resp)

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPublicChannelDeletion = model.PERMISSIONS_CHANNEL_ADMIN
	})
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelDeletion = model.PERMISSIONS_CHANNEL_ADMIN
	})
	th.App.SetDefaultRolesBasedOnConfig()

	// channels created by SystemAdmin
	publicChannel6 = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_OPEN)
	privateChannel7 = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_PRIVATE)
	th.App.AddUserToChannel(user, publicChannel6)
	th.App.AddUserToChannel(user, privateChannel7)
	th.App.AddUserToChannel(user2, privateChannel7)

	// cannot delete by user
	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckForbiddenStatus(t, resp)

	// successful delete by channel admin
	th.MakeUserChannelAdmin(user, publicChannel6)
	th.MakeUserChannelAdmin(user, privateChannel7)
	sqlstore.ClearChannelCaches()

	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckNoError(t, resp)

	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckNoError(t, resp)

	// // channels created by SystemAdmin
	publicChannel6 = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_OPEN)
	privateChannel7 = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_PRIVATE)
	th.App.AddUserToChannel(user, publicChannel6)
	th.App.AddUserToChannel(user, privateChannel7)
	th.App.AddUserToChannel(user2, privateChannel7)

	// successful delete by team admin
	th.UpdateUserToTeamAdmin(user, team)
	th.App.InvalidateAllCaches()
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()

	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckNoError(t, resp)

	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckNoError(t, resp)

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPublicChannelDeletion = model.PERMISSIONS_TEAM_ADMIN
	})
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelDeletion = model.PERMISSIONS_TEAM_ADMIN
	})
	th.App.SetDefaultRolesBasedOnConfig()
	th.UpdateUserToNonTeamAdmin(user, team)
	th.App.InvalidateAllCaches()
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()

	// channels created by SystemAdmin
	publicChannel6 = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_OPEN)
	privateChannel7 = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_PRIVATE)
	th.App.AddUserToChannel(user, publicChannel6)
	th.App.AddUserToChannel(user, privateChannel7)
	th.App.AddUserToChannel(user2, privateChannel7)

	// cannot delete by user
	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckForbiddenStatus(t, resp)

	// // cannot delete by channel admin
	th.MakeUserChannelAdmin(user, publicChannel6)
	th.MakeUserChannelAdmin(user, privateChannel7)
	sqlstore.ClearChannelCaches()

	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckForbiddenStatus(t, resp)

	// successful delete by team admin
	th.UpdateUserToTeamAdmin(th.BasicUser, team)
	th.App.InvalidateAllCaches()
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()

	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckNoError(t, resp)

	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckNoError(t, resp)

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPublicChannelDeletion = model.PERMISSIONS_SYSTEM_ADMIN
	})
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelDeletion = model.PERMISSIONS_SYSTEM_ADMIN
	})
	th.App.SetDefaultRolesBasedOnConfig()

	// channels created by SystemAdmin
	publicChannel6 = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_OPEN)
	privateChannel7 = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_PRIVATE)
	th.App.AddUserToChannel(user, publicChannel6)
	th.App.AddUserToChannel(user, privateChannel7)
	th.App.AddUserToChannel(user2, privateChannel7)

	// cannot delete by user
	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckForbiddenStatus(t, resp)

	// cannot delete by channel admin
	th.MakeUserChannelAdmin(user, publicChannel6)
	th.MakeUserChannelAdmin(user, privateChannel7)
	sqlstore.ClearChannelCaches()

	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckForbiddenStatus(t, resp)

	// cannot delete by team admin
	th.UpdateUserToTeamAdmin(th.BasicUser, team)
	th.App.InvalidateAllCaches()
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()

	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckForbiddenStatus(t, resp)

	// successful delete by SystemAdmin
	_, resp = th.SystemAdminClient.DeleteChannel(publicChannel6.Id)
	CheckNoError(t, resp)

	_, resp = th.SystemAdminClient.DeleteChannel(privateChannel7.Id)
	CheckNoError(t, resp)

	// last member of a public channel should have required permission to delete
	publicChannel6 = th.CreateChannelWithClient(th.Client, model.CHANNEL_OPEN)
	_, resp = Client.DeleteChannel(publicChannel6.Id)
	CheckForbiddenStatus(t, resp)

	// last member of a private channel should not be able to delete it if they don't have required permissions
	privateChannel7 = th.CreateChannelWithClient(th.Client, model.CHANNEL_PRIVATE)
	_, resp = Client.DeleteChannel(privateChannel7.Id)
	CheckForbiddenStatus(t, resp)
}

func TestRestoreChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	publicChannel1 := th.CreatePublicChannel()
	Client.DeleteChannel(publicChannel1.Id)

	privateChannel1 := th.CreatePrivateChannel()
	Client.DeleteChannel(privateChannel1.Id)

	_, resp := Client.RestoreChannel(publicChannel1.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.RestoreChannel(privateChannel1.Id)
	CheckForbiddenStatus(t, resp)

	th.LoginTeamAdmin()

	_, resp = Client.RestoreChannel(publicChannel1.Id)
	CheckOKStatus(t, resp)

	_, resp = Client.RestoreChannel(privateChannel1.Id)
	CheckOKStatus(t, resp)
}

func TestGetChannelByName(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	channel, resp := Client.GetChannelByName(th.BasicChannel.Name, th.BasicTeam.Id, "")
	CheckNoError(t, resp)

	if channel.Name != th.BasicChannel.Name {
		t.Fatal("names did not match")
	}

	channel, resp = Client.GetChannelByName(th.BasicPrivateChannel.Name, th.BasicTeam.Id, "")
	CheckNoError(t, resp)

	if channel.Name != th.BasicPrivateChannel.Name {
		t.Fatal("names did not match")
	}

	Client.RemoveUserFromChannel(th.BasicChannel.Id, th.BasicUser.Id)
	_, resp = Client.GetChannelByName(th.BasicChannel.Name, th.BasicTeam.Id, "")
	CheckNoError(t, resp)

	Client.RemoveUserFromChannel(th.BasicPrivateChannel.Id, th.BasicUser.Id)
	_, resp = Client.GetChannelByName(th.BasicPrivateChannel.Name, th.BasicTeam.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetChannelByName(GenerateTestChannelName(), th.BasicTeam.Id, "")
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetChannelByName(GenerateTestChannelName(), "junk", "")
	CheckBadRequestStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetChannelByName(th.BasicChannel.Name, th.BasicTeam.Id, "")
	CheckUnauthorizedStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetChannelByName(th.BasicChannel.Name, th.BasicTeam.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetChannelByName(th.BasicChannel.Name, th.BasicTeam.Id, "")
	CheckNoError(t, resp)
}

func TestGetChannelByNameForTeamName(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	channel, resp := th.SystemAdminClient.GetChannelByNameForTeamName(th.BasicChannel.Name, th.BasicTeam.Name, "")
	CheckNoError(t, resp)

	if channel.Name != th.BasicChannel.Name {
		t.Fatal("names did not match")
	}

	_, resp = Client.GetChannelByNameForTeamName(th.BasicChannel.Name, th.BasicTeam.Name, "")
	CheckNoError(t, resp)

	_, resp = Client.GetChannelByNameForTeamName(th.BasicChannel.Name, model.NewRandomString(15), "")
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetChannelByNameForTeamName(GenerateTestChannelName(), th.BasicTeam.Name, "")
	CheckNotFoundStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetChannelByNameForTeamName(th.BasicChannel.Name, th.BasicTeam.Name, "")
	CheckUnauthorizedStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetChannelByNameForTeamName(th.BasicChannel.Name, th.BasicTeam.Name, "")
	CheckForbiddenStatus(t, resp)
}

func TestGetChannelMembers(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	members, resp := Client.GetChannelMembers(th.BasicChannel.Id, 0, 60, "")
	CheckNoError(t, resp)

	if len(*members) != 3 {
		t.Fatal("should only be 3 users in channel")
	}

	members, resp = Client.GetChannelMembers(th.BasicChannel.Id, 0, 2, "")
	CheckNoError(t, resp)

	if len(*members) != 2 {
		t.Fatal("should only be 2 users")
	}

	members, resp = Client.GetChannelMembers(th.BasicChannel.Id, 1, 1, "")
	CheckNoError(t, resp)

	if len(*members) != 1 {
		t.Fatal("should only be 1 user")
	}

	members, resp = Client.GetChannelMembers(th.BasicChannel.Id, 1000, 100000, "")
	CheckNoError(t, resp)

	if len(*members) != 0 {
		t.Fatal("should be 0 users")
	}

	_, resp = Client.GetChannelMembers("", 0, 60, "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelMembers("junk", 0, 60, "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelMembers(model.NewId(), 0, 60, "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetChannelMembers(th.BasicChannel.Id, 0, 60, "")
	CheckUnauthorizedStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetChannelMembers(th.BasicChannel.Id, 0, 60, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetChannelMembers(th.BasicChannel.Id, 0, 60, "")
	CheckNoError(t, resp)
}

func TestGetChannelMembersByIds(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	cm, resp := Client.GetChannelMembersByIds(th.BasicChannel.Id, []string{th.BasicUser.Id})
	CheckNoError(t, resp)

	if (*cm)[0].UserId != th.BasicUser.Id {
		t.Fatal("returned wrong user")
	}

	_, resp = Client.GetChannelMembersByIds(th.BasicChannel.Id, []string{})
	CheckBadRequestStatus(t, resp)

	cm1, resp := Client.GetChannelMembersByIds(th.BasicChannel.Id, []string{"junk"})
	CheckNoError(t, resp)
	if len(*cm1) > 0 {
		t.Fatal("no users should be returned")
	}

	cm1, resp = Client.GetChannelMembersByIds(th.BasicChannel.Id, []string{"junk", th.BasicUser.Id})
	CheckNoError(t, resp)
	if len(*cm1) != 1 {
		t.Fatal("1 member should be returned")
	}

	cm1, resp = Client.GetChannelMembersByIds(th.BasicChannel.Id, []string{th.BasicUser2.Id, th.BasicUser.Id})
	CheckNoError(t, resp)
	if len(*cm1) != 2 {
		t.Fatal("2 members should be returned")
	}

	_, resp = Client.GetChannelMembersByIds("junk", []string{th.BasicUser.Id})
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelMembersByIds(model.NewId(), []string{th.BasicUser.Id})
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetChannelMembersByIds(th.BasicChannel.Id, []string{th.BasicUser.Id})
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.GetChannelMembersByIds(th.BasicChannel.Id, []string{th.BasicUser2.Id, th.BasicUser.Id})
	CheckNoError(t, resp)
}

func TestGetChannelMember(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	member, resp := Client.GetChannelMember(th.BasicChannel.Id, th.BasicUser.Id, "")
	CheckNoError(t, resp)

	if member.ChannelId != th.BasicChannel.Id {
		t.Fatal("wrong channel id")
	}

	if member.UserId != th.BasicUser.Id {
		t.Fatal("wrong user id")
	}

	_, resp = Client.GetChannelMember("", th.BasicUser.Id, "")
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetChannelMember("junk", th.BasicUser.Id, "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelMember(model.NewId(), th.BasicUser.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetChannelMember(th.BasicChannel.Id, "", "")
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetChannelMember(th.BasicChannel.Id, "junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelMember(th.BasicChannel.Id, model.NewId(), "")
	CheckNotFoundStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetChannelMember(th.BasicChannel.Id, th.BasicUser.Id, "")
	CheckUnauthorizedStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetChannelMember(th.BasicChannel.Id, th.BasicUser.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetChannelMember(th.BasicChannel.Id, th.BasicUser.Id, "")
	CheckNoError(t, resp)
}

func TestGetChannelMembersForUser(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	members, resp := Client.GetChannelMembersForUser(th.BasicUser.Id, th.BasicTeam.Id, "")
	CheckNoError(t, resp)

	if len(*members) != 5 {
		t.Fatal("should have 5 members on team")
	}

	_, resp = Client.GetChannelMembersForUser("", th.BasicTeam.Id, "")
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetChannelMembersForUser("junk", th.BasicTeam.Id, "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelMembersForUser(model.NewId(), th.BasicTeam.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetChannelMembersForUser(th.BasicUser.Id, "", "")
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetChannelMembersForUser(th.BasicUser.Id, "junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelMembersForUser(th.BasicUser.Id, model.NewId(), "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetChannelMembersForUser(th.BasicUser.Id, th.BasicTeam.Id, "")
	CheckUnauthorizedStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetChannelMembersForUser(th.BasicUser.Id, th.BasicTeam.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetChannelMembersForUser(th.BasicUser.Id, th.BasicTeam.Id, "")
	CheckNoError(t, resp)
}

func TestViewChannel(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	view := &model.ChannelView{
		ChannelId: th.BasicChannel.Id,
	}

	viewResp, resp := Client.ViewChannel(th.BasicUser.Id, view)
	CheckNoError(t, resp)

	if viewResp.Status != "OK" {
		t.Fatal("should have passed")
	}

	channel, _ := th.App.GetChannel(th.BasicChannel.Id)

	if lastViewedAt := viewResp.LastViewedAtTimes[channel.Id]; lastViewedAt != channel.LastPostAt {
		t.Fatal("LastPostAt does not match returned LastViewedAt time")
	}

	view.PrevChannelId = th.BasicChannel.Id
	_, resp = Client.ViewChannel(th.BasicUser.Id, view)
	CheckNoError(t, resp)

	view.PrevChannelId = ""
	_, resp = Client.ViewChannel(th.BasicUser.Id, view)
	CheckNoError(t, resp)

	view.PrevChannelId = "junk"
	_, resp = Client.ViewChannel(th.BasicUser.Id, view)
	CheckNoError(t, resp)

	member, resp := Client.GetChannelMember(th.BasicChannel.Id, th.BasicUser.Id, "")
	CheckNoError(t, resp)
	channel, resp = Client.GetChannel(th.BasicChannel.Id, "")
	CheckNoError(t, resp)

	if member.MsgCount != channel.TotalMsgCount {
		t.Fatal("should match message counts")
	}

	if member.MentionCount != 0 {
		t.Fatal("should have no mentions")
	}

	_, resp = Client.ViewChannel("junk", view)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.ViewChannel(th.BasicUser2.Id, view)
	CheckForbiddenStatus(t, resp)

	if r, err := Client.DoApiPost(fmt.Sprintf("/channels/members/%v/view", th.BasicUser.Id), "garbage"); err == nil {
		t.Fatal("should have errored")
	} else {
		if r.StatusCode != http.StatusBadRequest {
			t.Log("actual: " + strconv.Itoa(r.StatusCode))
			t.Log("expected: " + strconv.Itoa(http.StatusBadRequest))
			t.Fatal("wrong status code")
		}
	}

	Client.Logout()
	_, resp = Client.ViewChannel(th.BasicUser.Id, view)
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.ViewChannel(th.BasicUser.Id, view)
	CheckNoError(t, resp)
}

func TestGetChannelUnread(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	user := th.BasicUser
	channel := th.BasicChannel

	channelUnread, resp := Client.GetChannelUnread(channel.Id, user.Id)
	CheckNoError(t, resp)
	if channelUnread.TeamId != th.BasicTeam.Id {
		t.Fatal("wrong team id returned for a regular user call")
	} else if channelUnread.ChannelId != channel.Id {
		t.Fatal("wrong team id returned for a regular user call")
	}

	_, resp = Client.GetChannelUnread("junk", user.Id)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelUnread(channel.Id, "junk")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelUnread(channel.Id, model.NewId())
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetChannelUnread(model.NewId(), user.Id)
	CheckForbiddenStatus(t, resp)

	newUser := th.CreateUser()
	Client.Login(newUser.Email, newUser.Password)
	_, resp = Client.GetChannelUnread(th.BasicChannel.Id, user.Id)
	CheckForbiddenStatus(t, resp)

	Client.Logout()

	_, resp = th.SystemAdminClient.GetChannelUnread(channel.Id, user.Id)
	CheckNoError(t, resp)

	_, resp = th.SystemAdminClient.GetChannelUnread(model.NewId(), user.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetChannelUnread(channel.Id, model.NewId())
	CheckNotFoundStatus(t, resp)
}

func TestGetChannelStats(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	channel := th.CreatePrivateChannel()

	stats, resp := Client.GetChannelStats(channel.Id, "")
	CheckNoError(t, resp)

	if stats.ChannelId != channel.Id {
		t.Fatal("couldnt't get extra info")
	} else if stats.MemberCount != 1 {
		t.Fatal("got incorrect member count")
	}

	_, resp = Client.GetChannelStats("junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetChannelStats(model.NewId(), "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetChannelStats(channel.Id, "")
	CheckUnauthorizedStatus(t, resp)

	th.LoginBasic2()

	_, resp = Client.GetChannelStats(channel.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetChannelStats(channel.Id, "")
	CheckNoError(t, resp)
}

func TestGetPinnedPosts(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	channel := th.BasicChannel

	posts, resp := Client.GetPinnedPosts(channel.Id, "")
	CheckNoError(t, resp)
	if len(posts.Posts) != 0 {
		t.Fatal("should not have gotten a pinned post")
	}

	pinnedPost := th.CreatePinnedPost()
	posts, resp = Client.GetPinnedPosts(channel.Id, "")
	CheckNoError(t, resp)
	if len(posts.Posts) != 1 {
		t.Fatal("should have returned 1 pinned post")
	}
	if _, ok := posts.Posts[pinnedPost.Id]; !ok {
		t.Fatal("missing pinned post")
	}

	posts, resp = Client.GetPinnedPosts(channel.Id, resp.Etag)
	CheckEtag(t, posts, resp)

	_, resp = Client.GetPinnedPosts(GenerateTestId(), "")
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetPinnedPosts("junk", "")
	CheckBadRequestStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetPinnedPosts(channel.Id, "")
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.GetPinnedPosts(channel.Id, "")
	CheckNoError(t, resp)
}

func TestUpdateChannelRoles(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	const CHANNEL_ADMIN = "channel_admin channel_user"
	const CHANNEL_MEMBER = "channel_user"

	// User 1 creates a channel, making them channel admin by default.
	channel := th.CreatePublicChannel()

	// Adds User 2 to the channel, making them a channel member by default.
	th.App.AddUserToChannel(th.BasicUser2, channel)

	// User 1 promotes User 2
	pass, resp := Client.UpdateChannelRoles(channel.Id, th.BasicUser2.Id, CHANNEL_ADMIN)
	CheckNoError(t, resp)

	if !pass {
		t.Fatal("should have passed")
	}

	member, resp := Client.GetChannelMember(channel.Id, th.BasicUser2.Id, "")
	CheckNoError(t, resp)

	if member.Roles != CHANNEL_ADMIN {
		t.Fatal("roles don't match")
	}

	// User 1 demotes User 2
	_, resp = Client.UpdateChannelRoles(channel.Id, th.BasicUser2.Id, CHANNEL_MEMBER)
	CheckNoError(t, resp)

	th.LoginBasic2()

	// User 2 cannot demote User 1
	_, resp = Client.UpdateChannelRoles(channel.Id, th.BasicUser.Id, CHANNEL_MEMBER)
	CheckForbiddenStatus(t, resp)

	// User 2 cannot promote self
	_, resp = Client.UpdateChannelRoles(channel.Id, th.BasicUser2.Id, CHANNEL_ADMIN)
	CheckForbiddenStatus(t, resp)

	th.LoginBasic()

	// User 1 demotes self
	_, resp = Client.UpdateChannelRoles(channel.Id, th.BasicUser.Id, CHANNEL_MEMBER)
	CheckNoError(t, resp)

	// System Admin promotes User 1
	_, resp = th.SystemAdminClient.UpdateChannelRoles(channel.Id, th.BasicUser.Id, CHANNEL_ADMIN)
	CheckNoError(t, resp)

	// System Admin demotes User 1
	_, resp = th.SystemAdminClient.UpdateChannelRoles(channel.Id, th.BasicUser.Id, CHANNEL_MEMBER)
	CheckNoError(t, resp)

	// System Admin promotes User 1
	pass, resp = th.SystemAdminClient.UpdateChannelRoles(channel.Id, th.BasicUser.Id, CHANNEL_ADMIN)
	CheckNoError(t, resp)

	th.LoginBasic()

	_, resp = Client.UpdateChannelRoles(channel.Id, th.BasicUser.Id, "junk")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.UpdateChannelRoles(channel.Id, "junk", CHANNEL_MEMBER)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.UpdateChannelRoles("junk", th.BasicUser.Id, CHANNEL_MEMBER)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.UpdateChannelRoles(channel.Id, model.NewId(), CHANNEL_MEMBER)
	CheckNotFoundStatus(t, resp)

	_, resp = Client.UpdateChannelRoles(model.NewId(), th.BasicUser.Id, CHANNEL_MEMBER)
	CheckForbiddenStatus(t, resp)
}

func TestUpdateChannelNotifyProps(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	props := map[string]string{}
	props[model.DESKTOP_NOTIFY_PROP] = model.CHANNEL_NOTIFY_MENTION
	props[model.MARK_UNREAD_NOTIFY_PROP] = model.CHANNEL_MARK_UNREAD_MENTION

	pass, resp := Client.UpdateChannelNotifyProps(th.BasicChannel.Id, th.BasicUser.Id, props)
	CheckNoError(t, resp)

	if !pass {
		t.Fatal("should have passed")
	}

	member, err := th.App.GetChannelMember(th.BasicChannel.Id, th.BasicUser.Id)
	if err != nil {
		t.Fatal(err)
	}

	if member.NotifyProps[model.DESKTOP_NOTIFY_PROP] != model.CHANNEL_NOTIFY_MENTION {
		t.Fatal("bad update")
	} else if member.NotifyProps[model.MARK_UNREAD_NOTIFY_PROP] != model.CHANNEL_MARK_UNREAD_MENTION {
		t.Fatal("bad update")
	}

	_, resp = Client.UpdateChannelNotifyProps("junk", th.BasicUser.Id, props)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.UpdateChannelNotifyProps(th.BasicChannel.Id, "junk", props)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.UpdateChannelNotifyProps(model.NewId(), th.BasicUser.Id, props)
	CheckNotFoundStatus(t, resp)

	_, resp = Client.UpdateChannelNotifyProps(th.BasicChannel.Id, model.NewId(), props)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.UpdateChannelNotifyProps(th.BasicChannel.Id, th.BasicUser.Id, map[string]string{})
	CheckNoError(t, resp)

	Client.Logout()
	_, resp = Client.UpdateChannelNotifyProps(th.BasicChannel.Id, th.BasicUser.Id, props)
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.UpdateChannelNotifyProps(th.BasicChannel.Id, th.BasicUser.Id, props)
	CheckNoError(t, resp)
}

func TestAddChannelMember(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	user := th.BasicUser
	user2 := th.BasicUser2
	team := th.BasicTeam
	publicChannel := th.CreatePublicChannel()
	privateChannel := th.CreatePrivateChannel()

	user3 := th.CreateUserWithClient(th.SystemAdminClient)
	_, resp := th.SystemAdminClient.AddTeamMember(team.Id, user3.Id)
	CheckNoError(t, resp)

	cm, resp := Client.AddChannelMember(publicChannel.Id, user2.Id)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	if cm.ChannelId != publicChannel.Id {
		t.Fatal("should have returned exact channel")
	}

	if cm.UserId != user2.Id {
		t.Fatal("should have returned exact user added to public channel")
	}

	cm, resp = Client.AddChannelMember(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	if cm.ChannelId != privateChannel.Id {
		t.Fatal("should have returned exact channel")
	}

	if cm.UserId != user2.Id {
		t.Fatal("should have returned exact user added to private channel")
	}

	post := &model.Post{ChannelId: publicChannel.Id, Message: "a" + GenerateTestId() + "a"}
	rpost, err := Client.CreatePost(post)
	if err == nil {
		t.Fatal("should have created a post")
	}

	Client.RemoveUserFromChannel(publicChannel.Id, user.Id)
	_, resp = Client.AddChannelMemberWithRootId(publicChannel.Id, user.Id, rpost.Id)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	Client.RemoveUserFromChannel(publicChannel.Id, user.Id)
	_, resp = Client.AddChannelMemberWithRootId(publicChannel.Id, user.Id, "junk")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.AddChannelMemberWithRootId(publicChannel.Id, user.Id, GenerateTestId())
	CheckNotFoundStatus(t, resp)

	Client.RemoveUserFromChannel(publicChannel.Id, user.Id)
	_, resp = Client.AddChannelMember(publicChannel.Id, user.Id)
	CheckNoError(t, resp)

	cm, resp = Client.AddChannelMember(publicChannel.Id, "junk")
	CheckBadRequestStatus(t, resp)

	if cm != nil {
		t.Fatal("should return nothing")
	}

	_, resp = Client.AddChannelMember(publicChannel.Id, GenerateTestId())
	CheckNotFoundStatus(t, resp)

	_, resp = Client.AddChannelMember("junk", user2.Id)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.AddChannelMember(GenerateTestId(), user2.Id)
	CheckNotFoundStatus(t, resp)

	otherUser := th.CreateUser()
	otherChannel := th.CreatePublicChannel()
	Client.Logout()
	Client.Login(user2.Id, user2.Password)

	_, resp = Client.AddChannelMember(publicChannel.Id, otherUser.Id)
	CheckUnauthorizedStatus(t, resp)

	_, resp = Client.AddChannelMember(privateChannel.Id, otherUser.Id)
	CheckUnauthorizedStatus(t, resp)

	_, resp = Client.AddChannelMember(otherChannel.Id, otherUser.Id)
	CheckUnauthorizedStatus(t, resp)

	Client.Logout()
	Client.Login(user.Id, user.Password)

	// should fail adding user who is not a member of the team
	_, resp = Client.AddChannelMember(otherChannel.Id, otherUser.Id)
	CheckUnauthorizedStatus(t, resp)

	Client.DeleteChannel(otherChannel.Id)

	// should fail adding user to a deleted channel
	_, resp = Client.AddChannelMember(otherChannel.Id, user2.Id)
	CheckUnauthorizedStatus(t, resp)

	Client.Logout()
	_, resp = Client.AddChannelMember(publicChannel.Id, user2.Id)
	CheckUnauthorizedStatus(t, resp)

	_, resp = Client.AddChannelMember(privateChannel.Id, user2.Id)
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.AddChannelMember(publicChannel.Id, user2.Id)
	CheckNoError(t, resp)

	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	// Test policy does not apply to TE.
	restrictPrivateChannel := *th.App.Config().TeamSettings.RestrictPrivateChannelManageMembers
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.TeamSettings.RestrictPrivateChannelManageMembers = restrictPrivateChannel
		})
	}()
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelManageMembers = model.PERMISSIONS_CHANNEL_ADMIN
	})
	th.App.SetDefaultRolesBasedOnConfig()

	Client.Login(user2.Username, user2.Password)
	privateChannel = th.CreatePrivateChannel()
	_, resp = Client.AddChannelMember(privateChannel.Id, user.Id)
	CheckNoError(t, resp)
	Client.Logout()

	Client.Login(user.Username, user.Password)
	_, resp = Client.AddChannelMember(privateChannel.Id, user3.Id)
	CheckNoError(t, resp)
	Client.Logout()

	// Add a license
	isLicensed := utils.IsLicensed()
	license := utils.License()
	defer func() {
		utils.SetIsLicensed(isLicensed)
		utils.SetLicense(license)
		th.App.SetDefaultRolesBasedOnConfig()
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictPrivateChannelManageMembers = model.PERMISSIONS_ALL })
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	// Check that a regular channel user can add other users.
	Client.Login(user2.Username, user2.Password)
	privateChannel = th.CreatePrivateChannel()
	_, resp = Client.AddChannelMember(privateChannel.Id, user.Id)
	CheckNoError(t, resp)
	Client.Logout()

	Client.Login(user.Username, user.Password)
	_, resp = Client.AddChannelMember(privateChannel.Id, user3.Id)
	CheckNoError(t, resp)
	Client.Logout()

	// Test with CHANNEL_ADMIN level permission.
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelManageMembers = model.PERMISSIONS_CHANNEL_ADMIN
	})
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	Client.Login(user2.Username, user2.Password)
	privateChannel = th.CreatePrivateChannel()
	_, resp = Client.AddChannelMember(privateChannel.Id, user.Id)
	CheckNoError(t, resp)
	Client.Logout()

	Client.Login(user.Username, user.Password)
	_, resp = Client.AddChannelMember(privateChannel.Id, user3.Id)
	CheckForbiddenStatus(t, resp)
	Client.Logout()

	th.MakeUserChannelAdmin(user, privateChannel)
	th.App.InvalidateAllCaches()
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	Client.Login(user.Username, user.Password)
	_, resp = Client.AddChannelMember(privateChannel.Id, user3.Id)
	CheckNoError(t, resp)
	Client.Logout()

	// Test with TEAM_ADMIN level permission.
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelManageMembers = model.PERMISSIONS_TEAM_ADMIN
	})
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	Client.Login(user2.Username, user2.Password)
	privateChannel = th.CreatePrivateChannel()
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user.Id)
	CheckNoError(t, resp)
	Client.Logout()

	Client.Login(user.Username, user.Password)
	_, resp = Client.AddChannelMember(privateChannel.Id, user3.Id)
	CheckForbiddenStatus(t, resp)
	Client.Logout()

	th.UpdateUserToTeamAdmin(user, team)
	th.App.InvalidateAllCaches()
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	Client.Login(user.Username, user.Password)
	_, resp = Client.AddChannelMember(privateChannel.Id, user3.Id)
	CheckNoError(t, resp)
	Client.Logout()

	// Test with SYSTEM_ADMIN level permission.
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelManageMembers = model.PERMISSIONS_SYSTEM_ADMIN
	})
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	Client.Login(user2.Username, user2.Password)
	privateChannel = th.CreatePrivateChannel()
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user.Id)
	CheckNoError(t, resp)
	Client.Logout()

	Client.Login(user.Username, user.Password)
	_, resp = Client.AddChannelMember(privateChannel.Id, user3.Id)
	CheckForbiddenStatus(t, resp)
	Client.Logout()

	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user3.Id)
	CheckNoError(t, resp)
}

func TestRemoveChannelMember(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	user1 := th.BasicUser
	user2 := th.BasicUser2
	team := th.BasicTeam
	defer th.TearDown()
	Client := th.Client

	pass, resp := Client.RemoveUserFromChannel(th.BasicChannel.Id, th.BasicUser2.Id)
	CheckNoError(t, resp)

	if !pass {
		t.Fatal("should have passed")
	}

	_, resp = Client.RemoveUserFromChannel(th.BasicChannel.Id, th.BasicUser2.Id)
	CheckNoError(t, resp)

	_, resp = Client.RemoveUserFromChannel(th.BasicChannel.Id, "junk")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.RemoveUserFromChannel(th.BasicChannel.Id, model.NewId())
	CheckNotFoundStatus(t, resp)

	_, resp = Client.RemoveUserFromChannel(model.NewId(), th.BasicUser2.Id)
	CheckNotFoundStatus(t, resp)

	th.LoginBasic2()
	_, resp = Client.RemoveUserFromChannel(th.BasicChannel.Id, th.BasicUser.Id)
	CheckForbiddenStatus(t, resp)

	th.App.AddUserToChannel(th.BasicUser2, th.BasicChannel)
	_, resp = Client.RemoveUserFromChannel(th.BasicChannel.Id, th.BasicUser2.Id)
	CheckNoError(t, resp)

	_, resp = Client.RemoveUserFromChannel(th.BasicChannel2.Id, th.BasicUser.Id)
	CheckNoError(t, resp)

	_, resp = th.SystemAdminClient.RemoveUserFromChannel(th.BasicChannel.Id, th.BasicUser.Id)
	CheckNoError(t, resp)

	th.LoginBasic()
	private := th.CreatePrivateChannel()
	th.App.AddUserToChannel(th.BasicUser2, private)

	_, resp = Client.RemoveUserFromChannel(private.Id, th.BasicUser2.Id)
	CheckNoError(t, resp)

	th.LoginBasic2()
	_, resp = Client.RemoveUserFromChannel(private.Id, th.BasicUser.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.RemoveUserFromChannel(private.Id, th.BasicUser.Id)
	CheckNoError(t, resp)

	th.LoginBasic()
	th.UpdateUserToNonTeamAdmin(user1, team)
	th.App.InvalidateAllCaches()

	// Test policy does not apply to TE.
	restrictPrivateChannel := *th.App.Config().TeamSettings.RestrictPrivateChannelManageMembers
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.TeamSettings.RestrictPrivateChannelManageMembers = restrictPrivateChannel
		})
	}()
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelManageMembers = model.PERMISSIONS_CHANNEL_ADMIN
	})
	th.App.SetDefaultRolesBasedOnConfig()

	privateChannel := th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_PRIVATE)
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user1.Id)
	CheckNoError(t, resp)
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	_, resp = Client.RemoveUserFromChannel(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	// Add a license
	isLicensed := utils.IsLicensed()
	license := utils.License()
	defer func() {
		utils.SetIsLicensed(isLicensed)
		utils.SetLicense(license)
		th.App.SetDefaultRolesBasedOnConfig()
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictPrivateChannelManageMembers = model.PERMISSIONS_ALL })
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	// Check that a regular channel user can remove other users.
	privateChannel = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_PRIVATE)
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user1.Id)
	CheckNoError(t, resp)
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	_, resp = Client.RemoveUserFromChannel(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	// Test with CHANNEL_ADMIN level permission.
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelManageMembers = model.PERMISSIONS_CHANNEL_ADMIN
	})
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	privateChannel = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_PRIVATE)
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user1.Id)
	CheckNoError(t, resp)
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	_, resp = Client.RemoveUserFromChannel(privateChannel.Id, user2.Id)
	CheckForbiddenStatus(t, resp)

	th.MakeUserChannelAdmin(user1, privateChannel)
	th.App.InvalidateAllCaches()
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()

	_, resp = Client.RemoveUserFromChannel(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	// Test with TEAM_ADMIN level permission.
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelManageMembers = model.PERMISSIONS_TEAM_ADMIN
	})
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	privateChannel = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_PRIVATE)
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user1.Id)
	CheckNoError(t, resp)
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	_, resp = Client.RemoveUserFromChannel(privateChannel.Id, user2.Id)
	CheckForbiddenStatus(t, resp)

	th.UpdateUserToTeamAdmin(user1, team)
	th.App.InvalidateAllCaches()
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()

	_, resp = Client.RemoveUserFromChannel(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	// Test with SYSTEM_ADMIN level permission.
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.TeamSettings.RestrictPrivateChannelManageMembers = model.PERMISSIONS_SYSTEM_ADMIN
	})
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()

	privateChannel = th.CreateChannelWithClient(th.SystemAdminClient, model.CHANNEL_PRIVATE)
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user1.Id)
	CheckNoError(t, resp)
	_, resp = th.SystemAdminClient.AddChannelMember(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)

	_, resp = Client.RemoveUserFromChannel(privateChannel.Id, user2.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.RemoveUserFromChannel(privateChannel.Id, user2.Id)
	CheckNoError(t, resp)
}
