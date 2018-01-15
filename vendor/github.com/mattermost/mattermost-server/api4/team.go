// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"strconv"

	"github.com/mattermost/mattermost-server/model"
)

const (
	MAX_ADD_MEMBERS_BATCH = 20
)

func (api *API) InitTeam() {
	api.BaseRoutes.Teams.Handle("", api.ApiSessionRequired(createTeam)).Methods("POST")
	api.BaseRoutes.Teams.Handle("", api.ApiSessionRequired(getAllTeams)).Methods("GET")
	api.BaseRoutes.Teams.Handle("/search", api.ApiSessionRequired(searchTeams)).Methods("POST")
	api.BaseRoutes.TeamsForUser.Handle("", api.ApiSessionRequired(getTeamsForUser)).Methods("GET")
	api.BaseRoutes.TeamsForUser.Handle("/unread", api.ApiSessionRequired(getTeamsUnreadForUser)).Methods("GET")

	api.BaseRoutes.Team.Handle("", api.ApiSessionRequired(getTeam)).Methods("GET")
	api.BaseRoutes.Team.Handle("", api.ApiSessionRequired(updateTeam)).Methods("PUT")
	api.BaseRoutes.Team.Handle("", api.ApiSessionRequired(deleteTeam)).Methods("DELETE")
	api.BaseRoutes.Team.Handle("/patch", api.ApiSessionRequired(patchTeam)).Methods("PUT")
	api.BaseRoutes.Team.Handle("/stats", api.ApiSessionRequired(getTeamStats)).Methods("GET")
	api.BaseRoutes.TeamMembers.Handle("", api.ApiSessionRequired(getTeamMembers)).Methods("GET")
	api.BaseRoutes.TeamMembers.Handle("/ids", api.ApiSessionRequired(getTeamMembersByIds)).Methods("POST")
	api.BaseRoutes.TeamMembersForUser.Handle("", api.ApiSessionRequired(getTeamMembersForUser)).Methods("GET")
	api.BaseRoutes.TeamMembers.Handle("", api.ApiSessionRequired(addTeamMember)).Methods("POST")
	api.BaseRoutes.Teams.Handle("/members/invite", api.ApiSessionRequired(addUserToTeamFromInvite)).Methods("POST")
	api.BaseRoutes.TeamMembers.Handle("/batch", api.ApiSessionRequired(addTeamMembers)).Methods("POST")
	api.BaseRoutes.TeamMember.Handle("", api.ApiSessionRequired(removeTeamMember)).Methods("DELETE")

	api.BaseRoutes.TeamForUser.Handle("/unread", api.ApiSessionRequired(getTeamUnread)).Methods("GET")

	api.BaseRoutes.TeamByName.Handle("", api.ApiSessionRequired(getTeamByName)).Methods("GET")
	api.BaseRoutes.TeamMember.Handle("", api.ApiSessionRequired(getTeamMember)).Methods("GET")
	api.BaseRoutes.TeamByName.Handle("/exists", api.ApiSessionRequired(teamExists)).Methods("GET")
	api.BaseRoutes.TeamMember.Handle("/roles", api.ApiSessionRequired(updateTeamMemberRoles)).Methods("PUT")

	api.BaseRoutes.Team.Handle("/import", api.ApiSessionRequired(importTeam)).Methods("POST")
	api.BaseRoutes.Team.Handle("/invite/email", api.ApiSessionRequired(inviteUsersToTeam)).Methods("POST")
	api.BaseRoutes.Teams.Handle("/invite/{invite_id:[A-Za-z0-9]+}", api.ApiHandler(getInviteInfo)).Methods("GET")
}

func createTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	team := model.TeamFromJson(r.Body)
	if team == nil {
		c.SetInvalidParam("team")
		return
	}

	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_CREATE_TEAM) {
		c.Err = model.NewAppError("createTeam", "api.team.is_team_creation_allowed.disabled.app_error", nil, "", http.StatusForbidden)
		return
	}

	rteam, err := c.App.CreateTeamWithUser(team, c.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}

	// Don't sanitize the team here since the user will be a team admin and their session won't reflect that yet

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(rteam.ToJson()))
}

func getTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if team, err := c.App.GetTeam(c.Params.TeamId); err != nil {
		c.Err = err
		return
	} else {
		if (!team.AllowOpenInvite || team.Type != model.TEAM_OPEN) && !c.App.SessionHasPermissionToTeam(c.Session, team.Id, model.PERMISSION_VIEW_TEAM) {
			c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
			return
		}

		c.App.SanitizeTeam(c.Session, team)

		w.Write([]byte(team.ToJson()))
		return
	}
}

