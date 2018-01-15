// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"encoding/base64"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

func TestCreateTeam(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()
	Client := th.Client

	team := &model.Team{Name: GenerateTestUsername(), DisplayName: "Some Team", Type: model.TEAM_OPEN}
	rteam, resp := Client.CreateTeam(team)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	if rteam.Name != team.Name {
		t.Fatal("names did not match")
	}

	if rteam.DisplayName != team.DisplayName {
		t.Fatal("display names did not match")
	}

	if rteam.Type != team.Type {
		t.Fatal("types did not match")
	}

	_, resp = Client.CreateTeam(rteam)
	CheckBadRequestStatus(t, resp)

	rteam.Id = ""
	_, resp = Client.CreateTeam(rteam)
	CheckErrorMessage(t, resp, "store.sql_team.save.domain_exists.app_error")
	CheckBadRequestStatus(t, resp)

	rteam.Name = ""
	_, resp = Client.CreateTeam(rteam)
	CheckErrorMessage(t, resp, "model.team.is_valid.characters.app_error")
	CheckBadRequestStatus(t, resp)

	if r, err := Client.DoApiPost("/teams", "garbage"); err == nil {
		t.Fatal("should have errored")
	} else {
		if r.StatusCode != http.StatusBadRequest {
			t.Log("actual: " + strconv.Itoa(r.StatusCode))
			t.Log("expected: " + strconv.Itoa(http.StatusBadRequest))
			t.Fatal("wrong status code")
		}
	}

	Client.Logout()

	_, resp = Client.CreateTeam(rteam)
	CheckUnauthorizedStatus(t, resp)

	// Update permission
	enableTeamCreation := th.App.Config().TeamSettings.EnableTeamCreation
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { cfg.TeamSettings.EnableTeamCreation = enableTeamCreation })
		th.App.SetDefaultRolesBasedOnConfig()
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { cfg.TeamSettings.EnableTeamCreation = false })
	th.App.SetDefaultRolesBasedOnConfig()

	th.LoginBasic()
	_, resp = Client.CreateTeam(team)
	CheckForbiddenStatus(t, resp)
}

func TestCreateTeamSanitization(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()

	// Non-admin users can create a team, but they become a team admin by doing so

	t.Run("team admin", func(t *testing.T) {
		team := &model.Team{
			DisplayName:    t.Name() + "_1",
			Name:           GenerateTestTeamName(),
			Email:          GenerateTestEmail(),
			Type:           model.TEAM_OPEN,
			AllowedDomains: "simulator.amazonses.com",
		}

		rteam, resp := th.Client.CreateTeam(team)
		CheckNoError(t, resp)
		if rteam.Email == "" {
			t.Fatal("should not have sanitized email")
		} else if rteam.AllowedDomains == "" {
			t.Fatal("should not have sanitized allowed domains")
		}
	})

	t.Run("system admin", func(t *testing.T) {
		team := &model.Team{
			DisplayName:    t.Name() + "_2",
			Name:           GenerateTestTeamName(),
			Email:          GenerateTestEmail(),
			Type:           model.TEAM_OPEN,
			AllowedDomains: "simulator.amazonses.com",
		}

		rteam, resp := th.SystemAdminClient.CreateTeam(team)
		CheckNoError(t, resp)
		if rteam.Email == "" {
			t.Fatal("should not have sanitized email")
		} else if rteam.AllowedDomains == "" {
			t.Fatal("should not have sanitized allowed domains")
		}
	})
}

func TestGetTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam

	rteam, resp := Client.GetTeam(team.Id, "")
	CheckNoError(t, resp)

	if rteam.Id != team.Id {
		t.Fatal("wrong team")
	}

	_, resp = Client.GetTeam("junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeam("", "")
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetTeam(model.NewId(), "")
	CheckNotFoundStatus(t, resp)

	th.LoginTeamAdmin()

	team2 := &model.Team{DisplayName: "Name", Name: GenerateTestTeamName(), Email: GenerateTestEmail(), Type: model.TEAM_OPEN, AllowOpenInvite: false}
	rteam2, _ := Client.CreateTeam(team2)

	team3 := &model.Team{DisplayName: "Name", Name: GenerateTestTeamName(), Email: GenerateTestEmail(), Type: model.TEAM_INVITE, AllowOpenInvite: true}
	rteam3, _ := Client.CreateTeam(team3)

	th.LoginBasic()
	// AllowInviteOpen is false and team is open, and user is not on team
	_, resp = Client.GetTeam(rteam2.Id, "")
	CheckForbiddenStatus(t, resp)

	// AllowInviteOpen is true and team is invite, and user is not on team
	_, resp = Client.GetTeam(rteam3.Id, "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetTeam(team.Id, "")
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.GetTeam(rteam2.Id, "")
	CheckNoError(t, resp)
}

func TestGetTeamSanitization(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()

	team, resp := th.Client.CreateTeam(&model.Team{
		DisplayName:    t.Name() + "_1",
		Name:           GenerateTestTeamName(),
		Email:          GenerateTestEmail(),
		Type:           model.TEAM_OPEN,
		AllowedDomains: "simulator.amazonses.com",
	})
	CheckNoError(t, resp)

	t.Run("team user", func(t *testing.T) {
		th.LinkUserToTeam(th.BasicUser2, team)

		client := th.CreateClient()
		th.LoginBasic2WithClient(client)

		rteam, resp := client.GetTeam(team.Id, "")
		CheckNoError(t, resp)
		if rteam.Email != "" {
			t.Fatal("should've sanitized email")
		} else if rteam.AllowedDomains != "" {
			t.Fatal("should've sanitized allowed domains")
		}
	})

	t.Run("team admin", func(t *testing.T) {
		rteam, resp := th.Client.GetTeam(team.Id, "")
		CheckNoError(t, resp)
		if rteam.Email == "" {
			t.Fatal("should not have sanitized email")
		} else if rteam.AllowedDomains == "" {
			t.Fatal("should not have sanitized allowed domains")
		}
	})

	t.Run("system admin", func(t *testing.T) {
		rteam, resp := th.SystemAdminClient.GetTeam(team.Id, "")
		CheckNoError(t, resp)
		if rteam.Email == "" {
			t.Fatal("should not have sanitized email")
		} else if rteam.AllowedDomains == "" {
			t.Fatal("should not have sanitized allowed domains")
		}
	})
}

func TestGetTeamUnread(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	teamUnread, resp := Client.GetTeamUnread(th.BasicTeam.Id, th.BasicUser.Id)
	CheckNoError(t, resp)
	if teamUnread.TeamId != th.BasicTeam.Id {
		t.Fatal("wrong team id returned for regular user call")
	}

	_, resp = Client.GetTeamUnread("junk", th.BasicUser.Id)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeamUnread(th.BasicTeam.Id, "junk")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeamUnread(model.NewId(), th.BasicUser.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetTeamUnread(th.BasicTeam.Id, model.NewId())
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetTeamUnread(th.BasicTeam.Id, th.BasicUser.Id)
	CheckUnauthorizedStatus(t, resp)

	teamUnread, resp = th.SystemAdminClient.GetTeamUnread(th.BasicTeam.Id, th.BasicUser.Id)
	CheckNoError(t, resp)
	if teamUnread.TeamId != th.BasicTeam.Id {
		t.Fatal("wrong team id returned")
	}
}

func TestUpdateTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	team := &model.Team{DisplayName: "Name", Description: "Some description", AllowOpenInvite: false, InviteId: "inviteid0", Name: "z-z-" + model.NewId() + "a", Email: "success+" + model.NewId() + "@simulator.amazonses.com", Type: model.TEAM_OPEN}
	team, _ = Client.CreateTeam(team)

	team.Description = "updated description"
	uteam, resp := Client.UpdateTeam(team)
	CheckNoError(t, resp)

	if uteam.Description != "updated description" {
		t.Fatal("Update failed")
	}

	team.DisplayName = "Updated Name"
	uteam, resp = Client.UpdateTeam(team)
	CheckNoError(t, resp)

	if uteam.DisplayName != "Updated Name" {
		t.Fatal("Update failed")
	}

	team.AllowOpenInvite = true
	uteam, resp = Client.UpdateTeam(team)
	CheckNoError(t, resp)

	if !uteam.AllowOpenInvite {
		t.Fatal("Update failed")
	}

	team.InviteId = "inviteid1"
	uteam, resp = Client.UpdateTeam(team)
	CheckNoError(t, resp)

	if uteam.InviteId != "inviteid1" {
		t.Fatal("Update failed")
	}

	team.AllowedDomains = "domain"
	uteam, resp = Client.UpdateTeam(team)
	CheckNoError(t, resp)

	if uteam.AllowedDomains != "domain" {
		t.Fatal("Update failed")
	}

	team.Name = "Updated name"
	uteam, resp = Client.UpdateTeam(team)
	CheckNoError(t, resp)

	if uteam.Name == "Updated name" {
		t.Fatal("Should not update name")
	}

	team.Email = "test@domain.com"
	uteam, resp = Client.UpdateTeam(team)
	CheckNoError(t, resp)

	if uteam.Email == "test@domain.com" {
		t.Fatal("Should not update email")
	}

	team.Type = model.TEAM_INVITE
	uteam, resp = Client.UpdateTeam(team)
	CheckNoError(t, resp)

	if uteam.Type == model.TEAM_INVITE {
		t.Fatal("Should not update type")
	}

	originalTeamId := team.Id
	team.Id = model.NewId()

	if r, err := Client.DoApiPut(Client.GetTeamRoute(originalTeamId), team.ToJson()); err != nil {
		t.Fatal(err)
	} else {
		uteam = model.TeamFromJson(r.Body)
	}

	if uteam.Id != originalTeamId {
		t.Fatal("wrong team id")
	}

	team.Id = "fake"
	_, resp = Client.UpdateTeam(team)
	CheckBadRequestStatus(t, resp)

	Client.Logout()
	_, resp = Client.UpdateTeam(team)
	CheckUnauthorizedStatus(t, resp)

	team.Id = originalTeamId
	_, resp = th.SystemAdminClient.UpdateTeam(team)
	CheckNoError(t, resp)
}

func TestUpdateTeamSanitization(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()

	team, resp := th.Client.CreateTeam(&model.Team{
		DisplayName:    t.Name() + "_1",
		Name:           GenerateTestTeamName(),
		Email:          GenerateTestEmail(),
		Type:           model.TEAM_OPEN,
		AllowedDomains: "simulator.amazonses.com",
	})
	CheckNoError(t, resp)

	// Non-admin users cannot update the team

	t.Run("team admin", func(t *testing.T) {
		rteam, resp := th.Client.UpdateTeam(team)
		CheckNoError(t, resp)
		if rteam.Email == "" {
			t.Fatal("should not have sanitized email for admin")
		} else if rteam.AllowedDomains == "" {
			t.Fatal("should not have sanitized allowed domains")
		}
	})

	t.Run("system admin", func(t *testing.T) {
		rteam, resp := th.SystemAdminClient.UpdateTeam(team)
		CheckNoError(t, resp)
		if rteam.Email == "" {
			t.Fatal("should not have sanitized email for admin")
		} else if rteam.AllowedDomains == "" {
			t.Fatal("should not have sanitized allowed domains")
		}
	})
}

func TestPatchTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	team := &model.Team{DisplayName: "Name", Description: "Some description", CompanyName: "Some company name", AllowOpenInvite: false, InviteId: "inviteid0", Name: "z-z-" + model.NewId() + "a", Email: "success+" + model.NewId() + "@simulator.amazonses.com", Type: model.TEAM_OPEN}
	team, _ = Client.CreateTeam(team)

	patch := &model.TeamPatch{}

	patch.DisplayName = model.NewString("Other name")
	patch.Description = model.NewString("Other description")
	patch.CompanyName = model.NewString("Other company name")
	patch.InviteId = model.NewString("inviteid1")
	patch.AllowOpenInvite = model.NewBool(true)

	rteam, resp := Client.PatchTeam(team.Id, patch)
	CheckNoError(t, resp)

	if rteam.DisplayName != "Other name" {
		t.Fatal("DisplayName did not update properly")
	}
	if rteam.Description != "Other description" {
		t.Fatal("Description did not update properly")
	}
	if rteam.CompanyName != "Other company name" {
		t.Fatal("CompanyName did not update properly")
	}
	if rteam.InviteId != "inviteid1" {
		t.Fatal("InviteId did not update properly")
	}
	if !rteam.AllowOpenInvite {
		t.Fatal("AllowOpenInvite did not update properly")
	}

	_, resp = Client.PatchTeam("junk", patch)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.PatchTeam(GenerateTestId(), patch)
	CheckForbiddenStatus(t, resp)

	if r, err := Client.DoApiPut("/teams/"+team.Id+"/patch", "garbage"); err == nil {
		t.Fatal("should have errored")
	} else {
		if r.StatusCode != http.StatusBadRequest {
			t.Log("actual: " + strconv.Itoa(r.StatusCode))
			t.Log("expected: " + strconv.Itoa(http.StatusBadRequest))
			t.Fatal("wrong status code")
		}
	}

	Client.Logout()
	_, resp = Client.PatchTeam(team.Id, patch)
	CheckUnauthorizedStatus(t, resp)

	th.LoginBasic2()
	_, resp = Client.PatchTeam(team.Id, patch)
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.PatchTeam(team.Id, patch)
	CheckNoError(t, resp)
}

func TestPatchTeamSanitization(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()

	team, resp := th.Client.CreateTeam(&model.Team{
		DisplayName:    t.Name() + "_1",
		Name:           GenerateTestTeamName(),
		Email:          GenerateTestEmail(),
		Type:           model.TEAM_OPEN,
		AllowedDomains: "simulator.amazonses.com",
	})
	CheckNoError(t, resp)

	// Non-admin users cannot update the team

	t.Run("team admin", func(t *testing.T) {
		rteam, resp := th.Client.PatchTeam(team.Id, &model.TeamPatch{})
		CheckNoError(t, resp)
		if rteam.Email == "" {
			t.Fatal("should not have sanitized email for admin")
		} else if rteam.AllowedDomains == "" {
			t.Fatal("should not have sanitized allowed domains")
		}
	})

	t.Run("system admin", func(t *testing.T) {
		rteam, resp := th.SystemAdminClient.PatchTeam(team.Id, &model.TeamPatch{})
		CheckNoError(t, resp)
		if rteam.Email == "" {
			t.Fatal("should not have sanitized email for admin")
		} else if rteam.AllowedDomains == "" {
			t.Fatal("should not have sanitized allowed domains")
		}
	})
}

func TestSoftDeleteTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	team := &model.Team{DisplayName: "DisplayName", Name: GenerateTestTeamName(), Email: GenerateTestEmail(), Type: model.TEAM_OPEN}
	team, _ = Client.CreateTeam(team)

	ok, resp := Client.SoftDeleteTeam(team.Id)
	CheckNoError(t, resp)

	if !ok {
		t.Fatal("should have returned true")
	}

	rteam, err := th.App.GetTeam(team.Id)
	if err != nil {
		t.Fatal("should have returned archived team")
	}
	if rteam.DeleteAt == 0 {
		t.Fatal("should have not set to zero")
	}

	ok, resp = Client.SoftDeleteTeam("junk")
	CheckBadRequestStatus(t, resp)

	if ok {
		t.Fatal("should have returned false")
	}

	otherTeam := th.BasicTeam
	_, resp = Client.SoftDeleteTeam(otherTeam.Id)
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.SoftDeleteTeam(otherTeam.Id)
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.SoftDeleteTeam(otherTeam.Id)
	CheckNoError(t, resp)
}

func TestPermanentDeleteTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	team := &model.Team{DisplayName: "DisplayName", Name: GenerateTestTeamName(), Email: GenerateTestEmail(), Type: model.TEAM_OPEN}
	team, _ = Client.CreateTeam(team)

	ok, resp := Client.PermanentDeleteTeam(team.Id)
	CheckNoError(t, resp)

	if !ok {
		t.Fatal("should have returned true")
	}

	// The team is deleted in the background, its only soft deleted at this
	// time
	rteam, err := th.App.GetTeam(team.Id)
	if err != nil {
		t.Fatal("should have returned archived team")
	}
	if rteam.DeleteAt == 0 {
		t.Fatal("should have not set to zero")
	}

	ok, resp = Client.PermanentDeleteTeam("junk")
	CheckBadRequestStatus(t, resp)

	if ok {
		t.Fatal("should have returned false")
	}
}

func TestGetAllTeams(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	team := &model.Team{DisplayName: "Name", Name: GenerateTestTeamName(), Email: GenerateTestEmail(), Type: model.TEAM_OPEN, AllowOpenInvite: true}
	_, resp := Client.CreateTeam(team)
	CheckNoError(t, resp)

	rrteams, resp := Client.GetAllTeams("", 0, 1)
	CheckNoError(t, resp)

	if len(rrteams) != 1 {
		t.Log(len(rrteams))
		t.Fatal("wrong number of teams - should be 1")
	}

	for _, rt := range rrteams {
		if !rt.AllowOpenInvite {
			t.Fatal("not all teams are open")
		}
	}

	rrteams, resp = Client.GetAllTeams("", 0, 10)
	CheckNoError(t, resp)

	for _, rt := range rrteams {
		if !rt.AllowOpenInvite {
			t.Fatal("not all teams are open")
		}
	}

	rrteams1, resp := Client.GetAllTeams("", 1, 0)
	CheckNoError(t, resp)

	if len(rrteams1) != 0 {
		t.Fatal("wrong number of teams - should be 0")
	}

	rrteams2, resp := th.SystemAdminClient.GetAllTeams("", 1, 1)
	CheckNoError(t, resp)

	if len(rrteams2) != 1 {
		t.Fatal("wrong number of teams - should be 1")
	}

	rrteams2, resp = Client.GetAllTeams("", 1, 0)
	CheckNoError(t, resp)

	if len(rrteams2) != 0 {
		t.Fatal("wrong number of teams - should be 0")
	}

	Client.Logout()
	_, resp = Client.GetAllTeams("", 1, 10)
	CheckUnauthorizedStatus(t, resp)
}

func TestGetAllTeamsSanitization(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()

	team, resp := th.Client.CreateTeam(&model.Team{
		DisplayName:     t.Name() + "_1",
		Name:            GenerateTestTeamName(),
		Email:           GenerateTestEmail(),
		Type:            model.TEAM_OPEN,
		AllowedDomains:  "simulator.amazonses.com",
		AllowOpenInvite: true,
	})
	CheckNoError(t, resp)
	team2, resp := th.SystemAdminClient.CreateTeam(&model.Team{
		DisplayName:     t.Name() + "_2",
		Name:            GenerateTestTeamName(),
		Email:           GenerateTestEmail(),
		Type:            model.TEAM_OPEN,
		AllowedDomains:  "simulator.amazonses.com",
		AllowOpenInvite: true,
	})
	CheckNoError(t, resp)

	// This may not work if the server has over 1000 open teams on it

	t.Run("team admin/non-admin", func(t *testing.T) {
		teamFound := false
		team2Found := false

		rteams, resp := th.Client.GetAllTeams("", 0, 1000)
		CheckNoError(t, resp)
		for _, rteam := range rteams {
			if rteam.Id == team.Id {
				teamFound = true
				if rteam.Email == "" {
					t.Fatal("should not have sanitized email for team admin")
				} else if rteam.AllowedDomains == "" {
					t.Fatal("should not have sanitized allowed domains for team admin")
				}
			} else if rteam.Id == team2.Id {
				team2Found = true
				if rteam.Email != "" {
					t.Fatal("should've sanitized email for non-admin")
				} else if rteam.AllowedDomains != "" {
					t.Fatal("should've sanitized allowed domains for non-admin")
				}
			}
		}

		if !teamFound || !team2Found {
			t.Fatal("wasn't returned the expected teams so the test wasn't run correctly")
		}
	})

	t.Run("system admin", func(t *testing.T) {
		rteams, resp := th.SystemAdminClient.GetAllTeams("", 0, 1000)
		CheckNoError(t, resp)
		for _, rteam := range rteams {
			if rteam.Id != team.Id && rteam.Id != team2.Id {
				continue
			}

			if rteam.Email == "" {
				t.Fatal("should not have sanitized email")
			} else if rteam.AllowedDomains == "" {
				t.Fatal("should not have sanitized allowed domains")
			}
		}
	})
}

func TestGetTeamByName(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam

	rteam, resp := Client.GetTeamByName(team.Name, "")
	CheckNoError(t, resp)

	if rteam.Name != team.Name {
		t.Fatal("wrong team")
	}

	_, resp = Client.GetTeamByName("junk", "")
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetTeamByName("", "")
	CheckNotFoundStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetTeamByName(team.Name, "")
	CheckUnauthorizedStatus(t, resp)

	_, resp = th.SystemAdminClient.GetTeamByName(team.Name, "")
	CheckNoError(t, resp)

	th.LoginTeamAdmin()

	team2 := &model.Team{DisplayName: "Name", Name: GenerateTestTeamName(), Email: GenerateTestEmail(), Type: model.TEAM_OPEN, AllowOpenInvite: false}
	rteam2, _ := Client.CreateTeam(team2)

	team3 := &model.Team{DisplayName: "Name", Name: GenerateTestTeamName(), Email: GenerateTestEmail(), Type: model.TEAM_INVITE, AllowOpenInvite: true}
	rteam3, _ := Client.CreateTeam(team3)

	th.LoginBasic()
	// AllowInviteOpen is false and team is open, and user is not on team
	_, resp = Client.GetTeamByName(rteam2.Name, "")
	CheckForbiddenStatus(t, resp)

	// AllowInviteOpen is true and team is invite only, and user is not on team
	_, resp = Client.GetTeamByName(rteam3.Name, "")
	CheckForbiddenStatus(t, resp)
}

func TestGetTeamByNameSanitization(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()

	team, resp := th.Client.CreateTeam(&model.Team{
		DisplayName:    t.Name() + "_1",
		Name:           GenerateTestTeamName(),
		Email:          GenerateTestEmail(),
		Type:           model.TEAM_OPEN,
		AllowedDomains: "simulator.amazonses.com",
	})
	CheckNoError(t, resp)

	t.Run("team user", func(t *testing.T) {
		th.LinkUserToTeam(th.BasicUser2, team)

		client := th.CreateClient()
		th.LoginBasic2WithClient(client)

		rteam, resp := client.GetTeamByName(team.Name, "")
		CheckNoError(t, resp)
		if rteam.Email != "" {
			t.Fatal("should've sanitized email")
		} else if rteam.AllowedDomains != "" {
			t.Fatal("should've sanitized allowed domains")
		}
	})

	t.Run("team admin/non-admin", func(t *testing.T) {
		rteam, resp := th.Client.GetTeamByName(team.Name, "")
		CheckNoError(t, resp)
		if rteam.Email == "" {
			t.Fatal("should not have sanitized email")
		} else if rteam.AllowedDomains == "" {
			t.Fatal("should not have sanitized allowed domains")
		}
	})

	t.Run("system admin", func(t *testing.T) {
		rteam, resp := th.SystemAdminClient.GetTeamByName(team.Name, "")
		CheckNoError(t, resp)
		if rteam.Email == "" {
			t.Fatal("should not have sanitized email")
		} else if rteam.AllowedDomains == "" {
			t.Fatal("should not have sanitized allowed domains")
		}
	})
}

