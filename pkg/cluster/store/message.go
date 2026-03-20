package store

import (
	"context"
	"fmt"

	"github.com/WuKongIM/WuKongIM/pkg/raft/types"
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
)

// AppendMessages 追加消息
func (s *Store) AppendMessages(ctx context.Context, channelId string, channelType uint8, msgs []wkdb.Message) (types.ProposeRespSet, error) {

	if len(msgs) == 0 {
		return nil, nil
	}
	reqs := make([]types.ProposeReq, 0, len(msgs))
	for _, msg := range msgs {
		data, err := msg.Marshal()
		if err != nil {
			return nil, err
		}

		reqs = append(reqs, types.ProposeReq{
			Id:   uint64(msg.MessageID),
			Data: data,
		})
	}

	results, err := s.opts.Channel.ProposeBatchUntilAppliedTimeout(ctx, channelId, channelType, reqs)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// AppendMessageChunks 追加消息chunks
func (s *Store) AppendMessageChunks(ctx context.Context, channelId string, channelType uint8, msgs []wkdb.Message) error {
	return nil
}

func (s *Store) LoadNextRangeMsgs(channelID string, channelType uint8, startMessageSeq, endMessageSeq uint64, limit int) ([]wkdb.Message, error) {
	return s.messageQueryStore.LoadNextRangeMsgs(channelID, channelType, startMessageSeq, endMessageSeq, limit)
}

func (s *Store) LoadMsg(channelID string, channelType uint8, seq uint64) (wkdb.Message, error) {
	return s.messageQueryStore.LoadMsg(channelID, channelType, seq)
}

func (s *Store) LoadLastMsgs(channelID string, channelType uint8, limit int) ([]wkdb.Message, error) {
	return s.messageQueryStore.LoadLastMsgs(channelID, channelType, limit)
}

func (s *Store) LoadLastMsgsWithEnd(channelID string, channelType uint8, end uint64, limit int) ([]wkdb.Message, error) {
	return s.messageQueryStore.LoadLastMsgsWithEnd(channelID, channelType, end, limit)
}

func (s *Store) LoadPrevRangeMsgs(channelID string, channelType uint8, start, end uint64, limit int) ([]wkdb.Message, error) {
	return s.messageQueryStore.LoadPrevRangeMsgs(channelID, channelType, start, end, limit)
}

func (s *Store) GetLastMsgSeq(channelID string, channelType uint8) (uint64, error) {
	seq, _, err := s.messageQueryStore.GetChannelLastMessageSeq(channelID, channelType)
	return seq, err
}

func (s *Store) GetChannelLastMessageSeqAndTime(channelID string, channelType uint8) (uint64, uint64, error) {
	return s.messageQueryStore.GetChannelLastMessageSeq(channelID, channelType)
}

// Notify queue is node-local state and intentionally remains outside the slot log boundary.
func (s *Store) GetMessagesOfNotifyQueue(count int) ([]wkdb.Message, error) {
	return s.notifyQueueStore.GetMessagesOfNotifyQueue(count)
}

func (s *Store) AppendMessageOfNotifyQueue(messages []wkdb.Message) error {
	return s.notifyQueueStore.AppendMessageOfNotifyQueue(messages)
}

func (s *Store) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
	return s.notifyQueueStore.RemoveMessagesOfNotifyQueue(messageIDs)
}

func (s *Store) RemoveMessagesOfNotifyQueueCount(count int) error {
	return s.notifyQueueStore.RemoveMessagesOfNotifyQueueCount(count)
}

func (s *Store) DeleteChannelAndClearMessages(channelID string, channelType uint8) error {
	return nil
}

func (s *Store) requireMessageSearchStore() (MessageSearchStore, error) {
	if s.messageSearchStore == nil {
		return nil, fmt.Errorf("message search store is not configured")
	}
	return s.messageSearchStore, nil
}

func (s *Store) requireMessageIndexStore() (MessageIndexStore, error) {
	if s.messageIndexStore == nil {
		return nil, fmt.Errorf("message index store is not configured")
	}
	return s.messageIndexStore, nil
}

// 搜索消息
func (s *Store) SearchMessages(req wkdb.MessageSearchReq) ([]wkdb.Message, error) {
	messageSearchStore, err := s.requireMessageSearchStore()
	if err != nil {
		return nil, err
	}
	return messageSearchStore.SearchMessages(req)
}

func (s *Store) CountMessages() (int, error) {
	messageSearchStore, err := s.requireMessageSearchStore()
	if err != nil {
		return 0, err
	}
	return messageSearchStore.CountMessages()
}

// LoadMsgByClientMsgNo 通过 clientMsgNo 加载消息
func (s *Store) LoadMsgByClientMsgNo(channelId string, channelType uint8, clientMsgNo string) (wkdb.Message, error) {
	messageIndexStore, err := s.requireMessageIndexStore()
	if err != nil {
		return wkdb.EmptyMessage, err
	}
	return messageIndexStore.LoadMsgByClientMsgNo(channelId, channelType, clientMsgNo)
}

// GetUserLastMsgSeq 获取用户在指定频道内发送的最新一条消息的seq
func (s *Store) GetUserLastMsgSeq(fromUid string, channelId string, channelType uint8) (uint64, error) {
	messageIndexStore, err := s.requireMessageIndexStore()
	if err != nil {
		return 0, err
	}
	return messageIndexStore.GetUserLastMsgSeq(fromUid, channelId, channelType)
}

// LoadMsgsBatch 批量获取多个频道的消息
func (s *Store) LoadMsgsBatch(requests []wkdb.BatchMsgRequest) ([]wkdb.BatchMsgResponse, error) {
	if len(requests) == 0 {
		return nil, nil
	}
	responses := make([]wkdb.BatchMsgResponse, 0, len(requests))
	for _, req := range requests {
		var (
			messages []wkdb.Message
			err      error
		)
		if req.OrderByLast {
			messages, err = s.messageQueryStore.LoadLastMsgsWithEnd(req.ChannelId, req.ChannelType, req.MsgSeq, req.Limit)
		} else {
			messages, err = s.messageQueryStore.LoadNextRangeMsgs(req.ChannelId, req.ChannelType, req.MsgSeq, 0, req.Limit)
		}
		if err != nil {
			return nil, err
		}
		responses = append(responses, wkdb.BatchMsgResponse{
			ChannelId:   req.ChannelId,
			ChannelType: req.ChannelType,
			Messages:    messages,
		})
	}
	return responses, nil
}

// GetUserLastMsgSeqBatch 批量获取用户在多个频道的最后消息序号
func (s *Store) GetUserLastMsgSeqBatch(fromUid string, channels []wkdb.Channel) (map[string]uint64, error) {
	messageIndexStore, err := s.requireMessageIndexStore()
	if err != nil {
		return nil, err
	}
	return messageIndexStore.GetUserLastMsgSeqBatch(fromUid, channels)
}
