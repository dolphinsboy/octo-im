package store

import "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"

type MessageStore interface {
	GetLastMsg(channelId string, channelType uint8) (wkdb.Message, error)
	AppendMessages(channelId string, channelType uint8, msgs []wkdb.Message) error
	LoadNextRangeMsgsForSize(channelId string, channelType uint8, startMessageSeq, endMessageSeq, limitSize uint64) ([]wkdb.Message, error)
	TruncateLogTo(channelId string, channelType uint8, messageSeq uint64) error
	GetChannelLastMessageSeq(channelId string, channelType uint8) (seq uint64, lastTime uint64, err error)

	SetLeaderTermStartIndex(shardNo string, term uint32, index uint64) error
	LeaderTermStartIndex(shardNo string, term uint32) (uint64, error)
	LeaderLastTerm(shardNo string) (uint32, error)
	LeaderLastTermGreaterEqThan(shardNo string, term uint32) (uint32, error)
	DeleteLeaderTermStartIndexGreaterThanTerm(shardNo string, term uint32) error

	LoadNextRangeMsgs(channelId string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error)
	LoadMsg(channelId string, channelType uint8, seq uint64) (wkdb.Message, error)
	LoadLastMsgs(channelId string, channelType uint8, limit int) ([]wkdb.Message, error)
	LoadLastMsgsWithEnd(channelId string, channelType uint8, endMessageSeq uint64, limit int) ([]wkdb.Message, error)
	LoadPrevRangeMsgs(channelId string, channelType uint8, start, end uint64, limit int) ([]wkdb.Message, error)

	GetMessagesOfNotifyQueue(count int) ([]wkdb.Message, error)
	AppendMessageOfNotifyQueue(messages []wkdb.Message) error
	RemoveMessagesOfNotifyQueue(messageIDs []int64) error
	RemoveMessagesOfNotifyQueueCount(count int) error

	SearchMessages(req wkdb.MessageSearchReq) ([]wkdb.Message, error)
	LoadMsgByClientMsgNo(channelId string, channelType uint8, clientMsgNo string) (wkdb.Message, error)
	GetUserLastMsgSeq(fromUid string, channelId string, channelType uint8) (uint64, error)
	LoadMsgsBatch(requests []wkdb.BatchMsgRequest) ([]wkdb.BatchMsgResponse, error)
	GetUserLastMsgSeqBatch(fromUid string, channels []wkdb.Channel) (map[string]uint64, error)
}

type LegacyMessageStore struct {
	db wkdb.DB
}

func NewLegacyMessageStore(db wkdb.DB) *LegacyMessageStore {
	if db == nil {
		return nil
	}
	return &LegacyMessageStore{db: db}
}

func (l *LegacyMessageStore) GetLastMsg(channelId string, channelType uint8) (wkdb.Message, error) {
	return l.db.GetLastMsg(channelId, channelType)
}

func (l *LegacyMessageStore) AppendMessages(channelId string, channelType uint8, msgs []wkdb.Message) error {
	return l.db.AppendMessages(channelId, channelType, msgs)
}

func (l *LegacyMessageStore) LoadNextRangeMsgsForSize(channelId string, channelType uint8, startMessageSeq, endMessageSeq, limitSize uint64) ([]wkdb.Message, error) {
	return l.db.LoadNextRangeMsgsForSize(channelId, channelType, startMessageSeq, endMessageSeq, limitSize)
}

func (l *LegacyMessageStore) TruncateLogTo(channelId string, channelType uint8, messageSeq uint64) error {
	return l.db.TruncateLogTo(channelId, channelType, messageSeq)
}

func (l *LegacyMessageStore) GetChannelLastMessageSeq(channelId string, channelType uint8) (seq uint64, lastTime uint64, err error) {
	return l.db.GetChannelLastMessageSeq(channelId, channelType)
}

func (l *LegacyMessageStore) SetLeaderTermStartIndex(shardNo string, term uint32, index uint64) error {
	return l.db.SetLeaderTermStartIndex(shardNo, term, index)
}

func (l *LegacyMessageStore) LeaderTermStartIndex(shardNo string, term uint32) (uint64, error) {
	return l.db.LeaderTermStartIndex(shardNo, term)
}

func (l *LegacyMessageStore) LeaderLastTerm(shardNo string) (uint32, error) {
	return l.db.LeaderLastTerm(shardNo)
}

func (l *LegacyMessageStore) LeaderLastTermGreaterEqThan(shardNo string, term uint32) (uint32, error) {
	return l.db.LeaderLastTermGreaterEqThan(shardNo, term)
}

func (l *LegacyMessageStore) DeleteLeaderTermStartIndexGreaterThanTerm(shardNo string, term uint32) error {
	return l.db.DeleteLeaderTermStartIndexGreaterThanTerm(shardNo, term)
}

func (l *LegacyMessageStore) LoadNextRangeMsgs(channelId string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	return l.db.LoadNextRangeMsgs(channelId, channelType, startMessageSeq, endMessageSeq, limit)
}

func (l *LegacyMessageStore) LoadMsg(channelId string, channelType uint8, seq uint64) (wkdb.Message, error) {
	return l.db.LoadMsg(channelId, channelType, seq)
}

func (l *LegacyMessageStore) LoadLastMsgs(channelId string, channelType uint8, limit int) ([]wkdb.Message, error) {
	return l.db.LoadLastMsgs(channelId, channelType, limit)
}

func (l *LegacyMessageStore) LoadLastMsgsWithEnd(channelId string, channelType uint8, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	return l.db.LoadLastMsgsWithEnd(channelId, channelType, endMessageSeq, limit)
}

func (l *LegacyMessageStore) LoadPrevRangeMsgs(channelId string, channelType uint8, start, end uint64, limit int) ([]wkdb.Message, error) {
	return l.db.LoadPrevRangeMsgs(channelId, channelType, start, end, limit)
}

func (l *LegacyMessageStore) GetMessagesOfNotifyQueue(count int) ([]wkdb.Message, error) {
	return l.db.GetMessagesOfNotifyQueue(count)
}

func (l *LegacyMessageStore) AppendMessageOfNotifyQueue(messages []wkdb.Message) error {
	return l.db.AppendMessageOfNotifyQueue(messages)
}

func (l *LegacyMessageStore) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
	return l.db.RemoveMessagesOfNotifyQueue(messageIDs)
}

func (l *LegacyMessageStore) RemoveMessagesOfNotifyQueueCount(count int) error {
	return l.db.RemoveMessagesOfNotifyQueueCount(count)
}

func (l *LegacyMessageStore) SearchMessages(req wkdb.MessageSearchReq) ([]wkdb.Message, error) {
	return l.db.SearchMessages(req)
}

func (l *LegacyMessageStore) LoadMsgByClientMsgNo(channelId string, channelType uint8, clientMsgNo string) (wkdb.Message, error) {
	return l.db.LoadMsgByClientMsgNo(channelId, channelType, clientMsgNo)
}

func (l *LegacyMessageStore) GetUserLastMsgSeq(fromUid string, channelId string, channelType uint8) (uint64, error) {
	return l.db.GetUserLastMsgSeq(fromUid, channelId, channelType)
}

func (l *LegacyMessageStore) LoadMsgsBatch(requests []wkdb.BatchMsgRequest) ([]wkdb.BatchMsgResponse, error) {
	return l.db.LoadMsgsBatch(requests)
}

func (l *LegacyMessageStore) GetUserLastMsgSeqBatch(fromUid string, channels []wkdb.Channel) (map[string]uint64, error) {
	return l.db.GetUserLastMsgSeqBatch(fromUid, channels)
}
