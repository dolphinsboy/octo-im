package store

import (
	"sync"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/WuKongIM/WuKongIM/pkg/wklog"
	"github.com/lni/goutils/syncutil"
)

type Store struct {
	opts *Options
	wklog.Log

	primaryKeyAllocator       PrimaryKeyAllocator
	userDeviceStore           UserDeviceStore
	conversationStore         ConversationStore
	channelStateStore         ChannelStateStore
	channelClusterConfigStore ChannelClusterConfigStore
	messageQueryStore         MessageQueryStore
	messageIndexStore         MessageIndexStore
	messageSearchStore        MessageSearchStore
	notifyQueueStore          NotifyQueueStore
	eventStore                MessageEventStore
	metaStore                 MetaStore
	metaCommandProposer       MetaCommandProposer
	adminSearchStore          AdminSearchStore

	runtimeMu sync.RWMutex
	runtime   *storeRuntime
}

type storeRuntime struct {
	channelCfgCh chan *channelCfgReq
	stopper      *syncutil.Stopper
}

func New(opts *Options) *Store {
	primaryKeyAllocator := opts.PrimaryKeyAllocator
	userDeviceStore := opts.UserDeviceStore
	conversationStore := opts.ConversationStore
	channelStateStore := opts.ChannelStateStore
	channelClusterConfigStore := opts.ChannelClusterConfigStore
	messageQueryStore := opts.MessageQueryStore
	messageIndexStore := opts.MessageIndexStore
	messageSearchStore := opts.MessageSearchStore
	notifyQueueStore := opts.NotifyQueueStore
	eventStore := opts.MessageEventStore
	metaStore := opts.MetaStore
	metaCommandProposer := opts.MetaCommandProposer
	if metaCommandProposer == nil && opts.Slot != nil {
		metaCommandProposer = NewSlotZeroMetaCommandProposer(opts.Slot, 0)
	}
	adminSearchStore := opts.AdminSearchStore
	s := &Store{
		opts:                      opts,
		Log:                       wklog.NewWKLog("store"),
		primaryKeyAllocator:       primaryKeyAllocator,
		userDeviceStore:           userDeviceStore,
		conversationStore:         conversationStore,
		channelStateStore:         channelStateStore,
		channelClusterConfigStore: channelClusterConfigStore,
		messageQueryStore:         messageQueryStore,
		messageIndexStore:         messageIndexStore,
		messageSearchStore:        messageSearchStore,
		notifyQueueStore:          notifyQueueStore,
		eventStore:                eventStore,
		metaStore:                 metaStore,
		metaCommandProposer:       metaCommandProposer,
		adminSearchStore:          adminSearchStore,
	}

	return s
}

func (s *Store) nextPrimaryKey() uint64 {
	if s.primaryKeyAllocator == nil {
		return 0
	}
	return s.primaryKeyAllocator.NextPrimaryKey()
}

func (s *Store) Start() error {
	s.runtimeMu.RLock()
	if s.runtime != nil {
		s.runtimeMu.RUnlock()
		return nil
	}
	s.runtimeMu.RUnlock()

	lifecycle := s.storageLifecycle()
	if lifecycle != nil {
		if err := lifecycle.Open(); err != nil {
			return err
		}
	}

	rt := &storeRuntime{
		channelCfgCh: make(chan *channelCfgReq, 2048),
		stopper:      syncutil.NewStopper(),
	}

	s.runtimeMu.Lock()
	if s.runtime != nil {
		s.runtimeMu.Unlock()
		return nil
	}
	s.runtime = rt
	s.runtimeMu.Unlock()

	for i := 0; i < 50; i++ {
		rt.stopper.RunWorker(func() {
			s.loopSaveChannelClusterConfig(rt)
		})
	}
	return nil
}

func (s *Store) Stop() {
	s.runtimeMu.Lock()
	rt := s.runtime
	s.runtime = nil
	s.runtimeMu.Unlock()

	if rt != nil {
		rt.stopper.Stop()
	}
	if rt != nil {
		if lifecycle := s.storageLifecycle(); lifecycle != nil {
			_ = lifecycle.Close()
		}
	}
}

func (s *Store) runtimeSnapshot() *storeRuntime {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.runtime
}

func (s *Store) storageLifecycle() StorageLifecycle {
	return s.opts.Lifecycle
}

type channelCfgReq struct {
	cfg   wkdb.ChannelClusterConfig
	errCh chan error
}
