package v3

import (
	"sort"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/cockroachdb/pebble"
)

type pebbleLocalDB struct {
	owner *PebbleDB
}

var _ LocalDB = (*pebbleLocalDB)(nil)

func (p *pebbleLocalDB) NotifyQueue() NotifyQueueStore {
	return &pebbleNotifyQueueStore{owner: p.owner}
}

type pebbleNotifyQueueStore struct {
	owner *PebbleDB
}

var _ NotifyQueueStore = (*pebbleNotifyQueueStore)(nil)

func (p *pebbleNotifyQueueStore) Put(messages []wkdb.Message) error {
	if len(messages) == 0 {
		return nil
	}
	db, err := p.owner.localDB()
	if err != nil {
		return err
	}
	batch := db.NewBatch()
	defer batch.Close()

	for _, message := range messages {
		data, err := message.Marshal()
		if err != nil {
			return err
		}
		if err := batch.Set(v3key.EncodeNotifyQueueRowKey(uint64(message.MessageID)), data, p.owner.standaloneWriteOptions()); err != nil {
			return err
		}
	}
	return batch.Commit(p.owner.standaloneWriteOptions())
}

func (p *pebbleNotifyQueueStore) List(count int) ([]wkdb.Message, error) {
	db, err := p.owner.localDB()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.MetaTableRange(v3key.ScopeLocal, v3key.TableMessageNotifyQueue)
	iter := db.NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	messages := make([]wkdb.Message, 0)
	invalidKeys := make([][]byte, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		var message wkdb.Message
		if err := message.Unmarshal(iter.Value()); err != nil {
			invalidKeys = append(invalidKeys, append([]byte(nil), iter.Key()...))
			continue
		}
		messages = append(messages, message)
		if count > 0 && len(messages) >= count {
			break
		}
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	if len(invalidKeys) > 0 && !p.owner.readOnly {
		batch := db.NewBatch()
		defer batch.Close()
		for _, key := range invalidKeys {
			if err := batch.Delete(key, p.owner.standaloneWriteOptions()); err != nil {
				return nil, err
			}
		}
		if err := batch.Commit(p.owner.standaloneWriteOptions()); err != nil {
			return nil, err
		}
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return messages, nil
}

func (p *pebbleNotifyQueueStore) Delete(messageIDs []int64) error {
	if len(messageIDs) == 0 {
		return nil
	}
	db, err := p.owner.localDB()
	if err != nil {
		return err
	}
	batch := db.NewBatch()
	defer batch.Close()

	for _, messageID := range messageIDs {
		if err := batch.Delete(v3key.EncodeNotifyQueueRowKey(uint64(messageID)), p.owner.standaloneWriteOptions()); err != nil {
			return err
		}
	}
	return batch.Commit(p.owner.standaloneWriteOptions())
}

func (p *pebbleNotifyQueueStore) DeleteCount(count int) error {
	if count <= 0 {
		return nil
	}
	db, err := p.owner.localDB()
	if err != nil {
		return err
	}
	lower, upper := v3key.MetaTableRange(v3key.ScopeLocal, v3key.TableMessageNotifyQueue)
	iter := db.NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	keys := make([][]byte, 0, count)
	for iter.First(); iter.Valid() && len(keys) < count; iter.Next() {
		keys = append(keys, append([]byte(nil), iter.Key()...))
	}
	if err := iter.Error(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return string(keys[i]) < string(keys[j])
	})
	batch := db.NewBatch()
	defer batch.Close()
	for _, key := range keys {
		if err := batch.Delete(key, p.owner.standaloneWriteOptions()); err != nil {
			return err
		}
	}
	return batch.Commit(p.owner.standaloneWriteOptions())
}
