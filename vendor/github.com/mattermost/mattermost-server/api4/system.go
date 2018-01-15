// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"bytes"
	"io"
	"net/http"
	"runtime"
	"strconv"

	l4g "github.com/alecthomas/log4go"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

func (api *API) InitSystem() {
	api.BaseRoutes.System.Handle("/ping", api.ApiHandler(getSystemPing)).Methods("GET")

	api.BaseRoutes.ApiRoot.Handle("/config", api.ApiSessionRequired(getConfig)).Methods("GET")
	api.BaseRoutes.ApiRoot.Handle("/config", api.ApiSessionRequired(updateConfig)).Methods("PUT")
	api.BaseRoutes.ApiRoot.Handle("/config/reload", api.ApiSessionRequired(configReload)).Methods("POST")
	api.BaseRoutes.ApiRoot.Handle("/config/client", api.ApiHandler(getClientConfig)).Methods("GET")

	api.BaseRoutes.ApiRoot.Handle("/license", api.ApiSessionRequired(addLicense)).Methods("POST")
	api.BaseRoutes.ApiRoot.Handle("/license", api.ApiSessionRequired(removeLicense)).Methods("DELETE")
	api.BaseRoutes.ApiRoot.Handle("/license/client", api.ApiHandler(getClientLicense)).Methods("GET")

	api.BaseRoutes.ApiRoot.Handle("/audits", api.ApiSessionRequired(getAudits)).Methods("GET")
	api.BaseRoutes.ApiRoot.Handle("/email/test", api.ApiSessionRequired(testEmail)).Methods("POST")
	api.BaseRoutes.ApiRoot.Handle("/database/recycle", api.ApiSessionRequired(databaseRecycle)).Methods("POST")
	api.BaseRoutes.ApiRoot.Handle("/caches/invalidate", api.ApiSessionRequired(invalidateCaches)).Methods("POST")

	api.BaseRoutes.ApiRoot.Handle("/logs", api.ApiSessionRequired(getLogs)).Methods("GET")
	api.BaseRoutes.ApiRoot.Handle("/logs", api.ApiHandler(postLog)).Methods("POST")

	api.BaseRoutes.ApiRoot.Handle("/analytics/old", api.ApiSessionRequired(getAnalytics)).Methods("GET")
}

func getSystemPing(c *Context, w http.ResponseWriter, r *http.Request) {

	actualGoroutines := runtime.NumGoroutine()
	if *c.App.Config().ServiceSettings.GoroutineHealthThreshold <= 0 || actualGoroutines <= *c.App.Config().ServiceSettings.GoroutineHealthThreshold {
		m := make(map[string]string)
		m[model.STATUS] = model.STATUS_OK

		reqs := c.App.Config().ClientRequirements
		m["AndroidLatestVersion"] = reqs.AndroidLatestVersion
		m["AndroidMinVersion"] = reqs.AndroidMinVersion
		m["DesktopLatestVersion"] = reqs.DesktopLatestVersion
		m["DesktopMinVersion"] = reqs.DesktopMinVersion
		m["IosLatestVersion"] = reqs.IosLatestVersion
		m["IosMinVersion"] = reqs.IosMinVersion

		w.Write([]byte(model.MapToJson(m)))
	} else {
		rdata := map[string]string{}
		rdata["status"] = "unhealthy"

		l4g.Warn(utils.T("api.system.go_routines"), actualGoroutines, *c.App.Config().ServiceSettings.GoroutineHealthThreshold)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(model.MapToJson(rdata)))
	}
}

func testEmail(c *Context, w http.ResponseWriter, r *http.Request) {
	cfg := model.ConfigFromJson(r.Body)
	if cfg == nil {
		cfg = c.App.Config()
	}

	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	err := c.App.TestEmail(c.Session.UserId, cfg)
	if err != nil {
		c.Err = err
		return
	}

	ReturnStatusOK(w)
}

func getConfig(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	cfg := c.App.GetConfig()

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write([]byte(cfg.ToJson()))
}

func configReload(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	c.App.ReloadConfig()

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	ReturnStatusOK(w)
}

func updateConfig(c *Context, w http.ResponseWriter, r *http.Request) {
	cfg := model.ConfigFromJson(r.Body)
	if cfg == nil {
		c.SetInvalidParam("config")
		return
	}

	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	err := c.App.SaveConfig(cfg, true)
	if err != nil {
		c.Err = err
		return
	}

	c.LogAudit("updateConfig")

	cfg = c.App.GetConfig()

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write([]byte(cfg.ToJson()))
}

func getAudits(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	audits, err := c.App.GetAuditsPage("", c.Params.Page, c.Params.PerPage)

	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(audits.ToJson()))
}

func databaseRecycle(c *Context, w http.ResponseWriter, r *http.Request) {

	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	c.App.RecycleDatabaseConnection()

	ReturnStatusOK(w)
}

