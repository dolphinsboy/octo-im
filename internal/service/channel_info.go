package service

import (
	"errors"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
)

// LoadChannelInfoOrEmpty treats a missing channel record as empty default config.
func LoadChannelInfoOrEmpty(channelId string, channelType uint8) (wkdb.ChannelInfo, error) {
	channelInfo, err := Store.GetChannel(channelId, channelType)
	if errors.Is(err, wkdb.ErrNotFound) {
		return wkdb.ChannelInfo{}, nil
	}
	return channelInfo, err
}
