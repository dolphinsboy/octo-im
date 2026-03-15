package v3

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/cockroachdb/pebble"
)

type pebbleMetaDB struct {
	owner *PebbleDB
}

var _ MetaDB = (*pebbleMetaDB)(nil)

func (p *pebbleMetaDB) SystemUIDs() SystemUIDStore {
	return &pebbleSystemUIDStore{owner: p.owner}
}

func (p *pebbleMetaDB) Testers() TesterStore {
	return &pebbleTesterStore{owner: p.owner}
}

func (p *pebbleMetaDB) Plugins() PluginStore {
	return &pebblePluginStore{owner: p.owner}
}

func (p *pebbleMetaDB) PluginUsers() PluginUserStore {
	return &pebblePluginUserStore{owner: p.owner}
}

type pebbleSystemUIDStore struct {
	owner *PebbleDB
}

var _ SystemUIDStore = (*pebbleSystemUIDStore)(nil)

func (p *pebbleSystemUIDStore) GetAll() ([]string, error) {
	db, err := p.owner.metaDB()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.MetaTableRange(v3key.ScopeMetaPrimary, v3key.TableSystemUID)
	iter := db.NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	uids := make([]string, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		if len(iter.Value()) > 0 {
			uids = append(uids, string(iter.Value()))
			continue
		}
		uid, err := v3key.ParseSystemUIDRowKey(iter.Key())
		if err != nil {
			return nil, err
		}
		uids = append(uids, uid)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.Strings(uids)
	if len(uids) == 0 {
		return nil, nil
	}
	return uids, nil
}

func (p *pebbleSystemUIDStore) PutAll(uids []string) error {
	if len(uids) == 0 {
		return nil
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return err
	}
	batch := db.NewBatch()
	defer batch.Close()

	for _, uid := range uids {
		if strings.TrimSpace(uid) == "" {
			continue
		}
		if err := batch.Set(v3key.EncodeSystemUIDRowKey(uid), []byte(uid), p.owner.standaloneWriteOptions()); err != nil {
			return err
		}
	}
	return batch.Commit(p.owner.standaloneWriteOptions())
}

func (p *pebbleSystemUIDStore) Delete(uids []string) error {
	if len(uids) == 0 {
		return nil
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return err
	}
	batch := db.NewBatch()
	defer batch.Close()

	for _, uid := range uids {
		if strings.TrimSpace(uid) == "" {
			continue
		}
		if err := batch.Delete(v3key.EncodeSystemUIDRowKey(uid), p.owner.standaloneWriteOptions()); err != nil {
			return err
		}
	}
	return batch.Commit(p.owner.standaloneWriteOptions())
}

type pebbleTesterStore struct {
	owner *PebbleDB
}

var _ TesterStore = (*pebbleTesterStore)(nil)

func (p *pebbleTesterStore) Get(no string) (wkdb.Tester, error) {
	if strings.TrimSpace(no) == "" {
		return wkdb.Tester{}, fmt.Errorf("tester no is required")
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return wkdb.Tester{}, err
	}
	value, closer, err := db.Get(v3key.EncodeTesterRowKey(no))
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.Tester{}, wkdb.ErrNotFound
		}
		return wkdb.Tester{}, err
	}
	defer closer.Close()
	return decodeTesterValue(value)
}

func (p *pebbleTesterStore) List() ([]wkdb.Tester, error) {
	db, err := p.owner.metaDB()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.MetaTableRange(v3key.ScopeMetaPrimary, v3key.TableTester)
	iter := db.NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	testers := make([]wkdb.Tester, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		tester, err := decodeTesterValue(iter.Value())
		if err != nil {
			return nil, err
		}
		testers = append(testers, cloneTester(tester))
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.SliceStable(testers, func(i, j int) bool {
		return strings.Compare(testers[i].No, testers[j].No) < 0
	})
	if len(testers) == 0 {
		return nil, nil
	}
	return testers, nil
}

func (p *pebbleTesterStore) Put(tester wkdb.Tester) error {
	if strings.TrimSpace(tester.No) == "" {
		return fmt.Errorf("tester no is required")
	}
	oldTester, err := p.Get(tester.No)
	if err != nil && err != wkdb.ErrNotFound {
		return err
	}
	now := p.owner.currentTime()
	if tester.CreatedAt == nil {
		if err == nil && oldTester.CreatedAt != nil {
			tester.CreatedAt = cloneTimePtr(oldTester.CreatedAt)
		} else {
			tester.CreatedAt = cloneTimePtr(&now)
		}
	}
	if tester.UpdatedAt == nil {
		tester.UpdatedAt = cloneTimePtr(&now)
	}
	tester.Id = hashString64(tester.No)

	value, err := encodeTesterValue(tester)
	if err != nil {
		return err
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return err
	}
	return db.Set(v3key.EncodeTesterRowKey(tester.No), value, p.owner.standaloneWriteOptions())
}

func (p *pebbleTesterStore) Delete(no string) error {
	if strings.TrimSpace(no) == "" {
		return nil
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return err
	}
	return db.Delete(v3key.EncodeTesterRowKey(no), p.owner.standaloneWriteOptions())
}

type pebblePluginStore struct {
	owner *PebbleDB
}

var _ PluginStore = (*pebblePluginStore)(nil)

func (p *pebblePluginStore) Get(no string) (wkdb.Plugin, error) {
	if strings.TrimSpace(no) == "" {
		return wkdb.Plugin{}, fmt.Errorf("plugin no is required")
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return wkdb.Plugin{}, err
	}
	value, closer, err := db.Get(v3key.EncodePluginRowKey(no))
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.Plugin{}, wkdb.ErrNotFound
		}
		return wkdb.Plugin{}, err
	}
	defer closer.Close()
	return decodePluginValue(value)
}