func invalidateCaches(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	err := c.App.InvalidateAllCaches()
	if err != nil {
		c.Err = err
		return
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	ReturnStatusOK(w)
}

func getLogs(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	lines, err := c.App.GetLogs(c.Params.Page, c.Params.LogsPerPage)
	if err != nil {
		c.Err = err
		return
	}

	w.Write([]byte(model.ArrayToJson(lines)))
}

func postLog(c *Context, w http.ResponseWriter, r *http.Request) {
	forceToDebug := false

	if !*c.App.Config().ServiceSettings.EnableDeveloper {
		if c.Session.UserId == "" {
			c.Err = model.NewAppError("postLog", "api.context.permissions.app_error", nil, "", http.StatusForbidden)
			return
		}

		if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
			forceToDebug = true
		}
	}

	m := model.MapFromJson(r.Body)
	lvl := m["level"]
	msg := m["message"]

	if len(msg) > 400 {
		msg = msg[0:399]
	}

	if !forceToDebug && lvl == "ERROR" {
		err := &model.AppError{}
		err.Message = msg
		err.Id = msg
		err.Where = "client"
		c.LogError(err)
	} else {
		l4g.Debug(msg)
	}

	m["message"] = msg
	w.Write([]byte(model.MapToJson(m)))
}

func getClientConfig(c *Context, w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")

	if format == "" {
		c.Err = model.NewAppError("getClientConfig", "api.config.client.old_format.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	if format != "old" {
		c.SetInvalidParam("format")
		return
	}

	respCfg := map[string]string{}
	for k, v := range utils.ClientCfg {
		respCfg[k] = v
	}

	respCfg["NoAccounts"] = strconv.FormatBool(c.App.IsFirstUserAccount())

	w.Write([]byte(model.MapToJson(respCfg)))
}

func getClientLicense(c *Context, w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")

	if format == "" {
		c.Err = model.NewAppError("getClientLicense", "api.license.client.old_format.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	if format != "old" {
		c.SetInvalidParam("format")
		return
	}

	etag := utils.GetClientLicenseEtag(true)
	if c.HandleEtag(etag, "Get Client License", w, r) {
		return
	}

	var clientLicense map[string]string

	if c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		clientLicense = utils.ClientLicense()
	} else {
		clientLicense = utils.GetSanitizedClientLicense()
	}

	w.Header().Set(model.HEADER_ETAG_SERVER, etag)
	w.Write([]byte(model.MapToJson(clientLicense)))
}

func addLicense(c *Context, w http.ResponseWriter, r *http.Request) {
	c.LogAudit("attempt")

	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	err := r.ParseMultipartForm(*c.App.Config().FileSettings.MaxFileSize)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := r.MultipartForm

	fileArray, ok := m.File["license"]
	if !ok {
		c.Err = model.NewAppError("addLicense", "api.license.add_license.no_file.app_error", nil, "", http.StatusBadRequest)
		return
	}

	if len(fileArray) <= 0 {
		c.Err = model.NewAppError("addLicense", "api.license.add_license.array.app_error", nil, "", http.StatusBadRequest)
		return
	}

	fileData := fileArray[0]

	file, err := fileData.Open()
	if err != nil {
		c.Err = model.NewAppError("addLicense", "api.license.add_license.open.app_error", nil, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, file)

	if license, err := c.App.SaveLicense(buf.Bytes()); err != nil {
		if err.Id == model.EXPIRED_LICENSE_ERROR {
			c.LogAudit("failed - expired or non-started license")
		} else if err.Id == model.INVALID_LICENSE_ERROR {
			c.LogAudit("failed - invalid license")
		} else {
			c.LogAudit("failed - unable to save license")
		}
		c.Err = err
		return
	} else {
		c.LogAudit("success")
		w.Write([]byte(license.ToJson()))
	}
}

func removeLicense(c *Context, w http.ResponseWriter, r *http.Request) {
	c.LogAudit("attempt")

	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	if err := c.App.RemoveLicense(); err != nil {
		c.Err = err
		return
	}

	c.LogAudit("success")
	ReturnStatusOK(w)
}

func getAnalytics(c *Context, w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	teamId := r.URL.Query().Get("team_id")

	if name == "" {
		name = "standard"
	}

	if !c.App.SessionHasPermissionTo(c.Session, model.PERMISSION_MANAGE_SYSTEM) {
		c.SetPermissionError(model.PERMISSION_MANAGE_SYSTEM)
		return
	}

	rows, err := c.App.GetAnalytics(name, teamId)
	if err != nil {
		c.Err = err
		return
	}

	if rows == nil {
		c.SetInvalidParam("name")
		return
	}

	w.Write([]byte(rows.ToJson()))
}