func TestSearchAllTeams(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	oTeam := th.BasicTeam
	oTeam.AllowOpenInvite = true

	if updatedTeam, err := th.App.UpdateTeam(oTeam); err != nil {
		t.Fatal(err)
	} else {
		oTeam.UpdateAt = updatedTeam.UpdateAt
	}

	pTeam := &model.Team{DisplayName: "PName", Name: GenerateTestTeamName(), Email: GenerateTestEmail(), Type: model.TEAM_INVITE}
	Client.CreateTeam(pTeam)

	rteams, resp := Client.SearchTeams(&model.TeamSearch{Term: oTeam.Name})
	CheckNoError(t, resp)

	if len(rteams) != 1 {
		t.Fatal("should have returned 1 team")
	}

	if oTeam.Id != rteams[0].Id {
		t.Fatal("invalid team")
	}

	rteams, resp = Client.SearchTeams(&model.TeamSearch{Term: oTeam.DisplayName})
	CheckNoError(t, resp)

	if len(rteams) != 1 {
		t.Fatal("should have returned 1 team")
	}

	if rteams[0].Id != oTeam.Id {
		t.Fatal("invalid team")
	}

	rteams, resp = Client.SearchTeams(&model.TeamSearch{Term: pTeam.Name})
	CheckNoError(t, resp)

	if len(rteams) != 0 {
		t.Fatal("should have not returned team")
	}

	rteams, resp = Client.SearchTeams(&model.TeamSearch{Term: pTeam.DisplayName})
	CheckNoError(t, resp)

	if len(rteams) != 0 {
		t.Fatal("should have not returned team")
	}

	rteams, resp = th.SystemAdminClient.SearchTeams(&model.TeamSearch{Term: oTeam.Name})
	CheckNoError(t, resp)

	if len(rteams) != 1 {
		t.Fatal("should have returned 1 team")
	}

	rteams, resp = th.SystemAdminClient.SearchTeams(&model.TeamSearch{Term: pTeam.DisplayName})
	CheckNoError(t, resp)

	if len(rteams) != 1 {
		t.Fatal("should have returned 1 team")
	}

	rteams, resp = Client.SearchTeams(&model.TeamSearch{Term: "junk"})
	CheckNoError(t, resp)

	if len(rteams) != 0 {
		t.Fatal("should have not returned team")
	}

	Client.Logout()

	rteams, resp = Client.SearchTeams(&model.TeamSearch{Term: pTeam.Name})
	CheckUnauthorizedStatus(t, resp)

	rteams, resp = Client.SearchTeams(&model.TeamSearch{Term: pTeam.DisplayName})
	CheckUnauthorizedStatus(t, resp)
}

func TestSearchAllTeamsSanitization(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()

	team, resp := th.Client.CreateTeam(&model.Team{
		DisplayName:    t.Name() + "_1",
		Name:           GenerateTestTeamName(),
		Email:          GenerateTestEmail(),
		Type:           model.TEAM_OPEN,
		AllowedDomains: "simulator.amazonses.com",
	})
	CheckNoError(t, resp)
	team2, resp := th.Client.CreateTeam(&model.Team{
		DisplayName:    t.Name() + "_2",
		Name:           GenerateTestTeamName(),
		Email:          GenerateTestEmail(),
		Type:           model.TEAM_OPEN,
		AllowedDomains: "simulator.amazonses.com",
	})
	CheckNoError(t, resp)

	t.Run("non-team user", func(t *testing.T) {
		client := th.CreateClient()
		th.LoginBasic2WithClient(client)

		rteams, resp := client.SearchTeams(&model.TeamSearch{Term: t.Name()})
		CheckNoError(t, resp)
		for _, rteam := range rteams {
			if rteam.Email != "" {
				t.Fatal("should've sanitized email")
			} else if rteam.AllowedDomains != "" {
				t.Fatal("should've sanitized allowed domains")
			}
		}
	})

	t.Run("team user", func(t *testing.T) {
		th.LinkUserToTeam(th.BasicUser2, team)

		client := th.CreateClient()
		th.LoginBasic2WithClient(client)

		rteams, resp := client.SearchTeams(&model.TeamSearch{Term: t.Name()})
		CheckNoError(t, resp)
		for _, rteam := range rteams {
			if rteam.Email != "" {
				t.Fatal("should've sanitized email")
			} else if rteam.AllowedDomains != "" {
				t.Fatal("should've sanitized allowed domains")
			}
		}
	})

	t.Run("team admin", func(t *testing.T) {
		rteams, resp := th.Client.SearchTeams(&model.TeamSearch{Term: t.Name()})
		CheckNoError(t, resp)
		for _, rteam := range rteams {
			if rteam.Id == team.Id || rteam.Id == team2.Id || rteam.Id == th.BasicTeam.Id {
				if rteam.Email == "" {
					t.Fatal("should not have sanitized email")
				} else if rteam.AllowedDomains == "" {
					t.Fatal("should not have sanitized allowed domains")
				}
			}
		}
	})

	t.Run("system admin", func(t *testing.T) {
		rteams, resp := th.SystemAdminClient.SearchTeams(&model.TeamSearch{Term: t.Name()})
		CheckNoError(t, resp)
		for _, rteam := range rteams {
			if rteam.Email == "" {
				t.Fatal("should not have sanitized email")
			} else if rteam.AllowedDomains == "" {
				t.Fatal("should not have sanitized allowed domains")
			}
		}
	})
}

