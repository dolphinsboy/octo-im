package store

import (
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
)

type MessageQueryStore interface {
	GetChannelLastMessageSeq(channelId string, channelType uint8) (seq uint64, lastTime uint64, err error)

	LoadNextRangeMsgs(channelId string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error)
	LoadMsg(channelId string, channelType uint8, seq uint64) (wkdb.Message, error)
	LoadLastMsgs(channelId string, channelType uint8, limit int) ([]wkdb.Message, error)
	LoadLastMsgsWithEnd(channelId string, channelType uint8, endMessageSeq uint64, limit int) ([]wkdb.Message, error)
	LoadPrevRangeMsgs(channelId string, channelType uint8, start, end uint64, limit int) ([]wkdb.Message, error)
}

type V3MessageStore struct {
	store wkdbv3.ChannelLogStore
}

func NewV3MessageStore(store wkdbv3.ChannelLogStore) *V3MessageStore {
	if store == nil {
		return nil
	}
	return &V3MessageStore{store: store}
}

func (v *V3MessageStore) GetLastMsg(channelId string, channelType uint8) (wkdb.Message, error) {
	return v.store.GetLastMsg(channelId, channelType)
}

func (v *V3MessageStore) AppendMessages(channelId string, channelType uint8, msgs []wkdb.Message) error {
	return v.store.AppendMessages(channelId, channelType, msgs)
}

func (v *V3MessageStore) LoadNextRangeMsgsForSize(channelId string, channelType uint8, startMessageSeq, endMessageSeq, limitSize uint64) ([]wkdb.Message, error) {
	return v.store.LoadNextRangeMsgsForSize(channelId, channelType, startMessageSeq, endMessageSeq, limitSize)
}

func (v *V3MessageStore) TruncateLogTo(channelId string, channelType uint8, messageSeq uint64) error {
	return v.store.TruncateLogTo(channelId, channelType, messageSeq)
}

func (v *V3MessageStore) GetChannelLastMessageSeq(channelId string, channelType uint8) (seq uint64, lastTime uint64, err error) {
	return v.store.GetChannelLastMessageSeq(channelId, channelType)
}

func (v *V3MessageStore) SetLeaderTermStartIndex(shardNo string, term uint32, index uint64) error {
	return v.store.SetLeaderTermStartIndex(shardNo, term, index)
}

func (v *V3MessageStore) LeaderTermStartIndex(shardNo string, term uint32) (uint64, error) {
	return v.store.LeaderTermStartIndex(shardNo, term)
}

func (v *V3MessageStore) LeaderLastTerm(shardNo string) (uint32, error) {
	return v.store.LeaderLastTerm(shardNo)
}

func (v *V3MessageStore) LeaderLastTermGreaterEqThan(shardNo string, term uint32) (uint32, error) {
	return v.store.LeaderLastTermGreaterEqThan(shardNo, term)
}

func (v *V3MessageStore) DeleteLeaderTermStartIndexGreaterThanTerm(shardNo string, term uint32) error {
	return v.store.DeleteLeaderTermStartIndexGreaterThanTerm(shardNo, term)
}

func (v *V3MessageStore) LoadNextRangeMsgs(channelId string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	return v.store.LoadNextRangeMsgs(channelId, channelType, startMessageSeq, endMessageSeq, limit)
}

func (v *V3MessageStore) LoadMsg(channelId string, channelType uint8, seq uint64) (wkdb.Message, error) {
	return v.store.LoadMsg(channelId, channelType, seq)
}

func (v *V3MessageStore) LoadLastMsgs(channelId string, channelType uint8, limit int) ([]wkdb.Message, error) {
	return v.store.LoadLastMsgs(channelId, channelType, limit)
}

func (v *V3MessageStore) LoadLastMsgsWithEnd(channelId string, channelType uint8, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	return v.store.LoadLastMsgsWithEnd(channelId, channelType, endMessageSeq, limit)
}

func (v *V3MessageStore) LoadPrevRangeMsgs(channelId string, channelType uint8, start, end uint64, limit int) ([]wkdb.Message, error) {
	return v.store.LoadPrevRangeMsgs(channelId, channelType, start, end, limit)
}

