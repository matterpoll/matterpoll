// Copyright (c) 2016-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"bytes"
	b64 "encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	l4g "github.com/alecthomas/log4go"
	"github.com/mattermost/mattermost-server/einterfaces"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/store"
	"github.com/mattermost/mattermost-server/utils"
)

const (
	OAUTH_COOKIE_MAX_AGE_SECONDS = 30 * 60 // 30 minutes
	COOKIE_OAUTH                 = "MMOAUTH"
)

func (a *App) CreateOAuthApp(app *model.OAuthApp) (*model.OAuthApp, *model.AppError) {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return nil, model.NewAppError("CreateOAuthApp", "api.oauth.register_oauth_app.turn_off.app_error", nil, "", http.StatusNotImplemented)
	}

	secret := model.NewId()
	app.ClientSecret = secret

	if result := <-a.Srv.Store.OAuth().SaveApp(app); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.OAuthApp), nil
	}
}

func (a *App) GetOAuthApp(appId string) (*model.OAuthApp, *model.AppError) {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return nil, model.NewAppError("GetOAuthApp", "api.oauth.allow_oauth.turn_off.app_error", nil, "", http.StatusNotImplemented)
	}

	if result := <-a.Srv.Store.OAuth().GetApp(appId); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.OAuthApp), nil
	}
}

func (a *App) UpdateOauthApp(oldApp, updatedApp *model.OAuthApp) (*model.OAuthApp, *model.AppError) {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return nil, model.NewAppError("UpdateOauthApp", "api.oauth.allow_oauth.turn_off.app_error", nil, "", http.StatusNotImplemented)
	}

	updatedApp.Id = oldApp.Id
	updatedApp.CreatorId = oldApp.CreatorId
	updatedApp.CreateAt = oldApp.CreateAt
	updatedApp.ClientSecret = oldApp.ClientSecret

	if result := <-a.Srv.Store.OAuth().UpdateApp(updatedApp); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.([2]*model.OAuthApp)[0], nil
	}
}

func (a *App) DeleteOAuthApp(appId string) *model.AppError {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return model.NewAppError("DeleteOAuthApp", "api.oauth.allow_oauth.turn_off.app_error", nil, "", http.StatusNotImplemented)
	}

	if err := (<-a.Srv.Store.OAuth().DeleteApp(appId)).Err; err != nil {
		return err
	}

	a.InvalidateAllCaches()

	return nil
}

func (a *App) GetOAuthApps(page, perPage int) ([]*model.OAuthApp, *model.AppError) {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return nil, model.NewAppError("GetOAuthApps", "api.oauth.allow_oauth.turn_off.app_error", nil, "", http.StatusNotImplemented)
	}

	if result := <-a.Srv.Store.OAuth().GetApps(page*perPage, perPage); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.([]*model.OAuthApp), nil
	}
}

func (a *App) GetOAuthAppsByCreator(userId string, page, perPage int) ([]*model.OAuthApp, *model.AppError) {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return nil, model.NewAppError("GetOAuthAppsByUser", "api.oauth.allow_oauth.turn_off.app_error", nil, "", http.StatusNotImplemented)
	}

	if result := <-a.Srv.Store.OAuth().GetAppByUser(userId, page*perPage, perPage); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.([]*model.OAuthApp), nil
	}
}

