// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"fmt"
	"net/url"

	"net/http"

	l4g "github.com/alecthomas/log4go"
	"github.com/nicksnyder/go-i18n/i18n"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

func (a *App) SendChangeUsernameEmail(oldUsername, newUsername, email, locale, siteURL string) *model.AppError {
	T := utils.GetUserTranslations(locale)

	subject := T("api.templates.username_change_subject",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"],
			"TeamDisplayName": a.Config().TeamSettings.SiteName})

	bodyPage := a.NewEmailTemplate("email_change_body", locale)
	bodyPage.Props["SiteURL"] = siteURL
	bodyPage.Props["Title"] = T("api.templates.username_change_body.title")
	bodyPage.Html["Info"] = utils.TranslateAsHtml(T, "api.templates.username_change_body.info",
		map[string]interface{}{"TeamDisplayName": a.Config().TeamSettings.SiteName, "NewUsername": newUsername})

	if err := a.SendMail(email, subject, bodyPage.Render()); err != nil {
		return model.NewAppError("SendChangeUsernameEmail", "api.user.send_email_change_username_and_forget.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendEmailChangeVerifyEmail(newUserEmail, locale, siteURL, token string) *model.AppError {
	T := utils.GetUserTranslations(locale)

	link := fmt.Sprintf("%s/do_verify_email?token=%s&email=%s", siteURL, token, url.QueryEscape(newUserEmail))

	subject := T("api.templates.email_change_verify_subject",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"],
			"TeamDisplayName": a.Config().TeamSettings.SiteName})

	bodyPage := a.NewEmailTemplate("email_change_verify_body", locale)
	bodyPage.Props["SiteURL"] = siteURL
	bodyPage.Props["Title"] = T("api.templates.email_change_verify_body.title")
	bodyPage.Props["Info"] = T("api.templates.email_change_verify_body.info",
		map[string]interface{}{"TeamDisplayName": a.Config().TeamSettings.SiteName})
	bodyPage.Props["VerifyUrl"] = link
	bodyPage.Props["VerifyButton"] = T("api.templates.email_change_verify_body.button")

	if err := a.SendMail(newUserEmail, subject, bodyPage.Render()); err != nil {
		return model.NewAppError("SendEmailChangeVerifyEmail", "api.user.send_email_change_verify_email_and_forget.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendEmailChangeEmail(oldEmail, newEmail, locale, siteURL string) *model.AppError {
	T := utils.GetUserTranslations(locale)

	subject := T("api.templates.email_change_subject",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"],
			"TeamDisplayName": a.Config().TeamSettings.SiteName})

	bodyPage := a.NewEmailTemplate("email_change_body", locale)
	bodyPage.Props["SiteURL"] = siteURL
	bodyPage.Props["Title"] = T("api.templates.email_change_body.title")
	bodyPage.Html["Info"] = utils.TranslateAsHtml(T, "api.templates.email_change_body.info",
		map[string]interface{}{"TeamDisplayName": a.Config().TeamSettings.SiteName, "NewEmail": newEmail})

	if err := a.SendMail(oldEmail, subject, bodyPage.Render()); err != nil {
		return model.NewAppError("SendEmailChangeEmail", "api.user.send_email_change_email_and_forget.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendVerifyEmail(userEmail, locale, siteURL, token string) *model.AppError {
	T := utils.GetUserTranslations(locale)

	link := fmt.Sprintf("%s/do_verify_email?token=%s&email=%s", siteURL, token, url.QueryEscape(userEmail))

	url, _ := url.Parse(siteURL)

	subject := T("api.templates.verify_subject",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"]})

	bodyPage := a.NewEmailTemplate("verify_body", locale)
	bodyPage.Props["SiteURL"] = siteURL
	bodyPage.Props["Title"] = T("api.templates.verify_body.title", map[string]interface{}{"ServerURL": url.Host})
	bodyPage.Props["Info"] = T("api.templates.verify_body.info")
	bodyPage.Props["VerifyUrl"] = link
	bodyPage.Props["Button"] = T("api.templates.verify_body.button")

	if err := a.SendMail(userEmail, subject, bodyPage.Render()); err != nil {
		return model.NewAppError("SendVerifyEmail", "api.user.send_verify_email_and_forget.failed.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendSignInChangeEmail(email, method, locale, siteURL string) *model.AppError {
	T := utils.GetUserTranslations(locale)

	subject := T("api.templates.signin_change_email.subject",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"]})

	bodyPage := a.NewEmailTemplate("signin_change_body", locale)
	bodyPage.Props["SiteURL"] = siteURL
	bodyPage.Props["Title"] = T("api.templates.signin_change_email.body.title")
	bodyPage.Html["Info"] = utils.TranslateAsHtml(T, "api.templates.signin_change_email.body.info",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"], "Method": method})

	if err := a.SendMail(email, subject, bodyPage.Render()); err != nil {
		return model.NewAppError("SendSignInChangeEmail", "api.user.send_sign_in_change_email_and_forget.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendWelcomeEmail(userId string, email string, verified bool, locale, siteURL string) *model.AppError {
	T := utils.GetUserTranslations(locale)

	rawUrl, _ := url.Parse(siteURL)

	subject := T("api.templates.welcome_subject",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"],
			"ServerURL": rawUrl.Host})

	bodyPage := a.NewEmailTemplate("welcome_body", locale)
	bodyPage.Props["SiteURL"] = siteURL
	bodyPage.Props["Title"] = T("api.templates.welcome_body.title", map[string]interface{}{"ServerURL": rawUrl.Host})
	bodyPage.Props["Info"] = T("api.templates.welcome_body.info")
	bodyPage.Props["Button"] = T("api.templates.welcome_body.button")
	bodyPage.Props["Info2"] = T("api.templates.welcome_body.info2")
	bodyPage.Props["Info3"] = T("api.templates.welcome_body.info3")
	bodyPage.Props["SiteURL"] = siteURL

	if *a.Config().NativeAppSettings.AppDownloadLink != "" {
		bodyPage.Props["AppDownloadInfo"] = T("api.templates.welcome_body.app_download_info")
		bodyPage.Props["AppDownloadLink"] = *a.Config().NativeAppSettings.AppDownloadLink
	}

	if !verified {
		token, err := a.CreateVerifyEmailToken(userId)
		if err != nil {
			return err
		}
		link := fmt.Sprintf("%s/do_verify_email?token=%s&email=%s", siteURL, token.Token, url.QueryEscape(email))
		bodyPage.Props["VerifyUrl"] = link
	}

	if err := a.SendMail(email, subject, bodyPage.Render()); err != nil {
		return model.NewAppError("SendWelcomeEmail", "api.user.send_welcome_email_and_forget.failed.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendPasswordChangeEmail(email, method, locale, siteURL string) *model.AppError {
	T := utils.GetUserTranslations(locale)

	subject := T("api.templates.password_change_subject",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"],
			"TeamDisplayName": a.Config().TeamSettings.SiteName})

	bodyPage := a.NewEmailTemplate("password_change_body", locale)
	bodyPage.Props["SiteURL"] = siteURL
	bodyPage.Props["Title"] = T("api.templates.password_change_body.title")
	bodyPage.Html["Info"] = utils.TranslateAsHtml(T, "api.templates.password_change_body.info",
		map[string]interface{}{"TeamDisplayName": a.Config().TeamSettings.SiteName, "TeamURL": siteURL, "Method": method})

	if err := a.SendMail(email, subject, bodyPage.Render()); err != nil {
		return model.NewAppError("SendPasswordChangeEmail", "api.user.send_password_change_email_and_forget.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendUserAccessTokenAddedEmail(email, locale string) *model.AppError {
	T := utils.GetUserTranslations(locale)

	subject := T("api.templates.user_access_token_subject",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"]})

	bodyPage := a.NewEmailTemplate("password_change_body", locale)
	bodyPage.Props["Title"] = T("api.templates.user_access_token_body.title")
	bodyPage.Html["Info"] = utils.TranslateAsHtml(T, "api.templates.user_access_token_body.info",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"], "SiteURL": utils.GetSiteURL()})

	if err := a.SendMail(email, subject, bodyPage.Render()); err != nil {
		return model.NewAppError("SendUserAccessTokenAddedEmail", "api.user.send_user_access_token.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendPasswordResetEmail(email string, token *model.Token, locale, siteURL string) (bool, *model.AppError) {

	T := utils.GetUserTranslations(locale)

	link := fmt.Sprintf("%s/reset_password_complete?token=%s", siteURL, url.QueryEscape(token.Token))

	subject := T("api.templates.reset_subject",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"]})

	bodyPage := a.NewEmailTemplate("reset_body", locale)
	bodyPage.Props["SiteURL"] = siteURL
	bodyPage.Props["Title"] = T("api.templates.reset_body.title")
	bodyPage.Html["Info"] = utils.TranslateAsHtml(T, "api.templates.reset_body.info", nil)
	bodyPage.Props["ResetUrl"] = link
	bodyPage.Props["Button"] = T("api.templates.reset_body.button")

	if err := a.SendMail(email, subject, bodyPage.Render()); err != nil {
		return false, model.NewAppError("SendPasswordReset", "api.user.send_password_reset.send.app_error", nil, "err="+err.Message, http.StatusInternalServerError)
	}

	return true, nil
}

func (a *App) SendMfaChangeEmail(email string, activated bool, locale, siteURL string) *model.AppError {
	T := utils.GetUserTranslations(locale)

	subject := T("api.templates.mfa_change_subject",
		map[string]interface{}{"SiteName": utils.ClientCfg["SiteName"]})

	bodyPage := a.NewEmailTemplate("mfa_change_body", locale)
	bodyPage.Props["SiteURL"] = siteURL

	bodyText := ""
	if activated {
		bodyText = "api.templates.mfa_activated_body.info"
		bodyPage.Props["Title"] = T("api.templates.mfa_activated_body.title")
	} else {
		bodyText = "api.templates.mfa_deactivated_body.info"
		bodyPage.Props["Title"] = T("api.templates.mfa_deactivated_body.title")
	}

	bodyPage.Html["Info"] = utils.TranslateAsHtml(T, bodyText, map[string]interface{}{"SiteURL": siteURL})

	if err := a.SendMail(email, subject, bodyPage.Render()); err != nil {
		return model.NewAppError("SendMfaChangeEmail", "api.user.send_mfa_change_email.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) SendInviteEmails(team *model.Team, senderName string, invites []string, siteURL string) {
	for _, invite := range invites {
		if len(invite) > 0 {
			senderRole := utils.T("api.team.invite_members.member")

			subject := utils.T("api.templates.invite_subject",
				map[string]interface{}{"SenderName": senderName,
					"TeamDisplayName": team.DisplayName,
					"SiteName":        utils.ClientCfg["SiteName"]})

			bodyPage := a.NewEmailTemplate("invite_body", model.DEFAULT_LOCALE)
			bodyPage.Props["SiteURL"] = siteURL
			bodyPage.Props["Title"] = utils.T("api.templates.invite_body.title")
			bodyPage.Html["Info"] = utils.TranslateAsHtml(utils.T, "api.templates.invite_body.info",
				map[string]interface{}{"SenderStatus": senderRole, "SenderName": senderName, "TeamDisplayName": team.DisplayName})
			bodyPage.Props["Info"] = map[string]interface{}{}
			bodyPage.Props["Button"] = utils.T("api.templates.invite_body.button")
			bodyPage.Html["ExtraInfo"] = utils.TranslateAsHtml(utils.T, "api.templates.invite_body.extra_info",
				map[string]interface{}{"TeamDisplayName": team.DisplayName, "TeamURL": siteURL + "/" + team.Name})

			props := make(map[string]string)
			props["email"] = invite
			props["id"] = team.Id
			props["display_name"] = team.DisplayName
			props["name"] = team.Name
			props["time"] = fmt.Sprintf("%v", model.GetMillis())
			data := model.MapToJson(props)
			hash := utils.HashSha256(fmt.Sprintf("%v:%v", data, a.Config().EmailSettings.InviteSalt))
			bodyPage.Props["Link"] = fmt.Sprintf("%s/signup_user_complete/?d=%s&h=%s", siteURL, url.QueryEscape(data), url.QueryEscape(hash))

			if !a.Config().EmailSettings.SendEmailNotifications {
				l4g.Info(utils.T("api.team.invite_members.sending.info"), invite, bodyPage.Props["Link"])
			}

			if err := a.SendMail(invite, subject, bodyPage.Render()); err != nil {
				l4g.Error(utils.T("api.team.invite_members.send.error"), err)
			}
		}
	}
}

func (a *App) NewEmailTemplate(name, locale string) *utils.HTMLTemplate {
	t := utils.NewHTMLTemplate(a.HTMLTemplates(), name)

	var localT i18n.TranslateFunc
	if locale != "" {
		localT = utils.GetUserTranslations(locale)
	} else {
		localT = utils.T
	}

	t.Props["Footer"] = localT("api.templates.email_footer")

	if *a.Config().EmailSettings.FeedbackOrganization != "" {
		t.Props["Organization"] = localT("api.templates.email_organization") + *a.Config().EmailSettings.FeedbackOrganization
	} else {
		t.Props["Organization"] = ""
	}

	t.Html["EmailInfo"] = utils.TranslateAsHtml(localT, "api.templates.email_info",
		map[string]interface{}{"SupportEmail": *a.Config().SupportSettings.SupportEmail, "SiteName": a.Config().TeamSettings.SiteName})

	return t
}

func (a *App) SendMail(to, subject, htmlBody string) *model.AppError {
	return utils.SendMailUsingConfig(to, subject, htmlBody, a.Config())
}
