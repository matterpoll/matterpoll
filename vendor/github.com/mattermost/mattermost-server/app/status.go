// Copyright (c) 2016-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	l4g "github.com/alecthomas/log4go"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/store"
	"github.com/mattermost/mattermost-server/utils"
)

var statusCache *utils.Cache = utils.NewLru(model.STATUS_CACHE_SIZE)

func ClearStatusCache() {
	statusCache.Purge()
}

func (a *App) AddStatusCacheSkipClusterSend(status *model.Status) {
	statusCache.Add(status.UserId, status)
}

func (a *App) AddStatusCache(status *model.Status) {
	a.AddStatusCacheSkipClusterSend(status)

	if a.Cluster != nil {
		msg := &model.ClusterMessage{
			Event:    model.CLUSTER_EVENT_UPDATE_STATUS,
			SendType: model.CLUSTER_SEND_BEST_EFFORT,
			Data:     status.ToJson(),
		}
		a.Cluster.SendClusterMessage(msg)
	}
}

func (a *App) GetAllStatuses() map[string]*model.Status {
	if !*a.Config().ServiceSettings.EnableUserStatuses {
		return map[string]*model.Status{}
	}

	userIds := statusCache.Keys()
	statusMap := map[string]*model.Status{}

	for _, userId := range userIds {
		if id, ok := userId.(string); !ok {
			continue
		} else {
			status := GetStatusFromCache(id)
			if status != nil {
				statusMap[id] = status
			}
		}
	}

	return statusMap
}

func (a *App) GetStatusesByIds(userIds []string) (map[string]interface{}, *model.AppError) {
	if !*a.Config().ServiceSettings.EnableUserStatuses {
		return map[string]interface{}{}, nil
	}

	statusMap := map[string]interface{}{}
	metrics := a.Metrics

	missingUserIds := []string{}
	for _, userId := range userIds {
		if result, ok := statusCache.Get(userId); ok {
			statusMap[userId] = result.(*model.Status).Status
			if metrics != nil {
				metrics.IncrementMemCacheHitCounter("Status")
			}
		} else {
			missingUserIds = append(missingUserIds, userId)
			if metrics != nil {
				metrics.IncrementMemCacheMissCounter("Status")
			}
		}
	}

	if len(missingUserIds) > 0 {
		if result := <-a.Srv.Store.Status().GetByIds(missingUserIds); result.Err != nil {
			return nil, result.Err
		} else {
			statuses := result.Data.([]*model.Status)

			for _, s := range statuses {
				a.AddStatusCache(s)
				statusMap[s.UserId] = s.Status
			}
		}
	}

	// For the case where the user does not have a row in the Status table and cache
	for _, userId := range missingUserIds {
		if _, ok := statusMap[userId]; !ok {
			statusMap[userId] = model.STATUS_OFFLINE
		}
	}

	return statusMap, nil
}

//GetUserStatusesByIds used by apiV4
func (a *App) GetUserStatusesByIds(userIds []string) ([]*model.Status, *model.AppError) {
	if !*a.Config().ServiceSettings.EnableUserStatuses {
		return []*model.Status{}, nil
	}

	var statusMap []*model.Status
	metrics := a.Metrics

	missingUserIds := []string{}
	for _, userId := range userIds {
		if result, ok := statusCache.Get(userId); ok {
			statusMap = append(statusMap, result.(*model.Status))
			if metrics != nil {
				metrics.IncrementMemCacheHitCounter("Status")
			}
		} else {
			missingUserIds = append(missingUserIds, userId)
			if metrics != nil {
				metrics.IncrementMemCacheMissCounter("Status")
			}
		}
	}

	if len(missingUserIds) > 0 {
		if result := <-a.Srv.Store.Status().GetByIds(missingUserIds); result.Err != nil {
			return nil, result.Err
		} else {
			statuses := result.Data.([]*model.Status)

			for _, s := range statuses {
				a.AddStatusCache(s)
			}

			statusMap = append(statusMap, statuses...)
		}
	}

	// For the case where the user does not have a row in the Status table and cache
	// remove the existing ids from missingUserIds and then create a offline state for the missing ones
	// This also return the status offline for the non-existing Ids in the system
	for i := 0; i < len(missingUserIds); i++ {
		missingUserId := missingUserIds[i]
		for _, userMap := range statusMap {
			if missingUserId == userMap.UserId {
				missingUserIds = append(missingUserIds[:i], missingUserIds[i+1:]...)
				i--
				break
			}
		}
	}
	for _, userId := range missingUserIds {
		statusMap = append(statusMap, &model.Status{UserId: userId, Status: "offline"})
	}

	return statusMap, nil
}