func (a *App) AllowOAuthAppAccessToUser(userId string, authRequest *model.AuthorizeRequest) (string, *model.AppError) {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return "", model.NewAppError("AllowOAuthAppAccessToUser", "api.oauth.allow_oauth.turn_off.app_error", nil, "", http.StatusNotImplemented)
	}

	if len(authRequest.Scope) == 0 {
		authRequest.Scope = model.DEFAULT_SCOPE
	}

	var oauthApp *model.OAuthApp
	if result := <-a.Srv.Store.OAuth().GetApp(authRequest.ClientId); result.Err != nil {
		return "", result.Err
	} else {
		oauthApp = result.Data.(*model.OAuthApp)
	}

	if !oauthApp.IsValidRedirectURL(authRequest.RedirectUri) {
		return "", model.NewAppError("AllowOAuthAppAccessToUser", "api.oauth.allow_oauth.redirect_callback.app_error", nil, "", http.StatusBadRequest)
	}

	if authRequest.ResponseType != model.AUTHCODE_RESPONSE_TYPE {
		return authRequest.RedirectUri + "?error=unsupported_response_type&state=" + authRequest.State, nil
	}

	authData := &model.AuthData{UserId: userId, ClientId: authRequest.ClientId, CreateAt: model.GetMillis(), RedirectUri: authRequest.RedirectUri, State: authRequest.State, Scope: authRequest.Scope}
	authData.Code = model.NewId() + model.NewId()

	// this saves the OAuth2 app as authorized
	authorizedApp := model.Preference{
		UserId:   userId,
		Category: model.PREFERENCE_CATEGORY_AUTHORIZED_OAUTH_APP,
		Name:     authRequest.ClientId,
		Value:    authRequest.Scope,
	}

	if result := <-a.Srv.Store.Preference().Save(&model.Preferences{authorizedApp}); result.Err != nil {
		return authRequest.RedirectUri + "?error=server_error&state=" + authRequest.State, nil
	}

	if result := <-a.Srv.Store.OAuth().SaveAuthData(authData); result.Err != nil {
		return authRequest.RedirectUri + "?error=server_error&state=" + authRequest.State, nil
	}

	return authRequest.RedirectUri + "?code=" + url.QueryEscape(authData.Code) + "&state=" + url.QueryEscape(authData.State), nil
}

func (a *App) GetOAuthAccessToken(clientId, grantType, redirectUri, code, secret, refreshToken string) (*model.AccessResponse, *model.AppError) {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.disabled.app_error", nil, "", http.StatusNotImplemented)
	}

	var oauthApp *model.OAuthApp
	if result := <-a.Srv.Store.OAuth().GetApp(clientId); result.Err != nil {
		return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.credentials.app_error", nil, "", http.StatusNotFound)
	} else {
		oauthApp = result.Data.(*model.OAuthApp)
	}

	if oauthApp.ClientSecret != secret {
		return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.credentials.app_error", nil, "", http.StatusForbidden)
	}

	var user *model.User
	var accessData *model.AccessData
	var accessRsp *model.AccessResponse
	if grantType == model.ACCESS_TOKEN_GRANT_TYPE {

		var authData *model.AuthData
		if result := <-a.Srv.Store.OAuth().GetAuthData(code); result.Err != nil {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.expired_code.app_error", nil, "", http.StatusInternalServerError)
		} else {
			authData = result.Data.(*model.AuthData)
		}

		if authData.IsExpired() {
			<-a.Srv.Store.OAuth().RemoveAuthData(authData.Code)
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.expired_code.app_error", nil, "", http.StatusForbidden)
		}

		if authData.RedirectUri != redirectUri {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.redirect_uri.app_error", nil, "", http.StatusBadRequest)
		}

		if result := <-a.Srv.Store.User().Get(authData.UserId); result.Err != nil {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.internal_user.app_error", nil, "", http.StatusNotFound)
		} else {
			user = result.Data.(*model.User)
		}

		if result := <-a.Srv.Store.OAuth().GetPreviousAccessData(user.Id, clientId); result.Err != nil {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.internal.app_error", nil, "", http.StatusInternalServerError)
		} else if result.Data != nil {
			accessData := result.Data.(*model.AccessData)
			if accessData.IsExpired() {
				if access, err := a.newSessionUpdateToken(oauthApp.Name, accessData, user); err != nil {
					return nil, err
				} else {
					accessRsp = access
				}
			} else {
				//return the same token and no need to create a new session
				accessRsp = &model.AccessResponse{
					AccessToken:  accessData.Token,
					TokenType:    model.ACCESS_TOKEN_TYPE,
					RefreshToken: accessData.RefreshToken,
					ExpiresIn:    int32((accessData.ExpiresAt - model.GetMillis()) / 1000),
				}
			}
		} else {
			// create a new session and return new access token
			var session *model.Session
			if result, err := a.newSession(oauthApp.Name, user); err != nil {
				return nil, err
			} else {
				session = result
			}

			accessData = &model.AccessData{ClientId: clientId, UserId: user.Id, Token: session.Token, RefreshToken: model.NewId(), RedirectUri: redirectUri, ExpiresAt: session.ExpiresAt, Scope: authData.Scope}

			if result := <-a.Srv.Store.OAuth().SaveAccessData(accessData); result.Err != nil {
				l4g.Error(result.Err)
				return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.internal_saving.app_error", nil, "", http.StatusInternalServerError)
			}

			accessRsp = &model.AccessResponse{
				AccessToken:  session.Token,
				TokenType:    model.ACCESS_TOKEN_TYPE,
				RefreshToken: accessData.RefreshToken,
				ExpiresIn:    int32(*a.Config().ServiceSettings.SessionLengthSSOInDays * 60 * 60 * 24),
			}
		}

		<-a.Srv.Store.OAuth().RemoveAuthData(authData.Code)
	} else {
		// when grantType is refresh_token
		if result := <-a.Srv.Store.OAuth().GetAccessDataByRefreshToken(refreshToken); result.Err != nil {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.refresh_token.app_error", nil, "", http.StatusNotFound)
		} else {
			accessData = result.Data.(*model.AccessData)
		}

		if result := <-a.Srv.Store.User().Get(accessData.UserId); result.Err != nil {
			return nil, model.NewAppError("GetOAuthAccessToken", "api.oauth.get_access_token.internal_user.app_error", nil, "", http.StatusNotFound)
		} else {
			user = result.Data.(*model.User)
		}

		if access, err := a.newSessionUpdateToken(oauthApp.Name, accessData, user); err != nil {
			return nil, err
		} else {
			accessRsp = access
		}
	}

	return accessRsp, nil
}

