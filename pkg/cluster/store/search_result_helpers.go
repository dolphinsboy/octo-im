package store

import (
	"sort"
	"strings"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
)

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}

func userSortTime(user wkdb.User) int64 {
	if user.CreatedAt != nil {
		return user.CreatedAt.UnixNano()
	}
	return 0
}

func limitUsers(users []wkdb.User, limit int, pre bool) []wkdb.User {
	if len(users) == 0 {
		return nil
	}
	if limit > 0 && len(users) > limit {
		if pre {
			users = users[len(users)-limit:]
		} else {
			users = users[:limit]
		}
	}
	out := make([]wkdb.User, 0, len(users))
	for _, user := range users {
		user.CreatedAt = cloneTimePtr(user.CreatedAt)
		user.UpdatedAt = cloneTimePtr(user.UpdatedAt)
		out = append(out, user)
	}
	return out
}

func deviceSortTime(device wkdb.Device) int64 {
	if device.CreatedAt != nil {
		return device.CreatedAt.UnixNano()
	}
	return 0
}

func sortDevices(devices []wkdb.Device) {
	sort.SliceStable(devices, func(i, j int) bool {
		left := deviceSortTime(devices[i])
		right := deviceSortTime(devices[j])
		if left == right {
			if devices[i].Uid == devices[j].Uid {
				return devices[i].DeviceFlag < devices[j].DeviceFlag
			}
			return strings.Compare(devices[i].Uid, devices[j].Uid) < 0
		}
		return left > right
	})
}

func limitDevices(devices []wkdb.Device, limit int, pre bool) []wkdb.Device {
	if len(devices) == 0 {
		return nil
	}
	if limit > 0 && len(devices) > limit {
		if pre {
			devices = devices[len(devices)-limit:]
		} else {
			devices = devices[:limit]
		}
	}
	out := make([]wkdb.Device, 0, len(devices))
	for _, device := range devices {
		device.CreatedAt = cloneTimePtr(device.CreatedAt)
		device.UpdatedAt = cloneTimePtr(device.UpdatedAt)
		out = append(out, device)
	}
	return out
}

func channelSortTime(info wkdb.ChannelInfo) int64 {
	if info.CreatedAt != nil {
		return info.CreatedAt.UnixNano()
	}
	return 0
}

func limitChannelInfos(channels []wkdb.ChannelInfo, limit int, pre bool) []wkdb.ChannelInfo {
	if len(channels) == 0 {
		return nil
	}
	if limit > 0 && len(channels) > limit {
		if pre {
			channels = channels[len(channels)-limit:]
		} else {
			channels = channels[:limit]
		}
	}
	out := make([]wkdb.ChannelInfo, 0, len(channels))
	for _, info := range channels {
		info.CreatedAt = cloneTimePtr(info.CreatedAt)
		info.UpdatedAt = cloneTimePtr(info.UpdatedAt)
		out = append(out, info)
	}
	return out
}

func conversationSortTime(conversation wkdb.Conversation) int64 {
	if conversation.UpdatedAt != nil {
		return conversation.UpdatedAt.UnixNano()
	}
	if conversation.CreatedAt != nil {
		return conversation.CreatedAt.UnixNano()
	}
	return 0
}

func sortConversations(conversations []wkdb.Conversation) {
	sort.SliceStable(conversations, func(i, j int) bool {
		left := conversationSortTime(conversations[i])
		right := conversationSortTime(conversations[j])
		if left == right {
			leftKey := conversations[i].Uid + ":" + wkdb.ChannelToKey(conversations[i].ChannelId, conversations[i].ChannelType)
			rightKey := conversations[j].Uid + ":" + wkdb.ChannelToKey(conversations[j].ChannelId, conversations[j].ChannelType)
			return strings.Compare(leftKey, rightKey) < 0
		}
		return left > right
	})
}

func paginateConversations(conversations []wkdb.Conversation, currentPage, limit int) []wkdb.Conversation {
	if len(conversations) == 0 {
		return nil
	}
	if limit <= 0 {
		out := make([]wkdb.Conversation, 0, len(conversations))
		for _, conversation := range conversations {
			out = append(out, cloneConversation(conversation))
		}
		return out
	}
	if currentPage <= 0 {
		currentPage = 1
	}
	start := (currentPage - 1) * limit
	if start >= len(conversations) {
		return nil
	}
	end := start + limit
	if end > len(conversations) {
		end = len(conversations)
	}
	out := make([]wkdb.Conversation, 0, end-start)
	for _, conversation := range conversations[start:end] {
		out = append(out, cloneConversation(conversation))
	}
	return out
}

func cloneConversation(conversation wkdb.Conversation) wkdb.Conversation {
	conversation.CreatedAt = cloneTimePtr(conversation.CreatedAt)
	conversation.UpdatedAt = cloneTimePtr(conversation.UpdatedAt)
	return conversation
}

func channelClusterConfigSortTime(cfg wkdb.ChannelClusterConfig) int64 {
	if cfg.CreatedAt != nil {
		return cfg.CreatedAt.UnixNano()
	}
	return 0
}

func limitChannelClusterConfigs(cfgs []wkdb.ChannelClusterConfig, limit int, pre bool) []wkdb.ChannelClusterConfig {
	if len(cfgs) == 0 {
		return nil
	}
	if limit > 0 && len(cfgs) > limit {
		if pre {
			cfgs = cfgs[len(cfgs)-limit:]
		} else {
			cfgs = cfgs[:limit]
		}
	}
	out := make([]wkdb.ChannelClusterConfig, 0, len(cfgs))
	for _, cfg := range cfgs {
		out = append(out, cloneChannelClusterConfig(cfg))
	}
	return out
}

func cloneChannelClusterConfig(cfg wkdb.ChannelClusterConfig) wkdb.ChannelClusterConfig {
	cfg.CreatedAt = cloneTimePtr(cfg.CreatedAt)
	cfg.UpdatedAt = cloneTimePtr(cfg.UpdatedAt)
	cfg.Replicas = append([]uint64(nil), cfg.Replicas...)
	cfg.Learners = append([]uint64(nil), cfg.Learners...)
	return cfg
}