func TestGetTeamsForUser(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	team2 := &model.Team{DisplayName: "Name", Name: GenerateTestTeamName(), Email: GenerateTestEmail(), Type: model.TEAM_INVITE}
	rteam2, _ := Client.CreateTeam(team2)

	teams, resp := Client.GetTeamsForUser(th.BasicUser.Id, "")
	CheckNoError(t, resp)

	if len(teams) != 2 {
		t.Fatal("wrong number of teams")
	}

	found1 := false
	found2 := false
	for _, t := range teams {
		if t.Id == th.BasicTeam.Id {
			found1 = true
		} else if t.Id == rteam2.Id {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Fatal("missing team")
	}

	_, resp = Client.GetTeamsForUser("junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeamsForUser(model.NewId(), "")
	CheckForbiddenStatus(t, resp)

	_, resp = Client.GetTeamsForUser(th.BasicUser2.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetTeamsForUser(th.BasicUser2.Id, "")
	CheckNoError(t, resp)
}

func TestGetTeamsForUserSanitization(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()

	team, resp := th.Client.CreateTeam(&model.Team{
		DisplayName:    t.Name() + "_1",
		Name:           GenerateTestTeamName(),
		Email:          GenerateTestEmail(),
		Type:           model.TEAM_OPEN,
		AllowedDomains: "simulator.amazonses.com",
	})
	CheckNoError(t, resp)
	team2, resp := th.Client.CreateTeam(&model.Team{
		DisplayName:    t.Name() + "_2",
		Name:           GenerateTestTeamName(),
		Email:          GenerateTestEmail(),
		Type:           model.TEAM_OPEN,
		AllowedDomains: "simulator.amazonses.com",
	})
	CheckNoError(t, resp)

	t.Run("team user", func(t *testing.T) {
		th.LinkUserToTeam(th.BasicUser2, team)
		th.LinkUserToTeam(th.BasicUser2, team2)

		client := th.CreateClient()
		th.LoginBasic2WithClient(client)

		rteams, resp := client.GetTeamsForUser(th.BasicUser2.Id, "")
		CheckNoError(t, resp)
		for _, rteam := range rteams {
			if rteam.Id != team.Id && rteam.Id != team2.Id {
				continue
			}

			if rteam.Email != "" {
				t.Fatal("should've sanitized email")
			} else if rteam.AllowedDomains != "" {
				t.Fatal("should've sanitized allowed domains")
			}
		}
	})

	t.Run("team admin", func(t *testing.T) {
		rteams, resp := th.Client.GetTeamsForUser(th.BasicUser.Id, "")
		CheckNoError(t, resp)
		for _, rteam := range rteams {
			if rteam.Id != team.Id && rteam.Id != team2.Id {
				continue
			}

			if rteam.Email == "" {
				t.Fatal("should not have sanitized email")
			} else if rteam.AllowedDomains == "" {
				t.Fatal("should not have sanitized allowed domains")
			}
		}
	})

	t.Run("system admin", func(t *testing.T) {
		rteams, resp := th.SystemAdminClient.GetTeamsForUser(th.BasicUser.Id, "")
		CheckNoError(t, resp)
		for _, rteam := range rteams {
			if rteam.Id != team.Id && rteam.Id != team2.Id {
				continue
			}

			if rteam.Email == "" {
				t.Fatal("should not have sanitized email")
			} else if rteam.AllowedDomains == "" {
				t.Fatal("should not have sanitized allowed domains")
			}
		}
	})
}

func TestGetTeamMember(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam
	user := th.BasicUser

	rmember, resp := Client.GetTeamMember(team.Id, user.Id, "")
	CheckNoError(t, resp)

	if rmember.TeamId != team.Id {
		t.Fatal("wrong team id")
	}

	if rmember.UserId != user.Id {
		t.Fatal("wrong team id")
	}

	_, resp = Client.GetTeamMember("junk", user.Id, "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeamMember(team.Id, "junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeamMember("junk", "junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeamMember(team.Id, model.NewId(), "")
	CheckNotFoundStatus(t, resp)

	_, resp = Client.GetTeamMember(model.NewId(), user.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetTeamMember(team.Id, user.Id, "")
	CheckNoError(t, resp)
}

func TestGetTeamMembers(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam
	userNotMember := th.CreateUser()

	rmembers, resp := Client.GetTeamMembers(team.Id, 0, 100, "")
	CheckNoError(t, resp)

	t.Logf("rmembers count %v\n", len(rmembers))

	if len(rmembers) == 0 {
		t.Fatal("should have results")
	}

	for _, rmember := range rmembers {
		if rmember.TeamId != team.Id || rmember.UserId == userNotMember.Id {
			t.Fatal("user should be a member of team")
		}
	}

	rmembers, resp = Client.GetTeamMembers(team.Id, 0, 1, "")
	CheckNoError(t, resp)
	if len(rmembers) != 1 {
		t.Fatal("should be 1 per page")
	}

	rmembers, resp = Client.GetTeamMembers(team.Id, 1, 1, "")
	CheckNoError(t, resp)
	if len(rmembers) != 1 {
		t.Fatal("should be 1 per page")
	}

	rmembers, resp = Client.GetTeamMembers(team.Id, 10000, 100, "")
	CheckNoError(t, resp)
	if len(rmembers) != 0 {
		t.Fatal("should be no member")
	}

	_, resp = Client.GetTeamMembers("junk", 0, 100, "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeamMembers(model.NewId(), 0, 100, "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	rmembers, resp = Client.GetTeamMembers(team.Id, 0, 1, "")
	CheckUnauthorizedStatus(t, resp)

	rmembers, resp = th.SystemAdminClient.GetTeamMembers(team.Id, 0, 100, "")
	CheckNoError(t, resp)
}

func TestGetTeamMembersForUser(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	members, resp := Client.GetTeamMembersForUser(th.BasicUser.Id, "")
	CheckNoError(t, resp)

	found := false
	for _, m := range members {
		if m.TeamId == th.BasicTeam.Id {
			found = true
		}
	}

	if !found {
		t.Fatal("missing team member")
	}

	_, resp = Client.GetTeamMembersForUser("junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeamMembersForUser(model.NewId(), "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetTeamMembersForUser(th.BasicUser.Id, "")
	CheckUnauthorizedStatus(t, resp)

	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetTeamMembersForUser(th.BasicUser.Id, "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetTeamMembersForUser(th.BasicUser.Id, "")
	CheckNoError(t, resp)
}

func TestGetTeamMembersByIds(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()
	Client := th.Client

	tm, resp := Client.GetTeamMembersByIds(th.BasicTeam.Id, []string{th.BasicUser.Id})
	CheckNoError(t, resp)

	if tm[0].UserId != th.BasicUser.Id {
		t.Fatal("returned wrong user")
	}

	_, resp = Client.GetTeamMembersByIds(th.BasicTeam.Id, []string{})
	CheckBadRequestStatus(t, resp)

	tm1, resp := Client.GetTeamMembersByIds(th.BasicTeam.Id, []string{"junk"})
	CheckNoError(t, resp)
	if len(tm1) > 0 {
		t.Fatal("no users should be returned")
	}

	tm1, resp = Client.GetTeamMembersByIds(th.BasicTeam.Id, []string{"junk", th.BasicUser.Id})
	CheckNoError(t, resp)
	if len(tm1) != 1 {
		t.Fatal("1 user should be returned")
	}

	tm1, resp = Client.GetTeamMembersByIds("junk", []string{th.BasicUser.Id})
	CheckBadRequestStatus(t, resp)

	tm1, resp = Client.GetTeamMembersByIds(model.NewId(), []string{th.BasicUser.Id})
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetTeamMembersByIds(th.BasicTeam.Id, []string{th.BasicUser.Id})
	CheckUnauthorizedStatus(t, resp)
}

func TestAddTeamMember(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam
	otherUser := th.CreateUser()

	if err := th.App.RemoveUserFromTeam(th.BasicTeam.Id, th.BasicUser2.Id, ""); err != nil {
		t.Fatalf(err.Error())
	}

	// Regular user can't add a member to a team they don't belong to.
	th.LoginBasic2()
	tm, resp := Client.AddTeamMember(team.Id, otherUser.Id)
	CheckForbiddenStatus(t, resp)
	if resp.Error == nil {
		t.Fatalf("Error is nhul")
	}
	Client.Logout()

	// Regular user can add a member to a team they belong to.
	th.LoginBasic()
	tm, resp = Client.AddTeamMember(team.Id, otherUser.Id)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	// Check all the returned data.
	if tm == nil {
		t.Fatal("should have returned team member")
	}

	if tm.UserId != otherUser.Id {
		t.Fatal("user ids should have matched")
	}

	if tm.TeamId != team.Id {
		t.Fatal("team ids should have matched")
	}

	// Check with various invalid requests.
	tm, resp = Client.AddTeamMember(team.Id, "junk")
	CheckBadRequestStatus(t, resp)

	if tm != nil {
		t.Fatal("should have not returned team member")
	}

	_, resp = Client.AddTeamMember("junk", otherUser.Id)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.AddTeamMember(GenerateTestId(), otherUser.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.AddTeamMember(team.Id, GenerateTestId())
	CheckNotFoundStatus(t, resp)

	Client.Logout()

	// Check effects of config and license changes.
	restrictTeamInvite := *th.App.Config().TeamSettings.RestrictTeamInvite
	isLicensed := utils.IsLicensed()
	license := utils.License()
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = restrictTeamInvite })
		utils.SetIsLicensed(isLicensed)
		utils.SetLicense(license)
		th.App.SetDefaultRolesBasedOnConfig()
	}()

	// Set the config so that only team admins can add a user to a team.
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = model.PERMISSIONS_TEAM_ADMIN })
	th.App.SetDefaultRolesBasedOnConfig()
	th.LoginBasic()

	// Test without the EE license to see that the permission restriction is ignored.
	_, resp = Client.AddTeamMember(team.Id, otherUser.Id)
	CheckNoError(t, resp)

	// Add an EE license.
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()
	th.LoginBasic()

	// Check that a regular user can't add someone to the team.
	_, resp = Client.AddTeamMember(team.Id, otherUser.Id)
	CheckForbiddenStatus(t, resp)

	// Update user to team admin
	th.UpdateUserToTeamAdmin(th.BasicUser, th.BasicTeam)
	th.App.InvalidateAllCaches()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = model.PERMISSIONS_TEAM_ADMIN })
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()
	th.LoginBasic()

	// Should work as a team admin.
	_, resp = Client.AddTeamMember(team.Id, otherUser.Id)
	CheckNoError(t, resp)

	// Change permission level to System Admin
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = model.PERMISSIONS_SYSTEM_ADMIN })
	th.App.SetDefaultRolesBasedOnConfig()

	// Should not work as team admin.
	_, resp = Client.AddTeamMember(team.Id, otherUser.Id)
	CheckForbiddenStatus(t, resp)

	// Should work as system admin.
	_, resp = th.SystemAdminClient.AddTeamMember(team.Id, otherUser.Id)
	CheckNoError(t, resp)

	// Change permission level to All
	th.UpdateUserToNonTeamAdmin(th.BasicUser, th.BasicTeam)
	th.App.InvalidateAllCaches()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = model.PERMISSIONS_ALL })
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()
	th.LoginBasic()

	// Should work as a regular user.
	_, resp = Client.AddTeamMember(team.Id, otherUser.Id)
	CheckNoError(t, resp)

	// Reset config and license.
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = restrictTeamInvite })
	utils.SetIsLicensed(isLicensed)
	utils.SetLicense(license)
	th.App.SetDefaultRolesBasedOnConfig()
	th.LoginBasic()

	// by hash and data
	Client.Login(otherUser.Email, otherUser.Password)

	dataObject := make(map[string]string)
	dataObject["time"] = fmt.Sprintf("%v", model.GetMillis())
	dataObject["id"] = team.Id

	data := model.MapToJson(dataObject)
	hashed := utils.HashSha256(fmt.Sprintf("%v:%v", data, th.App.Config().EmailSettings.InviteSalt))

	tm, resp = Client.AddTeamMemberFromInvite(hashed, data, "")
	CheckNoError(t, resp)

	if tm == nil {
		t.Fatal("should have returned team member")
	}

	if tm.UserId != otherUser.Id {
		t.Fatal("user ids should have matched")
	}

	if tm.TeamId != team.Id {
		t.Fatal("team ids should have matched")
	}

	tm, resp = Client.AddTeamMemberFromInvite("junk", data, "")
	CheckBadRequestStatus(t, resp)

	if tm != nil {
		t.Fatal("should have not returned team member")
	}

	_, resp = Client.AddTeamMemberFromInvite(hashed, "junk", "")
	CheckBadRequestStatus(t, resp)

	// expired data of more than 50 hours
	dataObject["time"] = fmt.Sprintf("%v", model.GetMillis()-1000*60*60*50)
	data = model.MapToJson(dataObject)
	hashed = utils.HashSha256(fmt.Sprintf("%v:%v", data, th.App.Config().EmailSettings.InviteSalt))

	tm, resp = Client.AddTeamMemberFromInvite(hashed, data, "")
	CheckBadRequestStatus(t, resp)

	// invalid team id
	dataObject["id"] = GenerateTestId()
	data = model.MapToJson(dataObject)
	hashed = utils.HashSha256(fmt.Sprintf("%v:%v", data, th.App.Config().EmailSettings.InviteSalt))

	tm, resp = Client.AddTeamMemberFromInvite(hashed, data, "")
	CheckBadRequestStatus(t, resp)

	// by invite_id
	Client.Login(otherUser.Email, otherUser.Password)

	tm, resp = Client.AddTeamMemberFromInvite("", "", team.InviteId)
	CheckNoError(t, resp)

	if tm == nil {
		t.Fatal("should have returned team member")
	}

	if tm.UserId != otherUser.Id {
		t.Fatal("user ids should have matched")
	}

	if tm.TeamId != team.Id {
		t.Fatal("team ids should have matched")
	}

	tm, resp = Client.AddTeamMemberFromInvite("", "", "junk")
	CheckNotFoundStatus(t, resp)

	if tm != nil {
		t.Fatal("should have not returned team member")
	}
}