func getTeamByName(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamName()
	if c.Err != nil {
		return
	}

	if team, err := c.App.GetTeamByName(c.Params.TeamName); err != nil {
		c.Err = err
		return
	} else {
		if (!team.AllowOpenInvite || team.Type != model.TEAM_OPEN) && !c.App.SessionHasPermissionToTeam(c.Session, team.Id, model.PERMISSION_VIEW_TEAM) {
			c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
			return
		}

		c.App.SanitizeTeam(c.Session, team)

		w.Write([]byte(team.ToJson()))
		return
	}
}

func updateTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	team := model.TeamFromJson(r.Body)

	if team == nil {
		c.SetInvalidParam("team")
		return
	}

	team.Id = c.Params.TeamId

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_MANAGE_TEAM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_TEAM)
		return
	}

	updatedTeam, err := c.App.UpdateTeam(team)

	if err != nil {
		c.Err = err
		return
	}

	c.App.SanitizeTeam(c.Session, updatedTeam)

	w.Write([]byte(updatedTeam.ToJson()))
}

func patchTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	team := model.TeamPatchFromJson(r.Body)

	if team == nil {
		c.SetInvalidParam("team")
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_MANAGE_TEAM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_TEAM)
		return
	}

	patchedTeam, err := c.App.PatchTeam(c.Params.TeamId, team)

	if err != nil {
		c.Err = err
		return
	}

	c.App.SanitizeTeam(c.Session, patchedTeam)

	c.LogAudit("")
	w.Write([]byte(patchedTeam.ToJson()))
}

func deleteTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_MANAGE_TEAM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_TEAM)
		return
	}

	var err *model.AppError
	if c.Params.Permanent {
		err = c.App.PermanentDeleteTeamId(c.Params.TeamId)
	} else {
		err = c.App.SoftDeleteTeam(c.Params.TeamId)
	}

	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func getTeamsForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if c.Session.UserId != c.Params.UserId && !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	if teams, err := c.App.GetTeamsForUser(c.Params.UserId); err != nil {
		c.Err = err
		return
	} else {
		c.App.SanitizeTeams(c.Session, teams)

		w.Write([]byte(model.TeamListToJson(teams)))
	}
}

func getTeamsUnreadForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if c.Session.UserId != c.Params.UserId && !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	// optional team id to be excluded from the result
	teamId := r.URL.Query().Get("exclude_team")

	unreadTeamsList, err := c.App.GetTeamsUnreadForUser(teamId, c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.TeamsUnreadToJson(unreadTeamsList)))
}

func getTeamMember(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequireUserId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	if team, err := c.App.GetTeamMember(c.Params.TeamId, c.Params.UserId); err != nil {
		c.Err = err
		return
	} else {
		w.Write([]byte(team.ToJson()))
		return
	}
}

func getTeamMembers(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	if members, err := c.App.GetTeamMembers(c.Params.TeamId, c.Params.Page, c.Params.PerPage); err != nil {
		c.Err = err
		return
	} else {
		w.Write([]byte(model.TeamMembersToJson(members)))
		return
	}
}

func getTeamMembersForUser(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireUserId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToUser(c.Session, c.Params.UserId) {
		c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
		return
	}

	members, err := c.App.GetTeamMembersForUser(c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.TeamMembersToJson(members)))
}

func getTeamMembersByIds(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	userIds := model.ArrayFromJson(r.Body)

	if len(userIds) == 0 {
		c.SetInvalidParam("user_ids")
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	members, err := c.App.GetTeamMembersByIds(c.Params.TeamId, userIds)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.TeamMembersToJson(members)))
}

func addTeamMember(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	var err *model.AppError
	member := model.TeamMemberFromJson(r.Body)
	if member.TeamId != c.Params.TeamId {
		c.SetInvalidParam("team_id")
		return
	}

	if len(member.UserId) != 26 {
		c.SetInvalidParam("user_id")
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, member.TeamId, model.PERMISSION_ADD_USER_TO_TEAM) {
		c.SetPermissionError(model.PERMISSION_ADD_USER_TO_TEAM)
		return
	}

	member, err = c.App.AddTeamMember(member.TeamId, member.UserId)

	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(member.ToJson()))
}

