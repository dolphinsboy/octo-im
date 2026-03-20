package store

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
)

type SlotMessageEventStore struct {
	db        wkdbv3.DB
	routeSlot func(key string) uint32
}

var _ MessageEventStore = (*SlotMessageEventStore)(nil)

func NewSlotMessageEventStore(db wkdbv3.DB, routeSlot func(key string) uint32) *SlotMessageEventStore {
	if db == nil || routeSlot == nil {
		return nil
	}
	return &SlotMessageEventStore{
		db:        db,
		routeSlot: routeSlot,
	}
}

func (s *SlotMessageEventStore) slotScopeByChannel(channelID string) wkdbv3.SlotScope {
	return s.db.Slots().Scope(s.routeSlot(channelID))
}

func (s *SlotMessageEventStore) AppendMessageEventWithState(event *wkdb.MessageEvent) (*wkdb.MessageEvent, *wkdb.MessageEventState, error) {
	if event == nil {
		return nil, nil, wkdb.ErrNotFound
	}
	if strings.TrimSpace(event.ChannelId) == "" || strings.TrimSpace(event.ClientMsgNo) == "" || strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.EventType) == "" {
		return nil, nil, wkdb.ErrNotFound
	}

	event.ChannelId = strings.TrimSpace(event.ChannelId)
	event.ClientMsgNo = strings.TrimSpace(event.ClientMsgNo)
	event.EventID = strings.TrimSpace(event.EventID)
	event.EventType = strings.ToLower(strings.TrimSpace(event.EventType))
	event.EventKey = normalizeEventKey(event.EventKey)
	if event.OccurredAt == 0 {
		event.OccurredAt = time.Now().UnixMilli()
	}

	store := s.slotScopeByChannel(event.ChannelId).MessageEvents()
	state, err := store.GetState(event.ChannelId, event.ChannelType, event.ClientMsgNo, event.EventKey)
	if err != nil {
		return nil, nil, err
	}
	if state != nil && state.LastEventID != "" && state.LastEventID == event.EventID {
		cp := *event
		cp.MsgEventSeq = state.LastMsgEventSeq
		cp.EventKey = state.EventKey
		cp.EventType = state.LastEventType
		cp.Visibility = state.LastVisibility
		cp.OccurredAt = state.LastOccurredAt
		if len(state.SnapshotPayload) > 0 {
			cp.Payload = append([]byte(nil), state.SnapshotPayload...)
		}
		cloned := cloneMessageEventState(*state)
		return &cp, &cloned, nil
	}
	if state != nil && isTerminalStatus(state.Status) {
		projected := buildProjectedEvent(event.ChannelId, event.ChannelType, event.ClientMsgNo, *state)
		cloned := cloneMessageEventState(*state)
		return projected, &cloned, nil
	}

	seq, err := store.GetSeq(event.ChannelId, event.ChannelType, event.ClientMsgNo)
	if err != nil {
		return nil, nil, err
	}
	seq++
	event.MsgEventSeq = seq
	merged := reduceEventState(state, event)
	if err := store.PutState(*merged); err != nil {
		return nil, nil, err
	}
	if err := store.SetSeq(event.ChannelId, event.ChannelType, event.ClientMsgNo, seq); err != nil {
		return nil, nil, err
	}
	cloned := cloneMessageEventState(*merged)
	cp := *event
	cp.Payload = append([]byte(nil), event.Payload...)
	return &cp, &cloned, nil
}

func (s *SlotMessageEventStore) GetMessageEventByEventID(channelID string, channelType uint8, clientMsgNo, eventID string) (*wkdb.MessageEvent, error) {
	states, err := s.GetMessageEventStates(channelID, channelType, clientMsgNo)
	if err != nil {
		return nil, err
	}
	for _, state := range states {
		if state.LastEventID != strings.TrimSpace(eventID) {
			continue
		}
		return buildProjectedEvent(channelID, channelType, clientMsgNo, state), nil
	}
	return nil, nil
}

