package store

import (
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkdbv3 "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3"
)

type NotifyQueueStore interface {
	GetMessagesOfNotifyQueue(count int) ([]wkdb.Message, error)
	AppendMessageOfNotifyQueue(messages []wkdb.Message) error
	RemoveMessagesOfNotifyQueue(messageIDs []int64) error
	RemoveMessagesOfNotifyQueueCount(count int) error
}

type LegacyNotifyQueueStore struct {
	db wkdb.DB
}

var _ NotifyQueueStore = (*LegacyNotifyQueueStore)(nil)

func NewLegacyNotifyQueueStore(db wkdb.DB) *LegacyNotifyQueueStore {
	if db == nil {
		return nil
	}
	return &LegacyNotifyQueueStore{db: db}
}

func (l *LegacyNotifyQueueStore) GetMessagesOfNotifyQueue(count int) ([]wkdb.Message, error) {
	return l.db.GetMessagesOfNotifyQueue(count)
}

func (l *LegacyNotifyQueueStore) AppendMessageOfNotifyQueue(messages []wkdb.Message) error {
	return l.db.AppendMessageOfNotifyQueue(messages)
}

func (l *LegacyNotifyQueueStore) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
	return l.db.RemoveMessagesOfNotifyQueue(messageIDs)
}

func (l *LegacyNotifyQueueStore) RemoveMessagesOfNotifyQueueCount(count int) error {
	return l.db.RemoveMessagesOfNotifyQueueCount(count)
}

type LocalNotifyQueueStore struct {
	db wkdbv3.DB
}

var _ NotifyQueueStore = (*LocalNotifyQueueStore)(nil)

func NewLocalNotifyQueueStore(db wkdbv3.DB) *LocalNotifyQueueStore {
	if db == nil {
		return nil
	}
	return &LocalNotifyQueueStore{db: db}
}

func (l *LocalNotifyQueueStore) GetMessagesOfNotifyQueue(count int) ([]wkdb.Message, error) {
	return l.db.Local().NotifyQueue().List(count)
}

func (l *LocalNotifyQueueStore) AppendMessageOfNotifyQueue(messages []wkdb.Message) error {
	return l.db.Local().NotifyQueue().Put(messages)
}

func (l *LocalNotifyQueueStore) RemoveMessagesOfNotifyQueue(messageIDs []int64) error {
	return l.db.Local().NotifyQueue().Delete(messageIDs)
}

func (l *LocalNotifyQueueStore) RemoveMessagesOfNotifyQueueCount(count int) error {
	return l.db.Local().NotifyQueue().DeleteCount(count)
}
