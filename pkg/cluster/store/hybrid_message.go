package store

import (
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
)

func (h *HybridDB) channelLogStore() wkdbv3.ChannelLogStore {
	if !h.hasV3DB() {
		return nil
	}
	return h.slotDB.ChannelLogs()
}

func (h *HybridDB) GetMessage(messageID uint64) (wkdb.Message, error) {
	if store := h.channelLogStore(); store != nil {
		return store.GetMessage(messageID)
	}
	return h.DB.GetMessage(messageID)
}

func (h *HybridDB) AppendMessages(channelID string, channelType uint8, msgs []wkdb.Message) error {
	if store := h.channelLogStore(); store != nil {
		return store.AppendMessages(channelID, channelType, msgs)
	}
	return h.DB.AppendMessages(channelID, channelType, msgs)
}

func (h *HybridDB) LoadPrevRangeMsgs(channelID string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LoadPrevRangeMsgs(channelID, channelType, startMessageSeq, endMessageSeq, limit)
	}
	return h.DB.LoadPrevRangeMsgs(channelID, channelType, startMessageSeq, endMessageSeq, limit)
}

func (h *HybridDB) LoadNextRangeMsgs(channelID string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LoadNextRangeMsgs(channelID, channelType, startMessageSeq, endMessageSeq, limit)
	}
	return h.DB.LoadNextRangeMsgs(channelID, channelType, startMessageSeq, endMessageSeq, limit)
}

func (h *HybridDB) LoadNextRangeMsgsForSize(channelID string, channelType uint8, startMessageSeq, endMessageSeq, limitSize uint64) ([]wkdb.Message, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LoadNextRangeMsgsForSize(channelID, channelType, startMessageSeq, endMessageSeq, limitSize)
	}
	return h.DB.LoadNextRangeMsgsForSize(channelID, channelType, startMessageSeq, endMessageSeq, limitSize)
}

func (h *HybridDB) LoadMsg(channelID string, channelType uint8, seq uint64) (wkdb.Message, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LoadMsg(channelID, channelType, seq)
	}
	return h.DB.LoadMsg(channelID, channelType, seq)
}

func (h *HybridDB) TruncateLogTo(channelID string, channelType uint8, messageSeq uint64) error {
	if store := h.channelLogStore(); store != nil {
		return store.TruncateLogTo(channelID, channelType, messageSeq)
	}
	return h.DB.TruncateLogTo(channelID, channelType, messageSeq)
}

func (h *HybridDB) LoadLastMsgsWithEnd(channelID string, channelType uint8, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LoadLastMsgsWithEnd(channelID, channelType, endMessageSeq, limit)
	}
	return h.DB.LoadLastMsgsWithEnd(channelID, channelType, endMessageSeq, limit)
}

func (h *HybridDB) LoadLastMsgs(channelID string, channelType uint8, limit int) ([]wkdb.Message, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LoadLastMsgs(channelID, channelType, limit)
	}
	return h.DB.LoadLastMsgs(channelID, channelType, limit)
}

func (h *HybridDB) GetChannelLastMessageSeq(channelID string, channelType uint8) (uint64, uint64, error) {
	if store := h.channelLogStore(); store != nil {
		return store.GetChannelLastMessageSeq(channelID, channelType)
	}
	return h.DB.GetChannelLastMessageSeq(channelID, channelType)
}

func (h *HybridDB) SetChannelLastMessageSeq(channelID string, channelType uint8, seq uint64) error {
	if store := h.channelLogStore(); store != nil {
		return store.SetChannelLastMessageSeq(channelID, channelType, seq)
	}
	return h.DB.SetChannelLastMessageSeq(channelID, channelType, seq)
}

func (h *HybridDB) SearchMessages(req wkdb.MessageSearchReq) ([]wkdb.Message, error) {
	if store := h.channelLogStore(); store != nil {
		return store.SearchMessages(req)
	}
	return h.DB.SearchMessages(req)
}

func (h *HybridDB) GetLastMsg(channelID string, channelType uint8) (wkdb.Message, error) {
	if store := h.channelLogStore(); store != nil {
		return store.GetLastMsg(channelID, channelType)
	}
	return h.DB.GetLastMsg(channelID, channelType)
}

func (h *HybridDB) LoadMsgByClientMsgNo(channelID string, channelType uint8, clientMsgNo string) (wkdb.Message, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LoadMsgByClientMsgNo(channelID, channelType, clientMsgNo)
	}
	return h.DB.LoadMsgByClientMsgNo(channelID, channelType, clientMsgNo)
}

func (h *HybridDB) GetUserLastMsgSeq(fromUID string, channelID string, channelType uint8) (uint64, error) {
	if store := h.channelLogStore(); store != nil {
		return store.GetUserLastMsgSeq(fromUID, channelID, channelType)
	}
	return h.DB.GetUserLastMsgSeq(fromUID, channelID, channelType)
}

func (h *HybridDB) LoadMsgsBatch(requests []wkdb.BatchMsgRequest) ([]wkdb.BatchMsgResponse, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LoadMsgsBatch(requests)
	}
	return h.DB.LoadMsgsBatch(requests)
}

func (h *HybridDB) GetUserLastMsgSeqBatch(fromUID string, channels []wkdb.Channel) (map[string]uint64, error) {
	if store := h.channelLogStore(); store != nil {
		return store.GetUserLastMsgSeqBatch(fromUID, channels)
	}
	return h.DB.GetUserLastMsgSeqBatch(fromUID, channels)
}

func (h *HybridDB) SetLeaderTermStartIndex(shardNo string, term uint32, index uint64) error {
	if store := h.channelLogStore(); store != nil {
		return store.SetLeaderTermStartIndex(shardNo, term, index)
	}
	return h.DB.SetLeaderTermStartIndex(shardNo, term, index)
}

func (h *HybridDB) LeaderTermStartIndex(shardNo string, term uint32) (uint64, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LeaderTermStartIndex(shardNo, term)
	}
	return h.DB.LeaderTermStartIndex(shardNo, term)
}

func (h *HybridDB) LeaderLastTerm(shardNo string) (uint32, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LeaderLastTerm(shardNo)
	}
	return h.DB.LeaderLastTerm(shardNo)
}

func (h *HybridDB) LeaderLastTermGreaterEqThan(shardNo string, term uint32) (uint32, error) {
	if store := h.channelLogStore(); store != nil {
		return store.LeaderLastTermGreaterEqThan(shardNo, term)
	}
	return h.DB.LeaderLastTermGreaterEqThan(shardNo, term)
}

func (h *HybridDB) DeleteLeaderTermStartIndexGreaterThanTerm(shardNo string, term uint32) error {
	if store := h.channelLogStore(); store != nil {
		return store.DeleteLeaderTermStartIndexGreaterThanTerm(shardNo, term)
	}
	return h.DB.DeleteLeaderTermStartIndexGreaterThanTerm(shardNo, term)
}

func (h *HybridDB) GetTotalMessageCount() (int, error) {
	if store := h.channelLogStore(); store != nil {
		return store.CountMessages()
	}
	return h.DB.GetTotalMessageCount()
}