func TestAddTeamMembers(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam
	otherUser := th.CreateUser()
	userList := []string{
		otherUser.Id,
	}

	if err := th.App.RemoveUserFromTeam(th.BasicTeam.Id, th.BasicUser2.Id, ""); err != nil {
		t.Fatalf(err.Error())
	}

	// Regular user can't add a member to a team they don't belong to.
	th.LoginBasic2()
	tm, resp := Client.AddTeamMembers(team.Id, userList)
	CheckForbiddenStatus(t, resp)
	Client.Logout()

	// Regular user can add a member to a team they belong to.
	th.LoginBasic()
	tm, resp = Client.AddTeamMembers(team.Id, userList)
	CheckNoError(t, resp)
	CheckCreatedStatus(t, resp)

	// Check all the returned data.
	if tm[0] == nil {
		t.Fatal("should have returned team member")
	}

	if tm[0].UserId != otherUser.Id {
		t.Fatal("user ids should have matched")
	}

	if tm[0].TeamId != team.Id {
		t.Fatal("team ids should have matched")
	}

	// Check with various invalid requests.
	_, resp = Client.AddTeamMembers("junk", userList)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.AddTeamMembers(GenerateTestId(), userList)
	CheckForbiddenStatus(t, resp)

	testUserList := append(userList, GenerateTestId())
	_, resp = Client.AddTeamMembers(team.Id, testUserList)
	CheckNotFoundStatus(t, resp)

	// Test with many users.
	for i := 0; i < 25; i++ {
		testUserList = append(testUserList, GenerateTestId())
	}
	_, resp = Client.AddTeamMembers(team.Id, testUserList)
	CheckBadRequestStatus(t, resp)

	Client.Logout()

	// Check effects of config and license changes.
	restrictTeamInvite := *th.App.Config().TeamSettings.RestrictTeamInvite
	isLicensed := utils.IsLicensed()
	license := utils.License()
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = restrictTeamInvite })
		utils.SetIsLicensed(isLicensed)
		utils.SetLicense(license)
		th.App.SetDefaultRolesBasedOnConfig()
	}()

	// Set the config so that only team admins can add a user to a team.
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = model.PERMISSIONS_TEAM_ADMIN })
	th.App.SetDefaultRolesBasedOnConfig()
	th.LoginBasic()

	// Test without the EE license to see that the permission restriction is ignored.
	_, resp = Client.AddTeamMembers(team.Id, userList)
	CheckNoError(t, resp)

	// Add an EE license.
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()
	th.LoginBasic()

	// Check that a regular user can't add someone to the team.
	_, resp = Client.AddTeamMembers(team.Id, userList)
	CheckForbiddenStatus(t, resp)

	// Update user to team admin
	th.UpdateUserToTeamAdmin(th.BasicUser, th.BasicTeam)
	th.App.InvalidateAllCaches()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = model.PERMISSIONS_TEAM_ADMIN })
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()
	th.LoginBasic()

	// Should work as a team admin.
	_, resp = Client.AddTeamMembers(team.Id, userList)
	CheckNoError(t, resp)

	// Change permission level to System Admin
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = model.PERMISSIONS_SYSTEM_ADMIN })
	th.App.SetDefaultRolesBasedOnConfig()

	// Should not work as team admin.
	_, resp = Client.AddTeamMembers(team.Id, userList)
	CheckForbiddenStatus(t, resp)

	// Should work as system admin.
	_, resp = th.SystemAdminClient.AddTeamMembers(team.Id, userList)
	CheckNoError(t, resp)

	// Change permission level to All
	th.UpdateUserToNonTeamAdmin(th.BasicUser, th.BasicTeam)
	th.App.InvalidateAllCaches()
	th.App.UpdateConfig(func(cfg *model.Config) { *cfg.TeamSettings.RestrictTeamInvite = model.PERMISSIONS_ALL })
	utils.SetIsLicensed(true)
	utils.SetLicense(&model.License{Features: &model.Features{}})
	utils.License().Features.SetDefaults()
	th.App.SetDefaultRolesBasedOnConfig()
	th.LoginBasic()

	// Should work as a regular user.
	_, resp = Client.AddTeamMembers(team.Id, userList)
	CheckNoError(t, resp)
}