func (a *App) newSession(appName string, user *model.User) (*model.Session, *model.AppError) {
	// set new token an session
	session := &model.Session{UserId: user.Id, Roles: user.Roles, IsOAuth: true}
	session.SetExpireInDays(*a.Config().ServiceSettings.SessionLengthSSOInDays)
	session.AddProp(model.SESSION_PROP_PLATFORM, appName)
	session.AddProp(model.SESSION_PROP_OS, "OAuth2")
	session.AddProp(model.SESSION_PROP_BROWSER, "OAuth2")

	if result := <-a.Srv.Store.Session().Save(session); result.Err != nil {
		return nil, model.NewAppError("newSession", "api.oauth.get_access_token.internal_session.app_error", nil, "", http.StatusInternalServerError)
	} else {
		session = result.Data.(*model.Session)
		a.AddSessionToCache(session)
	}

	return session, nil
}

func (a *App) newSessionUpdateToken(appName string, accessData *model.AccessData, user *model.User) (*model.AccessResponse, *model.AppError) {
	var session *model.Session
	<-a.Srv.Store.Session().Remove(accessData.Token) //remove the previous session

	if result, err := a.newSession(appName, user); err != nil {
		return nil, err
	} else {
		session = result
	}

	accessData.Token = session.Token
	accessData.RefreshToken = model.NewId()
	accessData.ExpiresAt = session.ExpiresAt
	if result := <-a.Srv.Store.OAuth().UpdateAccessData(accessData); result.Err != nil {
		l4g.Error(result.Err)
		return nil, model.NewAppError("newSessionUpdateToken", "web.get_access_token.internal_saving.app_error", nil, "", http.StatusInternalServerError)
	}
	accessRsp := &model.AccessResponse{
		AccessToken:  session.Token,
		RefreshToken: accessData.RefreshToken,
		TokenType:    model.ACCESS_TOKEN_TYPE,
		ExpiresIn:    int32(*a.Config().ServiceSettings.SessionLengthSSOInDays * 60 * 60 * 24),
	}

	return accessRsp, nil
}

func (a *App) GetOAuthLoginEndpoint(w http.ResponseWriter, r *http.Request, service, teamId, action, redirectTo, loginHint string) (string, *model.AppError) {
	stateProps := map[string]string{}
	stateProps["action"] = action
	if len(teamId) != 0 {
		stateProps["team_id"] = teamId
	}

	if len(redirectTo) != 0 {
		stateProps["redirect_to"] = redirectTo
	}

	if authUrl, err := a.GetAuthorizationCode(w, r, service, stateProps, loginHint); err != nil {
		return "", err
	} else {
		return authUrl, nil
	}
}