func addUserToTeamFromInvite(c *Context, w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	data := r.URL.Query().Get("data")
	inviteId := r.URL.Query().Get("invite_id")

	var member *model.TeamMember
	var err *model.AppError

	if len(hash) > 0 && len(data) > 0 {
		member, err = c.App.AddTeamMemberByHash(c.Session.UserId, hash, data)
	} else if len(inviteId) > 0 {
		member, err = c.App.AddTeamMemberByInviteId(inviteId, c.Session.UserId)
	} else {
		err = model.NewAppError("addTeamMember", "api.team.add_user_to_team.missing_parameter.app_error", nil, "", http.StatusBadRequest)
	}

	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(member.ToJson()))
}

func addTeamMembers(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	var err *model.AppError
	members := model.TeamMembersFromJson(r.Body)

	if len(members) > MAX_ADD_MEMBERS_BATCH || len(members) == 0 {
		c.SetInvalidParam("too many members in batch")
		return
	}

	var userIds []string
	for _, member := range members {
		if member.TeamId != c.Params.TeamId {
			c.SetInvalidParam("team_id for member with user_id=" + member.UserId)
			return
		}

		if len(member.UserId) != 26 {
			c.SetInvalidParam("user_id")
			return
		}

		userIds = append(userIds, member.UserId)
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_ADD_USER_TO_TEAM) {
		c.SetPermissionError(model.PERMISSION_ADD_USER_TO_TEAM)
		return
	}

	members, err = c.App.AddTeamMembers(c.Params.TeamId, userIds, c.Session.UserId)

	if err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(model.TeamMembersToJson(members)))
}

func removeTeamMember(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequireUserId()
	if c.Err != nil {
		return
	}

	if c.Session.UserId != c.Params.UserId {
		if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_REMOVE_USER_FROM_TEAM) {
			c.SetPermissionError(model.PERMISSION_REMOVE_USER_FROM_TEAM)
			return
		}
	}

	if err := c.App.RemoveUserFromTeam(c.Params.TeamId, c.Params.UserId, c.Session.UserId); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func getTeamUnread(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequireUserId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToUser(c.Session, c.Params.UserId) {
		c.SetPermissionError(model.PERMISSION_EDIT_OTHER_USERS)
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	unreadTeam, err := c.App.GetTeamUnread(c.Params.TeamId, c.Params.UserId)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(unreadTeam.ToJson()))
}

func getTeamStats(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_VIEW_TEAM) {
		c.SetPermissionError(model.PERMISSION_VIEW_TEAM)
		return
	}

	if stats, err := c.App.GetTeamStats(c.Params.TeamId); err != nil {
		c.Err = err
		return
	} else {
		w.Write([]byte(stats.ToJson()))
		return
	}
}

func updateTeamMemberRoles(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId().RequireUserId()
	if c.Err != nil {
		return
	}

	props := model.MapFromJson(r.Body)

	newRoles := props["roles"]
	if !model.IsValidUserRoles(newRoles) {
		c.SetInvalidParam("team_member_roles")
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_MANAGE_TEAM_ROLES) {
		c.SetPermissionError(model.PERMISSION_MANAGE_TEAM_ROLES)
		return
	}

	if _, err := c.App.UpdateTeamMemberRoles(c.Params.TeamId, c.Params.UserId, newRoles); err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func getAllTeams(c *Context, w http.ResponseWriter, r *http.Request) {
	var teams []*model.Team
	var err *model.AppError

	if c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		teams, err = c.App.GetAllTeamsPage(c.Params.Page, c.Params.PerPage)
	} else {
		teams, err = c.App.GetAllOpenTeamsPage(c.Params.Page, c.Params.PerPage)
	}

	if err != nil {
		c.Err = err
		return
	}

	c.App.SanitizeTeams(c.Session, teams)

	w.Write([]byte(model.TeamListToJson(teams)))
}

func searchTeams(c *Context, w http.ResponseWriter, r *http.Request) {
	props := model.TeamSearchFromJson(r.Body)
	if props == nil {
		c.SetInvalidParam("team_search")
		return
	}

	if len(props.Term) == 0 {
		c.SetInvalidParam("term")
		return
	}

	var teams []*model.Team
	var err *model.AppError

	if c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		teams, err = c.App.SearchAllTeams(props.Term)
	} else {
		teams, err = c.App.SearchOpenTeams(props.Term)
	}

	if err != nil {
		c.Err = err
		return
	}

	c.App.SanitizeTeams(c.Session, teams)

	w.Write([]byte(model.TeamListToJson(teams)))
}

