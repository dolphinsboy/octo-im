package v3

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/cockroachdb/pebble"
)

type pebbleUserStore struct {
	slotID  uint32
	buckets *PebbleBucketManager
	now     func() time.Time
}

var _ UserStore = (*pebbleUserStore)(nil)

func (p *pebbleUserStore) Get(uid string) (wkdb.User, error) {
	bucket, err := p.bucket()
	if err != nil {
		return wkdb.User{}, err
	}
	value, closer, err := bucket.DB().Get(v3key.EncodeUserRowKey(p.slotID, uid))
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.User{}, wkdb.ErrNotFound
		}
		return wkdb.User{}, err
	}
	defer closer.Close()
	return decodeUserValue(value)
}

func (p *pebbleUserStore) Exists(uid string) (bool, error) {
	_, err := p.Get(uid)
	if err == nil {
		return true, nil
	}
	if err == wkdb.ErrNotFound {
		return false, nil
	}
	return false, err
}

func (p *pebbleUserStore) Put(user wkdb.User) error {
	if user.Uid == "" {
		return wkdb.ErrInvalidUserId
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	oldUser, err := p.Get(user.Uid)
	if err != nil && err != wkdb.ErrNotFound {
		return err
	}
	existed := err == nil
	now := p.currentTime()
	if existed && oldUser.CreatedAt != nil {
		user.CreatedAt = cloneTimePtr(oldUser.CreatedAt)
	} else if user.CreatedAt == nil {
		user.CreatedAt = cloneTimePtr(&now)
	}
	if user.UpdatedAt == nil {
		user.UpdatedAt = cloneTimePtr(&now)
	}

	batch := bucket.DB().NewBatch()
	defer batch.Close()
	if existed && oldUser.CreatedAt != nil {
		if err := batch.Delete(v3key.EncodeUserCreatedAtSecondIndexKey(p.slotID, uint64(oldUser.CreatedAt.UnixNano()), oldUser.Uid), p.writeOptions()); err != nil {
			return err
		}
	}
	value, err := encodeUserValue(user)
	if err != nil {
		return err
	}
	if err := batch.Set(v3key.EncodeUserRowKey(p.slotID, user.Uid), value, p.writeOptions()); err != nil {
		return err
	}
	if err := batch.Set(v3key.EncodeUserCreatedAtSecondIndexKey(p.slotID, uint64(user.CreatedAt.UnixNano()), user.Uid), []byte(user.Uid), p.writeOptions()); err != nil {
		return err
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleUserStore) Delete(uid string) error {
	user, err := p.Get(uid)
	if err != nil {
		if err == wkdb.ErrNotFound {
			return nil
		}
		return err
	}
	bucket, err := p.bucket()
	if err != nil {
		return err
	}
	batch := bucket.DB().NewBatch()
	defer batch.Close()
	if err := batch.Delete(v3key.EncodeUserRowKey(p.slotID, uid), p.writeOptions()); err != nil {
		return err
	}
	if user.CreatedAt != nil {
		if err := batch.Delete(v3key.EncodeUserCreatedAtSecondIndexKey(p.slotID, uint64(user.CreatedAt.UnixNano()), uid), p.writeOptions()); err != nil {
			return err
		}
	}
	return batch.Commit(p.writeOptions())
}

func (p *pebbleUserStore) Search(req wkdb.UserSearchReq) ([]wkdb.User, error) {
	if req.Uid != "" {
		user, err := p.Get(req.Uid)
		if err != nil {
			return nil, err
		}
		return []wkdb.User{user}, nil
	}
	bucket, err := p.bucket()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.SlotTableRange(p.slotID, v3key.ScopeSlotPrimary, v3key.TableUser)
	iter := bucket.DB().NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	users := make([]wkdb.User, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		user, err := decodeUserValue(iter.Value())
		if err != nil {
			return nil, err
		}
		if !matchUserSearch(user, req) {
			continue
		}
		users = append(users, user)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.SliceStable(users, func(i, j int) bool {
		left := userSortTime(users[i])
		right := userSortTime(users[j])
		if left == right {
			return strings.Compare(users[i].Uid, users[j].Uid) < 0
		}
		return left > right
	})
	return limitUsers(users, req.Limit, req.Pre), nil
}

func (p *pebbleUserStore) bucket() (*PebbleBucket, error) {
	return p.buckets.Bucket(p.slotID)
}

func (p *pebbleUserStore) currentTime() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

func (p *pebbleUserStore) writeOptions() *pebble.WriteOptions {
	bucket, _ := p.bucket()
	if bucket == nil {
		return pebble.Sync
	}
	return bucket.SnapshotStore().writeOptions
}

func encodeUserValue(user wkdb.User) ([]byte, error) {
	if user.Uid == "" {
		return nil, wkdb.ErrInvalidUserId
	}
	if user.CreatedAt == nil || user.UpdatedAt == nil {
		return nil, fmt.Errorf("user timestamps are required")
	}
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteUint64(user.Id)
	enc.WriteString(user.Uid)
	enc.WriteUint32(user.DeviceCount)
	enc.WriteUint32(user.OnlineDeviceCount)
	enc.WriteUint32(user.ConnCount)
	enc.WriteUint64(user.SendMsgCount)
	enc.WriteUint64(user.RecvMsgCount)
	enc.WriteUint64(user.SendMsgBytes)
	enc.WriteUint64(user.RecvMsgBytes)
	enc.WriteString(user.PluginNo)
	enc.WriteUint64(uint64(user.CreatedAt.UnixNano()))
	enc.WriteUint64(uint64(user.UpdatedAt.UnixNano()))
	return enc.Bytes(), nil
}

func decodeUserValue(data []byte) (wkdb.User, error) {
	dec := wkproto.NewDecoder(data)
	var (
		user wkdb.User
		err  error
	)
	if user.Id, err = dec.Uint64(); err != nil {
		return wkdb.User{}, err
	}
	if user.Uid, err = dec.String(); err != nil {
		return wkdb.User{}, err
	}
	if user.DeviceCount, err = dec.Uint32(); err != nil {
		return wkdb.User{}, err
	}
	if user.OnlineDeviceCount, err = dec.Uint32(); err != nil {
		return wkdb.User{}, err
	}
	if user.ConnCount, err = dec.Uint32(); err != nil {
		return wkdb.User{}, err
	}
	if user.SendMsgCount, err = dec.Uint64(); err != nil {
		return wkdb.User{}, err
	}
	if user.RecvMsgCount, err = dec.Uint64(); err != nil {
		return wkdb.User{}, err
	}
	if user.SendMsgBytes, err = dec.Uint64(); err != nil {
		return wkdb.User{}, err
	}
	if user.RecvMsgBytes, err = dec.Uint64(); err != nil {
		return wkdb.User{}, err
	}
	if user.PluginNo, err = dec.String(); err != nil {
		return wkdb.User{}, err
	}
	createdAt, err := dec.Uint64()
	if err != nil {
		return wkdb.User{}, err
	}
	updatedAt, err := dec.Uint64()
	if err != nil {
		return wkdb.User{}, err
	}
	user.CreatedAt = unixNanoPtr(createdAt)
	user.UpdatedAt = unixNanoPtr(updatedAt)
	return user, nil
}

func matchUserSearch(user wkdb.User, req wkdb.UserSearchReq) bool {
	ts := userSortTime(user)
	if req.OffsetCreatedAt <= 0 {
		return true
	}
	if req.Pre {
		return ts > req.OffsetCreatedAt
	}
	return ts < req.OffsetCreatedAt
}

func userSortTime(user wkdb.User) int64 {
	if user.CreatedAt != nil {
		return user.CreatedAt.UnixNano()
	}
	return 0
}

func limitUsers(users []wkdb.User, limit int, pre bool) []wkdb.User {
	if len(users) == 0 {
		return nil
	}
	if limit <= 0 {
		out := make([]wkdb.User, 0, len(users))
		for _, user := range users {
			out = append(out, cloneUser(user))
		}
		return out
	}
	if len(users) > limit {
		if pre {
			users = users[len(users)-limit:]
		} else {
			users = users[:limit]
		}
	}

	out := make([]wkdb.User, 0, len(users))
	for _, user := range users {
		out = append(out, cloneUser(user))
	}
	return out
}

func cloneUser(user wkdb.User) wkdb.User {
	user.CreatedAt = cloneTimePtr(user.CreatedAt)
	user.UpdatedAt = cloneTimePtr(user.UpdatedAt)
	return user
}