func (p *pebblePluginStore) List() ([]wkdb.Plugin, error) {
	db, err := p.owner.metaDB()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.MetaTableRange(v3key.ScopeMetaPrimary, v3key.TablePlugin)
	iter := db.NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	plugins := make([]wkdb.Plugin, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		plugin, err := decodePluginValue(iter.Value())
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, clonePlugin(plugin))
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.SliceStable(plugins, func(i, j int) bool {
		return strings.Compare(plugins[i].No, plugins[j].No) < 0
	})
	if len(plugins) == 0 {
		return nil, nil
	}
	return plugins, nil
}

func (p *pebblePluginStore) Put(plugin wkdb.Plugin) error {
	if strings.TrimSpace(plugin.No) == "" {
		return fmt.Errorf("plugin no is required")
	}
	oldPlugin, err := p.Get(plugin.No)
	if err != nil && err != wkdb.ErrNotFound {
		return err
	}
	now := p.owner.currentTime()
	if plugin.CreatedAt == nil {
		if err == nil && oldPlugin.CreatedAt != nil {
			plugin.CreatedAt = cloneTimePtr(oldPlugin.CreatedAt)
		} else {
			plugin.CreatedAt = cloneTimePtr(&now)
		}
	}
	if plugin.UpdatedAt == nil {
		plugin.UpdatedAt = cloneTimePtr(&now)
	}
	value, err := encodePluginValue(plugin)
	if err != nil {
		return err
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return err
	}
	return db.Set(v3key.EncodePluginRowKey(plugin.No), value, p.owner.standaloneWriteOptions())
}

func (p *pebblePluginStore) Delete(no string) error {
	if strings.TrimSpace(no) == "" {
		return nil
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return err
	}
	return db.Delete(v3key.EncodePluginRowKey(no), p.owner.standaloneWriteOptions())
}

func (p *pebblePluginStore) UpdateConfig(no string, config map[string]interface{}) error {
	plugin, err := p.Get(no)
	if err != nil {
		return err
	}
	plugin.Config = cloneConfigMap(config)
	now := p.owner.currentTime()
	plugin.UpdatedAt = cloneTimePtr(&now)
	return p.Put(plugin)
}

type pebblePluginUserStore struct {
	owner *PebbleDB
}

var _ PluginUserStore = (*pebblePluginUserStore)(nil)

