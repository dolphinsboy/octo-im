package store

import "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"

func (h *HybridDB) slotMessageEventStore() *SlotMessageEventStore {
	return NewSlotMessageEventStore(h.slotDB, h.routeSlot)
}

func (h *HybridDB) AppendMessageEventWithState(event *wkdb.MessageEvent) (*wkdb.MessageEvent, *wkdb.MessageEventState, error) {
	if !h.hasSlotDB() {
		return h.DB.AppendMessageEventWithState(event)
	}
	return h.slotMessageEventStore().AppendMessageEventWithState(event)
}

func (h *HybridDB) GetMessageEventByEventID(channelID string, channelType uint8, clientMsgNo, eventID string) (*wkdb.MessageEvent, error) {
	if !h.hasSlotDB() {
		return h.DB.GetMessageEventByEventID(channelID, channelType, clientMsgNo, eventID)
	}
	return h.slotMessageEventStore().GetMessageEventByEventID(channelID, channelType, clientMsgNo, eventID)
}

func (h *HybridDB) ListMessageEvents(channelID string, channelType uint8, clientMsgNo string, fromMsgEventSeq uint64, eventKey string, limit int) ([]wkdb.MessageEvent, error) {
	if !h.hasSlotDB() {
		return h.DB.ListMessageEvents(channelID, channelType, clientMsgNo, fromMsgEventSeq, eventKey, limit)
	}
	return h.slotMessageEventStore().ListMessageEvents(channelID, channelType, clientMsgNo, fromMsgEventSeq, eventKey, limit)
}

func (h *HybridDB) GetMessageEventStates(channelID string, channelType uint8, clientMsgNo string) ([]wkdb.MessageEventState, error) {
	if !h.hasSlotDB() {
		return h.DB.GetMessageEventStates(channelID, channelType, clientMsgNo)
	}
	return h.slotMessageEventStore().GetMessageEventStates(channelID, channelType, clientMsgNo)
}

func (h *HybridDB) GetMessageEventStatesBatch(channelID string, channelType uint8, clientMsgNos []string) (map[string][]wkdb.MessageEventState, error) {
	if !h.hasSlotDB() {
		return h.DB.GetMessageEventStatesBatch(channelID, channelType, clientMsgNos)
	}
	return h.slotMessageEventStore().GetMessageEventStatesBatch(channelID, channelType, clientMsgNos)
}

func (h *HybridDB) GetMessageEventState(channelID string, channelType uint8, clientMsgNo, eventKey string) (*wkdb.MessageEventState, error) {
	if !h.hasSlotDB() {
		return h.DB.GetMessageEventState(channelID, channelType, clientMsgNo, eventKey)
	}
	return h.slotMessageEventStore().GetMessageEventState(channelID, channelType, clientMsgNo, eventKey)
}