func (a *App) GetOAuthSignupEndpoint(w http.ResponseWriter, r *http.Request, service, teamId string) (string, *model.AppError) {
	stateProps := map[string]string{}
	stateProps["action"] = model.OAUTH_ACTION_SIGNUP
	if len(teamId) != 0 {
		stateProps["team_id"] = teamId
	}

	if authUrl, err := a.GetAuthorizationCode(w, r, service, stateProps, ""); err != nil {
		return "", err
	} else {
		return authUrl, nil
	}
}

func (a *App) GetAuthorizedAppsForUser(userId string, page, perPage int) ([]*model.OAuthApp, *model.AppError) {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return nil, model.NewAppError("GetAuthorizedAppsForUser", "api.oauth.allow_oauth.turn_off.app_error", nil, "", http.StatusNotImplemented)
	}

	if result := <-a.Srv.Store.OAuth().GetAuthorizedApps(userId, page*perPage, perPage); result.Err != nil {
		return nil, result.Err
	} else {
		apps := result.Data.([]*model.OAuthApp)
		for k, a := range apps {
			a.Sanitize()
			apps[k] = a
		}

		return apps, nil
	}
}

func (a *App) DeauthorizeOAuthAppForUser(userId, appId string) *model.AppError {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return model.NewAppError("DeauthorizeOAuthAppForUser", "api.oauth.allow_oauth.turn_off.app_error", nil, "", http.StatusNotImplemented)
	}

	// revoke app sessions
	if result := <-a.Srv.Store.OAuth().GetAccessDataByUserForApp(userId, appId); result.Err != nil {
		return result.Err
	} else {
		accessData := result.Data.([]*model.AccessData)

		for _, ad := range accessData {
			if err := a.RevokeAccessToken(ad.Token); err != nil {
				return err
			}

			if rad := <-a.Srv.Store.OAuth().RemoveAccessData(ad.Token); rad.Err != nil {
				return rad.Err
			}
		}
	}

	// Deauthorize the app
	if err := (<-a.Srv.Store.Preference().Delete(userId, model.PREFERENCE_CATEGORY_AUTHORIZED_OAUTH_APP, appId)).Err; err != nil {
		return err
	}

	return nil
}

func (a *App) RegenerateOAuthAppSecret(app *model.OAuthApp) (*model.OAuthApp, *model.AppError) {
	if !a.Config().ServiceSettings.EnableOAuthServiceProvider {
		return nil, model.NewAppError("RegenerateOAuthAppSecret", "api.oauth.allow_oauth.turn_off.app_error", nil, "", http.StatusNotImplemented)
	}

	app.ClientSecret = model.NewId()
	if update := <-a.Srv.Store.OAuth().UpdateApp(app); update.Err != nil {
		return nil, update.Err
	}

	return app, nil
}

func (a *App) RevokeAccessToken(token string) *model.AppError {
	session, _ := a.GetSession(token)
	schan := a.Srv.Store.Session().Remove(token)

	if result := <-a.Srv.Store.OAuth().GetAccessData(token); result.Err != nil {
		return model.NewAppError("RevokeAccessToken", "api.oauth.revoke_access_token.get.app_error", nil, "", http.StatusBadRequest)
	}

	tchan := a.Srv.Store.OAuth().RemoveAccessData(token)

	if result := <-tchan; result.Err != nil {
		return model.NewAppError("RevokeAccessToken", "api.oauth.revoke_access_token.del_token.app_error", nil, "", http.StatusInternalServerError)
	}

	if result := <-schan; result.Err != nil {
		return model.NewAppError("RevokeAccessToken", "api.oauth.revoke_access_token.del_session.app_error", nil, "", http.StatusInternalServerError)
	}

	if session != nil {
		a.ClearSessionCacheForUser(session.UserId)
	}

	return nil
}

func (a *App) CompleteOAuth(service string, body io.ReadCloser, teamId string, props map[string]string) (*model.User, *model.AppError) {
	defer body.Close()

	action := props["action"]

	switch action {
	case model.OAUTH_ACTION_SIGNUP:
		return a.CreateOAuthUser(service, body, teamId)
	case model.OAUTH_ACTION_LOGIN:
		return a.LoginByOAuth(service, body, teamId)
	case model.OAUTH_ACTION_EMAIL_TO_SSO:
		return a.CompleteSwitchWithOAuth(service, body, props["email"])
	case model.OAUTH_ACTION_SSO_TO_EMAIL:
		return a.LoginByOAuth(service, body, teamId)
	default:
		return a.LoginByOAuth(service, body, teamId)
	}
}