func TestRemoveTeamMember(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	pass, resp := Client.RemoveTeamMember(th.BasicTeam.Id, th.BasicUser.Id)
	CheckNoError(t, resp)

	if !pass {
		t.Fatal("should have passed")
	}

	_, resp = th.SystemAdminClient.AddTeamMember(th.BasicTeam.Id, th.BasicUser.Id)
	CheckNoError(t, resp)

	_, resp = Client.RemoveTeamMember(th.BasicTeam.Id, "junk")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.RemoveTeamMember("junk", th.BasicUser2.Id)
	CheckBadRequestStatus(t, resp)

	_, resp = Client.RemoveTeamMember(th.BasicTeam.Id, th.BasicUser2.Id)
	CheckForbiddenStatus(t, resp)

	_, resp = Client.RemoveTeamMember(model.NewId(), th.BasicUser.Id)
	CheckNotFoundStatus(t, resp)

	_, resp = th.SystemAdminClient.RemoveTeamMember(th.BasicTeam.Id, th.BasicUser.Id)
	CheckNoError(t, resp)
}

func TestGetTeamStats(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam

	rstats, resp := Client.GetTeamStats(team.Id, "")
	CheckNoError(t, resp)

	if rstats.TeamId != team.Id {
		t.Fatal("wrong team id")
	}

	if rstats.TotalMemberCount != 3 {
		t.Fatal("wrong count")
	}

	if rstats.ActiveMemberCount != 3 {
		t.Fatal("wrong count")
	}

	_, resp = Client.GetTeamStats("junk", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeamStats(model.NewId(), "")
	CheckForbiddenStatus(t, resp)

	_, resp = th.SystemAdminClient.GetTeamStats(team.Id, "")
	CheckNoError(t, resp)

	// deactivate BasicUser2
	th.UpdateActiveUser(th.BasicUser2, false)

	rstats, resp = th.SystemAdminClient.GetTeamStats(team.Id, "")
	CheckNoError(t, resp)

	if rstats.TotalMemberCount != 3 {
		t.Fatal("wrong count")
	}

	if rstats.ActiveMemberCount != 2 {
		t.Fatal("wrong count")
	}

	// login with different user and test if forbidden
	user := th.CreateUser()
	Client.Login(user.Email, user.Password)
	_, resp = Client.GetTeamStats(th.BasicTeam.Id, "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetTeamStats(th.BasicTeam.Id, "")
	CheckUnauthorizedStatus(t, resp)
}

func TestUpdateTeamMemberRoles(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	SystemAdminClient := th.SystemAdminClient

	const TEAM_MEMBER = "team_user"
	const TEAM_ADMIN = "team_user team_admin"

	// user 1 tries to promote user 2
	ok, resp := Client.UpdateTeamMemberRoles(th.BasicTeam.Id, th.BasicUser2.Id, TEAM_ADMIN)
	CheckForbiddenStatus(t, resp)
	if ok {
		t.Fatal("should have returned false")
	}

	// user 1 tries to promote himself
	_, resp = Client.UpdateTeamMemberRoles(th.BasicTeam.Id, th.BasicUser.Id, TEAM_ADMIN)
	CheckForbiddenStatus(t, resp)

	// user 1 tries to demote someone
	_, resp = Client.UpdateTeamMemberRoles(th.BasicTeam.Id, th.SystemAdminUser.Id, TEAM_MEMBER)
	CheckForbiddenStatus(t, resp)

	// system admin promotes user 1
	ok, resp = SystemAdminClient.UpdateTeamMemberRoles(th.BasicTeam.Id, th.BasicUser.Id, TEAM_ADMIN)
	CheckNoError(t, resp)
	if !ok {
		t.Fatal("should have returned true")
	}

	// user 1 (team admin) promotes user 2
	_, resp = Client.UpdateTeamMemberRoles(th.BasicTeam.Id, th.BasicUser2.Id, TEAM_ADMIN)
	CheckNoError(t, resp)

	// user 1 (team admin) demotes user 2 (team admin)
	_, resp = Client.UpdateTeamMemberRoles(th.BasicTeam.Id, th.BasicUser2.Id, TEAM_MEMBER)
	CheckNoError(t, resp)

	// user 1 (team admin) tries to demote system admin (not member of a team)
	_, resp = Client.UpdateTeamMemberRoles(th.BasicTeam.Id, th.SystemAdminUser.Id, TEAM_MEMBER)
	CheckBadRequestStatus(t, resp)

	// user 1 (team admin) demotes system admin (member of a team)
	th.LinkUserToTeam(th.SystemAdminUser, th.BasicTeam)
	_, resp = Client.UpdateTeamMemberRoles(th.BasicTeam.Id, th.SystemAdminUser.Id, TEAM_MEMBER)
	CheckNoError(t, resp)
	// Note from API v3
	// Note to anyone who thinks this (above) test is wrong:
	// This operation will not affect the system admin's permissions because they have global access to all teams.
	// Their team level permissions are irrelavent. A team admin should be able to manage team level permissions.

	// System admins should be able to manipulate permission no matter what their team level permissions are.
	// system admin promotes user 2
	_, resp = SystemAdminClient.UpdateTeamMemberRoles(th.BasicTeam.Id, th.BasicUser2.Id, TEAM_ADMIN)
	CheckNoError(t, resp)

	// system admin demotes user 2 (team admin)
	_, resp = SystemAdminClient.UpdateTeamMemberRoles(th.BasicTeam.Id, th.BasicUser2.Id, TEAM_MEMBER)
	CheckNoError(t, resp)

	// user 1 (team admin) tries to promote himself to a random team
	_, resp = Client.UpdateTeamMemberRoles(model.NewId(), th.BasicUser.Id, TEAM_ADMIN)
	CheckForbiddenStatus(t, resp)

	// user 1 (team admin) tries to promote a random user
	_, resp = Client.UpdateTeamMemberRoles(th.BasicTeam.Id, model.NewId(), TEAM_ADMIN)
	CheckBadRequestStatus(t, resp)

	// user 1 (team admin) tries to promote invalid team permission
	_, resp = Client.UpdateTeamMemberRoles(th.BasicTeam.Id, th.BasicUser.Id, "junk")
	CheckBadRequestStatus(t, resp)

	// user 1 (team admin) demotes himself
	_, resp = Client.UpdateTeamMemberRoles(th.BasicTeam.Id, th.BasicUser.Id, TEAM_MEMBER)
	CheckNoError(t, resp)
}

func TestGetMyTeamsUnread(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client

	user := th.BasicUser
	Client.Login(user.Email, user.Password)

	teams, resp := Client.GetTeamsUnreadForUser(user.Id, "")
	CheckNoError(t, resp)
	if len(teams) == 0 {
		t.Fatal("should have results")
	}

	teams, resp = Client.GetTeamsUnreadForUser(user.Id, th.BasicTeam.Id)
	CheckNoError(t, resp)
	if len(teams) != 0 {
		t.Fatal("should not have results")
	}

	_, resp = Client.GetTeamsUnreadForUser("fail", "")
	CheckBadRequestStatus(t, resp)

	_, resp = Client.GetTeamsUnreadForUser(model.NewId(), "")
	CheckForbiddenStatus(t, resp)

	Client.Logout()
	_, resp = Client.GetTeamsUnreadForUser(user.Id, "")
	CheckUnauthorizedStatus(t, resp)
}

func TestTeamExists(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam

	th.LoginBasic()

	exists, resp := Client.TeamExists(team.Name, "")
	CheckNoError(t, resp)
	if !exists {
		t.Fatal("team should exist")
	}

	exists, resp = Client.TeamExists("testingteam", "")
	CheckNoError(t, resp)
	if exists {
		t.Fatal("team should not exist")
	}

	Client.Logout()
	_, resp = Client.TeamExists(team.Name, "")
	CheckUnauthorizedStatus(t, resp)
}

func TestImportTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()

	t.Run("ImportTeam", func(t *testing.T) {
		var data []byte
		var err error
		data, err = readTestFile("Fake_Team_Import.zip")
		if err != nil && len(data) == 0 {
			t.Fatal("Error while reading the test file.")
		}

		// Import the channels/users/posts
		fileResp, resp := th.SystemAdminClient.ImportTeam(data, binary.Size(data), "slack", "Fake_Team_Import.zip", th.BasicTeam.Id)
		CheckNoError(t, resp)

		fileData, err := base64.StdEncoding.DecodeString(fileResp["results"])
		if err != nil {
			t.Fatal("failed to decode base64 results data")
		}

		fileReturned := fmt.Sprintf("%s", fileData)
		if !strings.Contains(fileReturned, "darth.vader@stardeath.com") {
			t.Log(fileReturned)
			t.Fatal("failed to report the user was imported")
		}

		// Checking the imported users
		importedUser, resp := th.SystemAdminClient.GetUserByUsername("bot_test", "")
		CheckNoError(t, resp)
		if importedUser.Username != "bot_test" {
			t.Fatal("username should match with the imported user")
		}

		importedUser, resp = th.SystemAdminClient.GetUserByUsername("lordvader", "")
		CheckNoError(t, resp)
		if importedUser.Username != "lordvader" {
			t.Fatal("username should match with the imported user")
		}

		// Checking the imported Channels
		importedChannel, resp := th.SystemAdminClient.GetChannelByName("testchannel", th.BasicTeam.Id, "")
		CheckNoError(t, resp)
		if importedChannel.Name != "testchannel" {
			t.Fatal("names did not match expected: testchannel")
		}

		importedChannel, resp = th.SystemAdminClient.GetChannelByName("general", th.BasicTeam.Id, "")
		CheckNoError(t, resp)
		if importedChannel.Name != "general" {
			t.Fatal("names did not match expected: general")
		}

		posts, resp := th.SystemAdminClient.GetPostsForChannel(importedChannel.Id, 0, 60, "")
		CheckNoError(t, resp)
		if posts.Posts[posts.Order[3]].Message != "This is a test post to test the import process" {
			t.Fatal("missing posts in the import process")
		}
	})

	t.Run("MissingFile", func(t *testing.T) {
		_, resp := th.SystemAdminClient.ImportTeam(nil, 4343, "slack", "Fake_Team_Import.zip", th.BasicTeam.Id)
		CheckBadRequestStatus(t, resp)
	})

	t.Run("WrongPermission", func(t *testing.T) {
		var data []byte
		var err error
		data, err = readTestFile("Fake_Team_Import.zip")
		if err != nil && len(data) == 0 {
			t.Fatal("Error while reading the test file.")
		}

		// Import the channels/users/posts
		_, resp := th.Client.ImportTeam(data, binary.Size(data), "slack", "Fake_Team_Import.zip", th.BasicTeam.Id)
		CheckForbiddenStatus(t, resp)
	})
}

func TestInviteUsersToTeam(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()

	user1 := GenerateTestEmail()
	user2 := GenerateTestEmail()

	emailList := []string{user1, user2}

	//Delete all the messages before check the sample email
	utils.DeleteMailBox(user1)
	utils.DeleteMailBox(user2)

	okMsg, resp := th.SystemAdminClient.InviteUsersToTeam(th.BasicTeam.Id, emailList)
	CheckNoError(t, resp)
	if !okMsg {
		t.Fatal("should return true")
	}

	nameFormat := *th.App.Config().TeamSettings.TeammateNameDisplay
	expectedSubject := utils.T("api.templates.invite_subject",
		map[string]interface{}{"SenderName": th.SystemAdminUser.GetDisplayName(nameFormat),
			"TeamDisplayName": th.BasicTeam.DisplayName,
			"SiteName":        utils.ClientCfg["SiteName"]})

	//Check if the email was send to the rigth email address
	for _, email := range emailList {
		var resultsMailbox utils.JSONMessageHeaderInbucket
		err := utils.RetryInbucket(5, func() error {
			var err error
			resultsMailbox, err = utils.GetMailBox(email)
			return err
		})
		if err != nil {
			t.Log(err)
			t.Log("No email was received, maybe due load on the server. Disabling this verification")
		}
		if err == nil && len(resultsMailbox) > 0 {
			if !strings.ContainsAny(resultsMailbox[len(resultsMailbox)-1].To[0], email) {
				t.Fatal("Wrong To recipient")
			} else {
				if resultsEmail, err := utils.GetMessageFromMailbox(email, resultsMailbox[len(resultsMailbox)-1].ID); err == nil {
					if resultsEmail.Subject != expectedSubject {
						t.Log(resultsEmail.Subject)
						t.Log(expectedSubject)
						t.Fatal("Wrong Subject")
					}
				}
			}
		}
	}

	restrictCreationToDomains := th.App.Config().TeamSettings.RestrictCreationToDomains
	defer func() {
		th.App.UpdateConfig(func(cfg *model.Config) { cfg.TeamSettings.RestrictCreationToDomains = restrictCreationToDomains })
	}()
	th.App.UpdateConfig(func(cfg *model.Config) { cfg.TeamSettings.RestrictCreationToDomains = "@example.com" })

	err := th.App.InviteNewUsersToTeam(emailList, th.BasicTeam.Id, th.BasicUser.Id)

	if err == nil {
		t.Fatal("Adding users with non-restricted domains was allowed")
	}
	if err.Where != "InviteNewUsersToTeam" || err.Message != "api.team.invite_members.invalid_email.app_error" {
		t.Log(err)
		t.Fatal("Got wrong error message!")
	}
}

func TestGetTeamInviteInfo(t *testing.T) {
	th := Setup().InitBasic().InitSystemAdmin()
	defer th.TearDown()
	Client := th.Client
	team := th.BasicTeam

	team, resp := Client.GetTeamInviteInfo(team.InviteId)
	CheckNoError(t, resp)

	if team.DisplayName == "" {
		t.Fatal("should not be empty")
	}

	if team.Email != "" {
		t.Fatal("should be empty")
	}

	team.InviteId = "12345678901234567890123456789012"
	team, resp = th.SystemAdminClient.UpdateTeam(team)
	CheckNoError(t, resp)

	team, resp = Client.GetTeamInviteInfo(team.InviteId)
	CheckNoError(t, resp)

	_, resp = Client.GetTeamInviteInfo("junk")
	CheckNotFoundStatus(t, resp)
}
