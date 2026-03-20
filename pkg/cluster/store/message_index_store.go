package store

import "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"

type MessageIndexStore interface {
	LoadMsgByClientMsgNo(channelId string, channelType uint8, clientMsgNo string) (wkdb.Message, error)
	GetUserLastMsgSeq(fromUid string, channelId string, channelType uint8) (uint64, error)
	GetUserLastMsgSeqBatch(fromUid string, channels []wkdb.Channel) (map[string]uint64, error)
}

type LegacyMessageIndexStore struct {
	db wkdb.DB
}

func NewLegacyMessageIndexStore(db wkdb.DB) *LegacyMessageIndexStore {
	if db == nil {
		return nil
	}
	return &LegacyMessageIndexStore{db: db}
}

func (l *LegacyMessageIndexStore) LoadMsgByClientMsgNo(channelId string, channelType uint8, clientMsgNo string) (wkdb.Message, error) {
	return l.db.LoadMsgByClientMsgNo(channelId, channelType, clientMsgNo)
}

func (l *LegacyMessageIndexStore) GetUserLastMsgSeq(fromUid string, channelId string, channelType uint8) (uint64, error) {
	return l.db.GetUserLastMsgSeq(fromUid, channelId, channelType)
}

func (l *LegacyMessageIndexStore) GetUserLastMsgSeqBatch(fromUid string, channels []wkdb.Channel) (map[string]uint64, error) {
	return l.db.GetUserLastMsgSeqBatch(fromUid, channels)
}