func (a *App) LoginByOAuth(service string, userData io.Reader, teamId string) (*model.User, *model.AppError) {
	buf := bytes.Buffer{}
	buf.ReadFrom(userData)

	authData := ""
	provider := einterfaces.GetOauthProvider(service)
	if provider == nil {
		return nil, model.NewAppError("LoginByOAuth", "api.user.login_by_oauth.not_available.app_error",
			map[string]interface{}{"Service": strings.Title(service)}, "", http.StatusNotImplemented)
	} else {
		authData = provider.GetAuthDataFromJson(bytes.NewReader(buf.Bytes()))
	}

	if len(authData) == 0 {
		return nil, model.NewAppError("LoginByOAuth", "api.user.login_by_oauth.parse.app_error",
			map[string]interface{}{"Service": service}, "", http.StatusBadRequest)
	}

	user, err := a.GetUserByAuth(&authData, service)
	if err != nil {
		if err.Id == store.MISSING_AUTH_ACCOUNT_ERROR {
			return a.CreateOAuthUser(service, bytes.NewReader(buf.Bytes()), teamId)
		}
		return nil, err
	}

	if err = a.UpdateOAuthUserAttrs(bytes.NewReader(buf.Bytes()), user, provider, service); err != nil {
		return nil, err
	}

	if len(teamId) > 0 {
		err = a.AddUserToTeamByTeamId(teamId, user)
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (a *App) CompleteSwitchWithOAuth(service string, userData io.ReadCloser, email string) (*model.User, *model.AppError) {
	authData := ""
	ssoEmail := ""
	provider := einterfaces.GetOauthProvider(service)
	if provider == nil {
		return nil, model.NewAppError("CompleteSwitchWithOAuth", "api.user.complete_switch_with_oauth.unavailable.app_error",
			map[string]interface{}{"Service": strings.Title(service)}, "", http.StatusNotImplemented)
	} else {
		ssoUser := provider.GetUserFromJson(userData)
		ssoEmail = ssoUser.Email

		if ssoUser.AuthData != nil {
			authData = *ssoUser.AuthData
		}
	}

	if len(authData) == 0 {
		return nil, model.NewAppError("CompleteSwitchWithOAuth", "api.user.complete_switch_with_oauth.parse.app_error",
			map[string]interface{}{"Service": service}, "", http.StatusBadRequest)
	}

	if len(email) == 0 {
		return nil, model.NewAppError("CompleteSwitchWithOAuth", "api.user.complete_switch_with_oauth.blank_email.app_error", nil, "", http.StatusBadRequest)
	}

	var user *model.User
	if result := <-a.Srv.Store.User().GetByEmail(email); result.Err != nil {
		return nil, result.Err
	} else {
		user = result.Data.(*model.User)
	}

	if err := a.RevokeAllSessions(user.Id); err != nil {
		return nil, err
	}

	if result := <-a.Srv.Store.User().UpdateAuthData(user.Id, service, &authData, ssoEmail, true); result.Err != nil {
		return nil, result.Err
	}

	a.Go(func() {
		if err := a.SendSignInChangeEmail(user.Email, strings.Title(service)+" SSO", user.Locale, utils.GetSiteURL()); err != nil {
			l4g.Error(err.Error())
		}
	})

	return user, nil
}

func (a *App) CreateOAuthStateToken(extra string) (*model.Token, *model.AppError) {
	token := model.NewToken(model.TOKEN_TYPE_OAUTH, extra)

	if result := <-a.Srv.Store.Token().Save(token); result.Err != nil {
		return nil, result.Err
	}

	return token, nil
}

func (a *App) GetOAuthStateToken(token string) (*model.Token, *model.AppError) {
	if result := <-a.Srv.Store.Token().GetByToken(token); result.Err != nil {
		return nil, model.NewAppError("GetOAuthStateToken", "api.oauth.invalid_state_token.app_error", nil, result.Err.Error(), http.StatusBadRequest)
	} else {
		token := result.Data.(*model.Token)
		if token.Type != model.TOKEN_TYPE_OAUTH {
			return nil, model.NewAppError("GetOAuthStateToken", "api.oauth.invalid_state_token.app_error", nil, "", http.StatusBadRequest)
		}

		return token, nil
	}
}

func generateOAuthStateTokenExtra(email, action, cookie string) string {
	return email + ":" + action + ":" + cookie
}

func (a *App) GetAuthorizationCode(w http.ResponseWriter, r *http.Request, service string, props map[string]string, loginHint string) (string, *model.AppError) {
	sso := a.Config().GetSSOService(service)
	if sso != nil && !sso.Enable {
		return "", model.NewAppError("GetAuthorizationCode", "api.user.get_authorization_code.unsupported.app_error", nil, "service="+service, http.StatusNotImplemented)
	}

	secure := false
	if GetProtocol(r) == "https" {
		secure = true
	}

	cookieValue := model.NewId()
	expiresAt := time.Unix(model.GetMillis()/1000+int64(OAUTH_COOKIE_MAX_AGE_SECONDS), 0)
	oauthCookie := &http.Cookie{
		Name:     COOKIE_OAUTH,
		Value:    cookieValue,
		Path:     "/",
		MaxAge:   OAUTH_COOKIE_MAX_AGE_SECONDS,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   secure,
	}

	http.SetCookie(w, oauthCookie)

	clientId := sso.Id
	endpoint := sso.AuthEndpoint
	scope := sso.Scope

	tokenExtra := generateOAuthStateTokenExtra(props["email"], props["action"], cookieValue)
	stateToken, err := a.CreateOAuthStateToken(tokenExtra)
	if err != nil {
		return "", err
	}

	props["token"] = stateToken.Token
	state := b64.StdEncoding.EncodeToString([]byte(model.MapToJson(props)))

	redirectUri := utils.GetSiteURL() + "/signup/" + service + "/complete"

	authUrl := endpoint + "?response_type=code&client_id=" + clientId + "&redirect_uri=" + url.QueryEscape(redirectUri) + "&state=" + url.QueryEscape(state)

	if len(scope) > 0 {
		authUrl += "&scope=" + utils.UrlEncode(scope)
	}

	if len(loginHint) > 0 {
		authUrl += "&login_hint=" + utils.UrlEncode(loginHint)
	}

	return authUrl, nil
}

func (a *App) AuthorizeOAuthUser(w http.ResponseWriter, r *http.Request, service, code, state, redirectUri string) (io.ReadCloser, string, map[string]string, *model.AppError) {
	sso := a.Config().GetSSOService(service)
	if sso == nil || !sso.Enable {
		return nil, "", nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.unsupported.app_error", nil, "service="+service, http.StatusNotImplemented)
	}

	stateStr := ""
	if b, err := b64.StdEncoding.DecodeString(state); err != nil {
		return nil, "", nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.invalid_state.app_error", nil, err.Error(), http.StatusBadRequest)
	} else {
		stateStr = string(b)
	}

	stateProps := model.MapFromJson(strings.NewReader(stateStr))

	expectedToken, err := a.GetOAuthStateToken(stateProps["token"])
	if err != nil {
		return nil, "", stateProps, err
	}

	stateEmail := stateProps["email"]
	stateAction := stateProps["action"]
	if stateAction == model.OAUTH_ACTION_EMAIL_TO_SSO && stateEmail == "" {
		return nil, "", stateProps, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.invalid_state.app_error", nil, "", http.StatusBadRequest)
	}

	cookieValue := ""
	if cookie, err := r.Cookie(COOKIE_OAUTH); err != nil {
		return nil, "", stateProps, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.invalid_state.app_error", nil, "", http.StatusBadRequest)
	} else {
		cookieValue = cookie.Value
	}

	expectedTokenExtra := generateOAuthStateTokenExtra(stateEmail, stateAction, cookieValue)
	if expectedTokenExtra != expectedToken.Extra {
		return nil, "", stateProps, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.invalid_state.app_error", nil, "", http.StatusBadRequest)
	}

	a.DeleteToken(expectedToken)

	cookie := &http.Cookie{
		Name:     COOKIE_OAUTH,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

	teamId := stateProps["team_id"]

	p := url.Values{}
	p.Set("client_id", sso.Id)
	p.Set("client_secret", sso.Secret)
	p.Set("code", code)
	p.Set("grant_type", model.ACCESS_TOKEN_GRANT_TYPE)
	p.Set("redirect_uri", redirectUri)

	req, _ := http.NewRequest("POST", sso.TokenEndpoint, strings.NewReader(p.Encode()))

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	var ar *model.AccessResponse
	var bodyBytes []byte
	if resp, err := a.HTTPClient(true).Do(req); err != nil {
		return nil, "", stateProps, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.token_failed.app_error", nil, err.Error(), http.StatusInternalServerError)
	} else {
		ar = model.AccessResponseFromJson(resp.Body)
		consumeAndClose(resp)

		if ar == nil {
			return nil, "", stateProps, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.bad_response.app_error", nil, "response_body="+string(bodyBytes), http.StatusInternalServerError)
		}
	}

	if strings.ToLower(ar.TokenType) != model.ACCESS_TOKEN_TYPE {
		return nil, "", stateProps, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.bad_token.app_error", nil, "token_type="+ar.TokenType+", response_body="+string(bodyBytes), http.StatusInternalServerError)
	}

	if len(ar.AccessToken) == 0 {
		return nil, "", stateProps, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.missing.app_error", nil, "response_body="+string(bodyBytes), http.StatusInternalServerError)
	}

	p = url.Values{}
	p.Set("access_token", ar.AccessToken)
	req, _ = http.NewRequest("GET", sso.UserApiEndpoint, strings.NewReader(""))

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+ar.AccessToken)

	if resp, err := a.HTTPClient(true).Do(req); err != nil {
		return nil, "", stateProps, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.service.app_error", map[string]interface{}{"Service": service}, err.Error(), http.StatusInternalServerError)
	} else {
		return resp.Body, teamId, stateProps, nil
	}

}

func (a *App) SwitchEmailToOAuth(w http.ResponseWriter, r *http.Request, email, password, code, service string) (string, *model.AppError) {
	if utils.IsLicensed() && !*a.Config().ServiceSettings.ExperimentalEnableAuthenticationTransfer {
		return "", model.NewAppError("emailToOAuth", "api.user.email_to_oauth.not_available.app_error", nil, "", http.StatusForbidden)
	}

	var user *model.User
	var err *model.AppError
	if user, err = a.GetUserByEmail(email); err != nil {
		return "", err
	}

	if err := a.CheckPasswordAndAllCriteria(user, password, code); err != nil {
		return "", err
	}

	stateProps := map[string]string{}
	stateProps["action"] = model.OAUTH_ACTION_EMAIL_TO_SSO
	stateProps["email"] = email

	if service == model.USER_AUTH_SERVICE_SAML {
		return utils.GetSiteURL() + "/login/sso/saml?action=" + model.OAUTH_ACTION_EMAIL_TO_SSO + "&email=" + utils.UrlEncode(email), nil
	} else {
		if authUrl, err := a.GetAuthorizationCode(w, r, service, stateProps, ""); err != nil {
			return "", err
		} else {
			return authUrl, nil
		}
	}
}

func (a *App) SwitchOAuthToEmail(email, password, requesterId string) (string, *model.AppError) {
	if utils.IsLicensed() && !*a.Config().ServiceSettings.ExperimentalEnableAuthenticationTransfer {
		return "", model.NewAppError("oauthToEmail", "api.user.oauth_to_email.not_available.app_error", nil, "", http.StatusForbidden)
	}

	var user *model.User
	var err *model.AppError
	if user, err = a.GetUserByEmail(email); err != nil {
		return "", err
	}

	if user.Id != requesterId {
		return "", model.NewAppError("SwitchOAuthToEmail", "api.user.oauth_to_email.context.app_error", nil, "", http.StatusForbidden)
	}

	if err := a.UpdatePassword(user, password); err != nil {
		return "", err
	}

	T := utils.GetUserTranslations(user.Locale)

	a.Go(func() {
		if err := a.SendSignInChangeEmail(user.Email, T("api.templates.signin_change_email.body.method_email"), user.Locale, utils.GetSiteURL()); err != nil {
			l4g.Error(err.Error())
		}
	})

	if err := a.RevokeAllSessions(requesterId); err != nil {
		return "", err
	}

	return "/login?extra=signin_change", nil
}