func (s *SlotMessageEventStore) ListMessageEvents(channelID string, channelType uint8, clientMsgNo string, fromMsgEventSeq uint64, eventKey string, limit int) ([]wkdb.MessageEvent, error) {
	states, err := s.GetMessageEventStates(channelID, channelType, clientMsgNo)
	if err != nil {
		return nil, err
	}
	eventKey = strings.TrimSpace(eventKey)
	if eventKey != "" {
		eventKey = normalizeEventKey(eventKey)
	}
	result := make([]wkdb.MessageEvent, 0, len(states))
	for _, state := range states {
		if eventKey != "" && state.EventKey != eventKey {
			continue
		}
		if state.LastMsgEventSeq <= fromMsgEventSeq {
			continue
		}
		projected := buildProjectedEvent(channelID, channelType, clientMsgNo, state)
		if projected != nil {
			result = append(result, *projected)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].MsgEventSeq < result[j].MsgEventSeq
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *SlotMessageEventStore) GetMessageEventStates(channelID string, channelType uint8, clientMsgNo string) ([]wkdb.MessageEventState, error) {
	return s.slotScopeByChannel(channelID).MessageEvents().GetStates(channelID, channelType, clientMsgNo)
}

func (s *SlotMessageEventStore) GetMessageEventStatesBatch(channelID string, channelType uint8, clientMsgNos []string) (map[string][]wkdb.MessageEventState, error) {
	result := make(map[string][]wkdb.MessageEventState, len(clientMsgNos))
	for _, clientMsgNo := range clientMsgNos {
		states, err := s.GetMessageEventStates(channelID, channelType, clientMsgNo)
		if err != nil {
			return nil, err
		}
		result[clientMsgNo] = states
	}
	return result, nil
}

func (s *SlotMessageEventStore) GetMessageEventState(channelID string, channelType uint8, clientMsgNo, eventKey string) (*wkdb.MessageEventState, error) {
	state, err := s.slotScopeByChannel(channelID).MessageEvents().GetState(channelID, channelType, clientMsgNo, eventKey)
	if err != nil || state == nil {
		return state, err
	}
	cloned := cloneMessageEventState(*state)
	return &cloned, nil
}

func normalizeEventKey(eventKey string) string {
	eventKey = strings.TrimSpace(eventKey)
	if eventKey == "" {
		return wkdb.EventKeyDefault
	}
	return eventKey
}

func isTerminalStatus(status string) bool {
	switch status {
	case wkdb.EventStatusClosed, wkdb.EventStatusError, wkdb.EventStatusCancelled:
		return true
	default:
		return false
	}
}

func reduceEventState(state *wkdb.MessageEventState, event *wkdb.MessageEvent) *wkdb.MessageEventState {
	if state == nil {
		state = &wkdb.MessageEventState{}
	} else {
		cloned := cloneMessageEventState(*state)
		state = &cloned
	}
	if state.ChannelId == "" {
		state.ChannelId = event.ChannelId
		state.ChannelType = event.ChannelType
	}
	if state.ClientMsgNo == "" {
		state.ClientMsgNo = event.ClientMsgNo
	}
	if state.EventKey == "" {
		state.EventKey = event.EventKey
	}
	state.LastEventID = event.EventID
	state.LastEventType = event.EventType
	state.LastVisibility = event.Visibility
	state.LastOccurredAt = event.OccurredAt

	if isTerminalStatus(state.Status) {
		return state
	}

	switch event.EventType {
	case wkdb.EventTypeStreamDelta:
		if state.Status == "" {
			state.Status = wkdb.EventStatusOpen
		}
		applyDeltaSnapshot(state, event)
	case wkdb.EventTypeStreamSnapshot:
		state.SnapshotPayload = append([]byte(nil), event.Payload...)
	case wkdb.EventTypeStreamClose:
		if snapshot := extractSnapshot(event.Payload); len(snapshot) > 0 {
			state.SnapshotPayload = snapshot
		}
		state.Status = wkdb.EventStatusClosed
		state.EndReason = extractEndReason(event.Payload)
	case wkdb.EventTypeStreamError:
		if snapshot := extractSnapshot(event.Payload); len(snapshot) > 0 {
			state.SnapshotPayload = snapshot
		}
		state.Status = wkdb.EventStatusError
		state.Error = extractError(event.Payload)
	case wkdb.EventTypeStreamCancel:
		if snapshot := extractSnapshot(event.Payload); len(snapshot) > 0 {
			state.SnapshotPayload = snapshot
		}
		state.Status = wkdb.EventStatusCancelled
	case wkdb.EventTypeStreamFinish:
		state.Status = wkdb.EventStatusClosed
	}
	state.LastMsgEventSeq = event.MsgEventSeq
	return state
}

func applyDeltaSnapshot(state *wkdb.MessageEventState, event *wkdb.MessageEvent) {
	if len(event.Payload) == 0 {
		return
	}
	payload := map[string]interface{}{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		state.SnapshotPayload = append(state.SnapshotPayload, event.Payload...)
		return
	}
	kind, _ := payload["kind"].(string)
	if kind != wkdb.SnapshotKindText {
		state.SnapshotPayload = append([]byte(nil), event.Payload...)
		return
	}
	delta, _ := payload["delta"].(string)
	if delta == "" {
		return
	}
	snapshot := map[string]interface{}{"kind": wkdb.SnapshotKindText, "text": ""}
	if len(state.SnapshotPayload) > 0 {
		_ = json.Unmarshal(state.SnapshotPayload, &snapshot)
	}
	text, _ := snapshot["text"].(string)
	snapshot["kind"] = wkdb.SnapshotKindText
	snapshot["text"] = text + delta
	data, _ := json.Marshal(snapshot)
	state.SnapshotPayload = data
}

func extractEndReason(payload []byte) uint8 {
	if len(payload) == 0 {
		return 0
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(payload, &m); err != nil {
		return 0
	}
	v, ok := m["end_reason"]
	if !ok {
		return 0
	}
	switch vv := v.(type) {
	case float64:
		return uint8(vv)
	case int:
		return uint8(vv)
	case string:
		n, _ := strconv.ParseUint(vv, 10, 8)
		return uint8(n)
	default:
		return 0
	}
}

func extractError(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(payload, &m); err != nil {
		return ""
	}
	errMsg, _ := m["error"].(string)
	return errMsg
}

func extractSnapshot(payload []byte) []byte {
	if len(payload) == 0 {
		return nil
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil
	}
	raw, ok := m["snapshot"]
	if !ok {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	return data
}

func buildProjectedEvent(channelID string, channelType uint8, clientMsgNo string, state wkdb.MessageEventState) *wkdb.MessageEvent {
	eventType := state.LastEventType
	if strings.TrimSpace(eventType) == "" {
		eventType = wkdb.EventTypeStreamSnapshot
	}
	payload := append([]byte(nil), state.SnapshotPayload...)
	if len(payload) == 0 {
		payloadMap := map[string]interface{}{
			"status":     state.Status,
			"end_reason": state.EndReason,
		}
		if state.Error != "" {
			payloadMap["error"] = state.Error
		}
		if data, err := json.Marshal(payloadMap); err == nil {
			payload = data
		}
	}
	return &wkdb.MessageEvent{
		ChannelId:   channelID,
		ChannelType: channelType,
		ClientMsgNo: clientMsgNo,
		MsgEventSeq: state.LastMsgEventSeq,
		EventID:     state.LastEventID,
		EventKey:    state.EventKey,
		EventType:   eventType,
		Visibility:  state.LastVisibility,
		OccurredAt:  state.LastOccurredAt,
		Payload:     payload,
	}
}

func cloneMessageEventState(state wkdb.MessageEventState) wkdb.MessageEventState {
	state.SnapshotPayload = append([]byte(nil), state.SnapshotPayload...)
	return state
}
