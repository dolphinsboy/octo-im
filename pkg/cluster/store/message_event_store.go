package store

import "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"

type MessageEventStore interface {
	AppendMessageEventWithState(event *wkdb.MessageEvent) (*wkdb.MessageEvent, *wkdb.MessageEventState, error)
	GetMessageEventByEventID(channelId string, channelType uint8, clientMsgNo, eventID string) (*wkdb.MessageEvent, error)
	ListMessageEvents(channelId string, channelType uint8, clientMsgNo string, fromMsgEventSeq uint64, eventKey string, limit int) ([]wkdb.MessageEvent, error)
	GetMessageEventStates(channelId string, channelType uint8, clientMsgNo string) ([]wkdb.MessageEventState, error)
	GetMessageEventStatesBatch(channelId string, channelType uint8, clientMsgNos []string) (map[string][]wkdb.MessageEventState, error)
	GetMessageEventState(channelId string, channelType uint8, clientMsgNo, eventKey string) (*wkdb.MessageEventState, error)
}