type LegacyChannelLogStore struct {
	db wkdb.DB
}

func NewLegacyChannelLogStore(db wkdb.DB) *LegacyChannelLogStore {
	if db == nil {
		return nil
	}
	return &LegacyChannelLogStore{db: db}
}

func (l *LegacyChannelLogStore) GetLastMsg(channelId string, channelType uint8) (wkdb.Message, error) {
	return l.db.GetLastMsg(channelId, channelType)
}

func (l *LegacyChannelLogStore) AppendMessages(channelId string, channelType uint8, msgs []wkdb.Message) error {
	return l.db.AppendMessages(channelId, channelType, msgs)
}

func (l *LegacyChannelLogStore) LoadNextRangeMsgsForSize(channelId string, channelType uint8, startMessageSeq, endMessageSeq, limitSize uint64) ([]wkdb.Message, error) {
	return l.db.LoadNextRangeMsgsForSize(channelId, channelType, startMessageSeq, endMessageSeq, limitSize)
}

func (l *LegacyChannelLogStore) TruncateLogTo(channelId string, channelType uint8, messageSeq uint64) error {
	return l.db.TruncateLogTo(channelId, channelType, messageSeq)
}

func (l *LegacyChannelLogStore) GetChannelLastMessageSeq(channelId string, channelType uint8) (seq uint64, lastTime uint64, err error) {
	return l.db.GetChannelLastMessageSeq(channelId, channelType)
}

func (l *LegacyChannelLogStore) SetLeaderTermStartIndex(shardNo string, term uint32, index uint64) error {
	return l.db.SetLeaderTermStartIndex(shardNo, term, index)
}

func (l *LegacyChannelLogStore) LeaderTermStartIndex(shardNo string, term uint32) (uint64, error) {
	return l.db.LeaderTermStartIndex(shardNo, term)
}

func (l *LegacyChannelLogStore) LeaderLastTerm(shardNo string) (uint32, error) {
	return l.db.LeaderLastTerm(shardNo)
}

func (l *LegacyChannelLogStore) LeaderLastTermGreaterEqThan(shardNo string, term uint32) (uint32, error) {
	return l.db.LeaderLastTermGreaterEqThan(shardNo, term)
}

func (l *LegacyChannelLogStore) DeleteLeaderTermStartIndexGreaterThanTerm(shardNo string, term uint32) error {
	return l.db.DeleteLeaderTermStartIndexGreaterThanTerm(shardNo, term)
}

type LegacyMessageQueryStore struct {
	db wkdb.DB
}

func NewLegacyMessageQueryStore(db wkdb.DB) *LegacyMessageQueryStore {
	if db == nil {
		return nil
	}
	return &LegacyMessageQueryStore{db: db}
}

func (l *LegacyMessageQueryStore) GetChannelLastMessageSeq(channelId string, channelType uint8) (seq uint64, lastTime uint64, err error) {
	return l.db.GetChannelLastMessageSeq(channelId, channelType)
}

func (l *LegacyMessageQueryStore) LoadNextRangeMsgs(channelId string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	return l.db.LoadNextRangeMsgs(channelId, channelType, startMessageSeq, endMessageSeq, limit)
}

func (l *LegacyMessageQueryStore) LoadMsg(channelId string, channelType uint8, seq uint64) (wkdb.Message, error) {
	return l.db.LoadMsg(channelId, channelType, seq)
}

func (l *LegacyMessageQueryStore) LoadLastMsgs(channelId string, channelType uint8, limit int) ([]wkdb.Message, error) {
	return l.db.LoadLastMsgs(channelId, channelType, limit)
}

func (l *LegacyMessageQueryStore) LoadLastMsgsWithEnd(channelId string, channelType uint8, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	return l.db.LoadLastMsgsWithEnd(channelId, channelType, endMessageSeq, limit)
}

func (l *LegacyMessageQueryStore) LoadPrevRangeMsgs(channelId string, channelType uint8, start, end uint64, limit int) ([]wkdb.Message, error) {
	return l.db.LoadPrevRangeMsgs(channelId, channelType, start, end, limit)
}
