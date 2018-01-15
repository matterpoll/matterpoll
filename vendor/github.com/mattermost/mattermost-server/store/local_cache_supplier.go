// Copyright (c) 2016-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package store

import (
	"context"

	"github.com/mattermost/mattermost-server/einterfaces"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

const (
	REACTION_CACHE_SIZE = 20000
	REACTION_CACHE_SEC  = 1800 // 30 minutes

	CLEAR_CACHE_MESSAGE_DATA = ""
)

type LocalCacheSupplier struct {
	next          LayeredStoreSupplier
	reactionCache *utils.Cache
	metrics       einterfaces.MetricsInterface
	cluster       einterfaces.ClusterInterface
}

func NewLocalCacheSupplier(metrics einterfaces.MetricsInterface, cluster einterfaces.ClusterInterface) *LocalCacheSupplier {
	supplier := &LocalCacheSupplier{
		reactionCache: utils.NewLruWithParams(REACTION_CACHE_SIZE, "Reaction", REACTION_CACHE_SEC, model.CLUSTER_EVENT_INVALIDATE_CACHE_FOR_REACTIONS),
		metrics:       metrics,
		cluster:       cluster,
	}

	if cluster != nil {
		cluster.RegisterClusterMessageHandler(model.CLUSTER_EVENT_INVALIDATE_CACHE_FOR_REACTIONS, supplier.handleClusterInvalidateReaction)
	}

	return supplier
}

func (s *LocalCacheSupplier) SetChainNext(next LayeredStoreSupplier) {
	s.next = next
}

func (s *LocalCacheSupplier) Next() LayeredStoreSupplier {
	return s.next
}

func (s *LocalCacheSupplier) doStandardReadCache(ctx context.Context, cache utils.ObjectCache, key string, hints ...LayeredStoreHint) *LayeredStoreSupplierResult {
	if hintsContains(hints, LSH_NO_CACHE) {
		if s.metrics != nil {
			s.metrics.IncrementMemCacheMissCounter(cache.Name())
		}
		return nil
	}

	if cacheItem, ok := cache.Get(key); ok {
		if s.metrics != nil {
			s.metrics.IncrementMemCacheHitCounter(cache.Name())
		}
		result := NewSupplierResult()
		result.Data = cacheItem
		return result
	}

	if s.metrics != nil {
		s.metrics.IncrementMemCacheMissCounter(cache.Name())
	}

	return nil
}

func (s *LocalCacheSupplier) doStandardAddToCache(ctx context.Context, cache utils.ObjectCache, key string, result *LayeredStoreSupplierResult, hints ...LayeredStoreHint) {
	if result.Err == nil && result.Data != nil {
		cache.AddWithDefaultExpires(key, result.Data)
	}
}

func (s *LocalCacheSupplier) doInvalidateCacheCluster(cache utils.ObjectCache, key string) {
	cache.Remove(key)
	if s.cluster != nil {
		msg := &model.ClusterMessage{
			Event:    cache.GetInvalidateClusterEvent(),
			SendType: model.CLUSTER_SEND_BEST_EFFORT,
			Data:     key,
		}
		s.cluster.SendClusterMessage(msg)
	}
}

func (s *LocalCacheSupplier) doClearCacheCluster(cache utils.ObjectCache) {
	cache.Purge()
	if s.cluster != nil {
		msg := &model.ClusterMessage{
			Event:    cache.GetInvalidateClusterEvent(),
			SendType: model.CLUSTER_SEND_BEST_EFFORT,
			Data:     CLEAR_CACHE_MESSAGE_DATA,
		}
		s.cluster.SendClusterMessage(msg)
	}
}