func (a *App) SetStatusOnline(userId string, sessionId string, manual bool) {
	if !*a.Config().ServiceSettings.EnableUserStatuses {
		return
	}

	broadcast := false

	var oldStatus string = model.STATUS_OFFLINE
	var oldTime int64 = 0
	var oldManual bool = false
	var status *model.Status
	var err *model.AppError

	if status, err = a.GetStatus(userId); err != nil {
		status = &model.Status{UserId: userId, Status: model.STATUS_ONLINE, Manual: false, LastActivityAt: model.GetMillis(), ActiveChannel: ""}
		broadcast = true
	} else {
		if status.Manual && !manual {
			return // manually set status always overrides non-manual one
		}

		if status.Status != model.STATUS_ONLINE {
			broadcast = true
		}

		oldStatus = status.Status
		oldTime = status.LastActivityAt
		oldManual = status.Manual

		status.Status = model.STATUS_ONLINE
		status.Manual = false // for "online" there's no manual setting
		status.LastActivityAt = model.GetMillis()
	}

	a.AddStatusCache(status)

	// Only update the database if the status has changed, the status has been manually set,
	// or enough time has passed since the previous action
	if status.Status != oldStatus || status.Manual != oldManual || status.LastActivityAt-oldTime > model.STATUS_MIN_UPDATE_TIME {

		var schan store.StoreChannel
		if broadcast {
			schan = a.Srv.Store.Status().SaveOrUpdate(status)
		} else {
			schan = a.Srv.Store.Status().UpdateLastActivityAt(status.UserId, status.LastActivityAt)
		}

		if result := <-schan; result.Err != nil {
			l4g.Error(utils.T("api.status.save_status.error"), userId, result.Err)
		}
	}

	if broadcast {
		a.BroadcastStatus(status)
	}
}

func (a *App) BroadcastStatus(status *model.Status) {
	event := model.NewWebSocketEvent(model.WEBSOCKET_EVENT_STATUS_CHANGE, "", "", status.UserId, nil)
	event.Add("status", status.Status)
	event.Add("user_id", status.UserId)
	a.Go(func() {
		a.Publish(event)
	})
}

func (a *App) SetStatusOffline(userId string, manual bool) {
	if !*a.Config().ServiceSettings.EnableUserStatuses {
		return
	}

	status, err := a.GetStatus(userId)
	if err == nil && status.Manual && !manual {
		return // manually set status always overrides non-manual one
	}

	status = &model.Status{UserId: userId, Status: model.STATUS_OFFLINE, Manual: manual, LastActivityAt: model.GetMillis(), ActiveChannel: ""}

	a.AddStatusCache(status)

	if result := <-a.Srv.Store.Status().SaveOrUpdate(status); result.Err != nil {
		l4g.Error(utils.T("api.status.save_status.error"), userId, result.Err)
	}

	event := model.NewWebSocketEvent(model.WEBSOCKET_EVENT_STATUS_CHANGE, "", "", status.UserId, nil)
	event.Add("status", model.STATUS_OFFLINE)
	event.Add("user_id", status.UserId)
	a.Go(func() {
		a.Publish(event)
	})
}

func (a *App) SetStatusAwayIfNeeded(userId string, manual bool) {
	if !*a.Config().ServiceSettings.EnableUserStatuses {
		return
	}

	status, err := a.GetStatus(userId)

	if err != nil {
		status = &model.Status{UserId: userId, Status: model.STATUS_OFFLINE, Manual: manual, LastActivityAt: 0, ActiveChannel: ""}
	}

	if !manual && status.Manual {
		return // manually set status always overrides non-manual one
	}

	if !manual {
		if status.Status == model.STATUS_AWAY {
			return
		}

		if !a.IsUserAway(status.LastActivityAt) {
			return
		}
	}

	status.Status = model.STATUS_AWAY
	status.Manual = manual
	status.ActiveChannel = ""

	a.AddStatusCache(status)

	if result := <-a.Srv.Store.Status().SaveOrUpdate(status); result.Err != nil {
		l4g.Error(utils.T("api.status.save_status.error"), userId, result.Err)
	}

	event := model.NewWebSocketEvent(model.WEBSOCKET_EVENT_STATUS_CHANGE, "", "", status.UserId, nil)
	event.Add("status", model.STATUS_AWAY)
	event.Add("user_id", status.UserId)
	a.Go(func() {
		a.Publish(event)
	})
}

func (a *App) SetStatusDoNotDisturb(userId string) {
	if !*a.Config().ServiceSettings.EnableUserStatuses {
		return
	}

	status, err := a.GetStatus(userId)

	if err != nil {
		status = &model.Status{UserId: userId, Status: model.STATUS_OFFLINE, Manual: false, LastActivityAt: 0, ActiveChannel: ""}
	}

	status.Status = model.STATUS_DND
	status.Manual = true

	a.AddStatusCache(status)

	if result := <-a.Srv.Store.Status().SaveOrUpdate(status); result.Err != nil {
		l4g.Error(utils.T("api.status.save_status.error"), userId, result.Err)
	}

	event := model.NewWebSocketEvent(model.WEBSOCKET_EVENT_STATUS_CHANGE, "", "", status.UserId, nil)
	event.Add("status", model.STATUS_DND)
	event.Add("user_id", status.UserId)
	a.Go(func() {
		a.Publish(event)
	})
}

func GetStatusFromCache(userId string) *model.Status {
	if result, ok := statusCache.Get(userId); ok {
		status := result.(*model.Status)
		statusCopy := &model.Status{}
		*statusCopy = *status
		return statusCopy
	}

	return nil
}

func (a *App) GetStatus(userId string) (*model.Status, *model.AppError) {
	if !*a.Config().ServiceSettings.EnableUserStatuses {
		return &model.Status{}, nil
	}

	status := GetStatusFromCache(userId)
	if status != nil {
		return status, nil
	}

	if result := <-a.Srv.Store.Status().Get(userId); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.Status), nil
	}
}

func (a *App) IsUserAway(lastActivityAt int64) bool {
	return model.GetMillis()-lastActivityAt >= *a.Config().TeamSettings.UserStatusAwayTimeout*1000
}