func (p *pebblePluginUserStore) ListByPlugin(pluginNo string) ([]wkdb.PluginUser, error) {
	if strings.TrimSpace(pluginNo) == "" {
		return nil, nil
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.PluginUserPluginRange(pluginNo)
	iter := db.NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	pluginUsers := make([]wkdb.PluginUser, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		pluginUser, err := decodePluginUserValue(iter.Value())
		if err != nil {
			return nil, err
		}
		pluginUsers = append(pluginUsers, clonePluginUser(pluginUser))
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sortPluginUsers(pluginUsers)
	if len(pluginUsers) == 0 {
		return nil, nil
	}
	return pluginUsers, nil
}

func (p *pebblePluginUserStore) Search(req wkdb.SearchPluginUserReq) ([]wkdb.PluginUser, error) {
	switch {
	case strings.TrimSpace(req.PluginNo) != "" && strings.TrimSpace(req.Uid) != "":
		pluginUser, err := p.get(req.PluginNo, req.Uid)
		if err != nil {
			if err == wkdb.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return []wkdb.PluginUser{clonePluginUser(pluginUser)}, nil
	case strings.TrimSpace(req.PluginNo) != "":
		return p.ListByPlugin(req.PluginNo)
	case strings.TrimSpace(req.Uid) != "":
		return p.searchByUID(req.Uid)
	default:
		return p.listAll()
	}
}

func (p *pebblePluginUserStore) Put(pluginUsers []wkdb.PluginUser) error {
	if len(pluginUsers) == 0 {
		return nil
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return err
	}
	batch := db.NewBatch()
	defer batch.Close()

	now := p.owner.currentTime()
	for _, pluginUser := range pluginUsers {
		normalized, err := p.normalize(pluginUser, now)
		if err != nil {
			return err
		}
		rowKey := v3key.EncodePluginUserRowKey(normalized.PluginNo, normalized.Uid)
		value, err := encodePluginUserValue(normalized)
		if err != nil {
			return err
		}
		if err := batch.Set(rowKey, value, p.owner.standaloneWriteOptions()); err != nil {
			return err
		}
		if err := batch.Set(v3key.EncodePluginUserUIDSecondIndexKey(normalized.Uid, normalized.PluginNo), rowKey, p.owner.standaloneWriteOptions()); err != nil {
			return err
		}
	}
	return batch.Commit(p.owner.standaloneWriteOptions())
}

func (p *pebblePluginUserStore) Delete(pluginNo, uid string) error {
	if strings.TrimSpace(pluginNo) == "" || strings.TrimSpace(uid) == "" {
		return nil
	}
	db, err := p.owner.metaDB()
	if err != nil {
		return err
	}
	batch := db.NewBatch()
	defer batch.Close()

	if err := batch.Delete(v3key.EncodePluginUserRowKey(pluginNo, uid), p.owner.standaloneWriteOptions()); err != nil {
		return err
	}
	if err := batch.Delete(v3key.EncodePluginUserUIDSecondIndexKey(uid, pluginNo), p.owner.standaloneWriteOptions()); err != nil {
		return err
	}
	return batch.Commit(p.owner.standaloneWriteOptions())
}

func (p *pebblePluginUserStore) normalize(pluginUser wkdb.PluginUser, now time.Time) (wkdb.PluginUser, error) {
	if strings.TrimSpace(pluginUser.PluginNo) == "" {
		return wkdb.PluginUser{}, fmt.Errorf("plugin user plugin no is required")
	}
	if strings.TrimSpace(pluginUser.Uid) == "" {
		return wkdb.PluginUser{}, fmt.Errorf("plugin user uid is required")
	}
	oldPluginUser, err := p.get(pluginUser.PluginNo, pluginUser.Uid)
	if err != nil && err != wkdb.ErrNotFound {
		return wkdb.PluginUser{}, err
	}
	if pluginUser.CreatedAt == nil {
		if err == nil && oldPluginUser.CreatedAt != nil {
			pluginUser.CreatedAt = cloneTimePtr(oldPluginUser.CreatedAt)
		} else {
			pluginUser.CreatedAt = cloneTimePtr(&now)
		}
	}
	if pluginUser.UpdatedAt == nil {
		pluginUser.UpdatedAt = cloneTimePtr(&now)
	}
	return pluginUser, nil
}

func (p *pebblePluginUserStore) get(pluginNo, uid string) (wkdb.PluginUser, error) {
	db, err := p.owner.metaDB()
	if err != nil {
		return wkdb.PluginUser{}, err
	}
	value, closer, err := db.Get(v3key.EncodePluginUserRowKey(pluginNo, uid))
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.PluginUser{}, wkdb.ErrNotFound
		}
		return wkdb.PluginUser{}, err
	}
	defer closer.Close()
	return decodePluginUserValue(value)
}

func (p *pebblePluginUserStore) getByRowKey(rowKey []byte) (wkdb.PluginUser, error) {
	db, err := p.owner.metaDB()
	if err != nil {
		return wkdb.PluginUser{}, err
	}
	value, closer, err := db.Get(rowKey)
	if err != nil {
		if err == pebble.ErrNotFound {
			return wkdb.PluginUser{}, wkdb.ErrNotFound
		}
		return wkdb.PluginUser{}, err
	}
	defer closer.Close()
	return decodePluginUserValue(value)
}

func (p *pebblePluginUserStore) searchByUID(uid string) ([]wkdb.PluginUser, error) {
	db, err := p.owner.metaDB()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.PluginUserUIDSecondIndexRange(uid)
	iter := db.NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	pluginUsers := make([]wkdb.PluginUser, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		rowKey := append([]byte(nil), iter.Value()...)
		if len(rowKey) == 0 {
			indexUID, pluginNo, err := v3key.ParsePluginUserUIDSecondIndexKey(iter.Key())
			if err != nil {
				return nil, err
			}
			rowKey = v3key.EncodePluginUserRowKey(pluginNo, indexUID)
		}
		pluginUser, err := p.getByRowKey(rowKey)
		if err != nil {
			if err == wkdb.ErrNotFound {
				continue
			}
			return nil, err
		}
		pluginUsers = append(pluginUsers, clonePluginUser(pluginUser))
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sortPluginUsers(pluginUsers)
	if len(pluginUsers) == 0 {
		return nil, nil
	}
	return pluginUsers, nil
}

func (p *pebblePluginUserStore) listAll() ([]wkdb.PluginUser, error) {
	db, err := p.owner.metaDB()
	if err != nil {
		return nil, err
	}
	lower, upper := v3key.MetaTableRange(v3key.ScopeMetaPrimary, v3key.TablePluginUser)
	iter := db.NewIter(&pebble.IterOptions{
		LowerBound: lower,
		UpperBound: upper,
	})
	defer iter.Close()

	pluginUsers := make([]wkdb.PluginUser, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		pluginUser, err := decodePluginUserValue(iter.Value())
		if err != nil {
			return nil, err
		}
		pluginUsers = append(pluginUsers, clonePluginUser(pluginUser))
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sortPluginUsers(pluginUsers)
	if len(pluginUsers) == 0 {
		return nil, nil
	}
	return pluginUsers, nil
}

func encodeTesterValue(tester wkdb.Tester) ([]byte, error) {
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteUint64(tester.Id)
	enc.WriteString(tester.No)
	enc.WriteString(tester.Addr)
	enc.WriteUint64(timePtrToUnixNano(tester.CreatedAt))
	enc.WriteUint64(timePtrToUnixNano(tester.UpdatedAt))
	return enc.Bytes(), nil
}

func decodeTesterValue(data []byte) (wkdb.Tester, error) {
	dec := wkproto.NewDecoder(data)
	var (
		tester wkdb.Tester
		err    error
	)
	if tester.Id, err = dec.Uint64(); err != nil {
		return wkdb.Tester{}, err
	}
	if tester.No, err = dec.String(); err != nil {
		return wkdb.Tester{}, err
	}
	if tester.Addr, err = dec.String(); err != nil {
		return wkdb.Tester{}, err
	}
	createdAt, err := dec.Uint64()
	if err != nil {
		return wkdb.Tester{}, err
	}
	updatedAt, err := dec.Uint64()
	if err != nil {
		return wkdb.Tester{}, err
	}
	tester.CreatedAt = unixNanoPtr(createdAt)
	tester.UpdatedAt = unixNanoPtr(updatedAt)
	return tester, nil
}

func encodePluginValue(plugin wkdb.Plugin) ([]byte, error) {
	methodsBytes, err := json.Marshal(plugin.Methods)
	if err != nil {
		return nil, err
	}
	configBytes, err := json.Marshal(plugin.Config)
	if err != nil {
		return nil, err
	}
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteString(plugin.No)
	enc.WriteString(plugin.Name)
	enc.WriteBinary(plugin.ConfigTemplate)
	enc.WriteUint64(timePtrToUnixNano(plugin.CreatedAt))
	enc.WriteUint64(timePtrToUnixNano(plugin.UpdatedAt))
	enc.WriteUint32(uint32(plugin.Status))
	enc.WriteString(plugin.Version)
	enc.WriteBinary(methodsBytes)
	enc.WriteUint32(plugin.Priority)
	enc.WriteBinary(configBytes)
	return enc.Bytes(), nil
}

func decodePluginValue(data []byte) (wkdb.Plugin, error) {
	dec := wkproto.NewDecoder(data)
	var (
		plugin wkdb.Plugin
		err    error
	)
	if plugin.No, err = dec.String(); err != nil {
		return wkdb.Plugin{}, err
	}
	if plugin.Name, err = dec.String(); err != nil {
		return wkdb.Plugin{}, err
	}
	if plugin.ConfigTemplate, err = dec.Binary(); err != nil {
		return wkdb.Plugin{}, err
	}
	createdAt, err := dec.Uint64()
	if err != nil {
		return wkdb.Plugin{}, err
	}
	updatedAt, err := dec.Uint64()
	if err != nil {
		return wkdb.Plugin{}, err
	}
	status, err := dec.Uint32()
	if err != nil {
		return wkdb.Plugin{}, err
	}
	if plugin.Version, err = dec.String(); err != nil {
		return wkdb.Plugin{}, err
	}
	methodsBytes, err := dec.Binary()
	if err != nil {
		return wkdb.Plugin{}, err
	}
	if plugin.Priority, err = dec.Uint32(); err != nil {
		return wkdb.Plugin{}, err
	}
	configBytes, err := dec.Binary()
	if err != nil {
		return wkdb.Plugin{}, err
	}
	plugin.CreatedAt = unixNanoPtr(createdAt)
	plugin.UpdatedAt = unixNanoPtr(updatedAt)
	plugin.Status = wkdb.PluginStatus(status)
	if len(methodsBytes) > 0 {
		if err := json.Unmarshal(methodsBytes, &plugin.Methods); err != nil {
			return wkdb.Plugin{}, err
		}
	}
	if len(configBytes) > 0 && string(configBytes) != "null" {
		if err := json.Unmarshal(configBytes, &plugin.Config); err != nil {
			return wkdb.Plugin{}, err
		}
	}
	return plugin, nil
}

func encodePluginUserValue(pluginUser wkdb.PluginUser) ([]byte, error) {
	enc := wkproto.NewEncoder()
	defer enc.End()
	enc.WriteString(pluginUser.PluginNo)
	enc.WriteString(pluginUser.Uid)
	enc.WriteUint64(timePtrToUnixNano(pluginUser.CreatedAt))
	enc.WriteUint64(timePtrToUnixNano(pluginUser.UpdatedAt))
	return enc.Bytes(), nil
}

func decodePluginUserValue(data []byte) (wkdb.PluginUser, error) {
	dec := wkproto.NewDecoder(data)
	var (
		pluginUser wkdb.PluginUser
		err        error
	)
	if pluginUser.PluginNo, err = dec.String(); err != nil {
		return wkdb.PluginUser{}, err
	}
	if pluginUser.Uid, err = dec.String(); err != nil {
		return wkdb.PluginUser{}, err
	}
	createdAt, err := dec.Uint64()
	if err != nil {
		return wkdb.PluginUser{}, err
	}
	updatedAt, err := dec.Uint64()
	if err != nil {
		return wkdb.PluginUser{}, err
	}
	pluginUser.CreatedAt = unixNanoPtr(createdAt)
	pluginUser.UpdatedAt = unixNanoPtr(updatedAt)
	return pluginUser, nil
}

func cloneTester(tester wkdb.Tester) wkdb.Tester {
	tester.CreatedAt = cloneTimePtr(tester.CreatedAt)
	tester.UpdatedAt = cloneTimePtr(tester.UpdatedAt)
	return tester
}

func clonePlugin(plugin wkdb.Plugin) wkdb.Plugin {
	plugin.ConfigTemplate = append([]byte(nil), plugin.ConfigTemplate...)
	plugin.Methods = append([]string(nil), plugin.Methods...)
	plugin.Config = cloneConfigMap(plugin.Config)
	plugin.CreatedAt = cloneTimePtr(plugin.CreatedAt)
	plugin.UpdatedAt = cloneTimePtr(plugin.UpdatedAt)
	return plugin
}

func clonePluginUser(pluginUser wkdb.PluginUser) wkdb.PluginUser {
	pluginUser.CreatedAt = cloneTimePtr(pluginUser.CreatedAt)
	pluginUser.UpdatedAt = cloneTimePtr(pluginUser.UpdatedAt)
	return pluginUser
}

func cloneConfigMap(config map[string]interface{}) map[string]interface{} {
	if len(config) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(config))
	for key, value := range config {
		cloned[key] = value
	}
	return cloned
}

func sortPluginUsers(pluginUsers []wkdb.PluginUser) {
	sort.SliceStable(pluginUsers, func(i, j int) bool {
		left := timePtrToUnixNano(pluginUsers[i].CreatedAt)
		right := timePtrToUnixNano(pluginUsers[j].CreatedAt)
		if left == right {
			if pluginUsers[i].PluginNo == pluginUsers[j].PluginNo {
				return strings.Compare(pluginUsers[i].Uid, pluginUsers[j].Uid) < 0
			}
			return strings.Compare(pluginUsers[i].PluginNo, pluginUsers[j].PluginNo) < 0
		}
		return left > right
	})
}

func hashString64(value string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(value))
	return h.Sum64()
}

func timePtrToUnixNano(t *time.Time) uint64 {
	if t == nil {
		return 0
	}
	return uint64(t.UnixNano())
}