func teamExists(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamName()
	if c.Err != nil {
		return
	}

	resp := make(map[string]bool)

	if _, err := c.App.GetTeamByName(c.Params.TeamName); err != nil {
		resp["exists"] = false
	} else {
		resp["exists"] = true
	}

	w.Write([]byte(model.MapBoolToJson(resp)))
}

func importTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_IMPORT_TEAM) {
		c.SetPermissionError(model.PERMISSION_IMPORT_TEAM)
		return
	}

	if err := r.ParseMultipartForm(10000000); err != nil {
		c.Err = model.NewAppError("importTeam", "api.team.import_team.parse.app_error", nil, err.Error(), http.StatusInternalServerError)
		return
	}

	importFromArray, ok := r.MultipartForm.Value["importFrom"]
	if !ok || len(importFromArray) < 1 {
		c.Err = model.NewAppError("importTeam", "api.team.import_team.no_import_from.app_error", nil, "", http.StatusBadRequest)
		return
	}
	importFrom := importFromArray[0]

	fileSizeStr, ok := r.MultipartForm.Value["filesize"]
	if !ok || len(fileSizeStr) < 1 {
		c.Err = model.NewAppError("importTeam", "api.team.import_team.unavailable.app_error", nil, "", http.StatusBadRequest)
		return
	}

	fileSize, err := strconv.ParseInt(fileSizeStr[0], 10, 64)
	if err != nil {
		c.Err = model.NewAppError("importTeam", "api.team.import_team.integer.app_error", nil, "", http.StatusBadRequest)
		return
	}

	fileInfoArray, ok := r.MultipartForm.File["file"]
	if !ok {
		c.Err = model.NewAppError("importTeam", "api.team.import_team.no_file.app_error", nil, "", http.StatusBadRequest)
		return
	}

	if len(fileInfoArray) <= 0 {
		c.Err = model.NewAppError("importTeam", "api.team.import_team.array.app_error", nil, "", http.StatusBadRequest)
		return
	}

	fileInfo := fileInfoArray[0]

	fileData, err := fileInfo.Open()
	if err != nil {
		c.Err = model.NewAppError("importTeam", "api.team.import_team.open.app_error", nil, err.Error(), http.StatusBadRequest)
		return
	}
	defer fileData.Close()

	var log *bytes.Buffer
	switch importFrom {
	case "slack":
		var err *model.AppError
		if err, log = c.App.SlackImport(fileData, fileSize, c.Params.TeamId); err != nil {
			c.Err = err
			c.Err.StatusCode = http.StatusBadRequest
		}
	}

	data := map[string]string{}
	data["results"] = base64.StdEncoding.EncodeToString([]byte(log.Bytes()))
	if c.Err != nil {
		w.WriteHeader(c.Err.StatusCode)
	}
	w.Write([]byte(model.MapToJson(data)))
}

func inviteUsersToTeam(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireTeamId()
	if c.Err != nil {
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_INVITE_USER) {
		c.SetPermissionError(model.PERMISSION_INVITE_USER)
		return
	}

	if !c.App.SessionHasPermissionToTeam(c.Session, c.Params.TeamId, model.PERMISSION_ADD_USER_TO_TEAM) {
		c.SetPermissionError(model.PERMISSION_INVITE_USER)
		return
	}

	emailList := model.ArrayFromJson(r.Body)

	if len(emailList) == 0 {
		c.SetInvalidParam("user_email")
		return
	}

	err := c.App.InviteNewUsersToTeam(emailList, c.Params.TeamId, c.Session.UserId)
	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func getInviteInfo(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireInviteId()
	if c.Err != nil {
		return
	}

	if team, err := c.App.GetTeamByInviteId(c.Params.InviteId); err != nil {
		c.Err = err
		return
	} else {
		if !(team.Type == model.TEAM_OPEN) {
			c.Err = model.NewAppError("getInviteInfo", "api.team.get_invite_info.not_open_team", nil, "id="+c.Params.InviteId, http.StatusForbidden)
			return
		}

		result := map[string]string{}
		result["display_name"] = team.DisplayName
		result["description"] = team.Description
		result["name"] = team.Name
		result["id"] = team.Id
		w.Write([]byte(model.MapToJson(result)))
	}
}
