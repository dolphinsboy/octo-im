package store

import (
	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	"github.com/WuKongIM/WuKongIM/pkg/wklog"
	"github.com/lni/goutils/syncutil"
)

type Store struct {
	opts *Options
	wklog.Log

	wdb                 wkdb.DB
	userDeviceStore     UserDeviceStore
	conversationStore   ConversationStore
	channelStateStore   ChannelStateStore
	messageStore        MessageStore
	eventStore          MessageEventStore
	metaStore           MetaStore
	metaCommandProposer MetaCommandProposer
	adminSearchStore    AdminSearchStore

	channelCfgCh chan *channelCfgReq
	stopper      *syncutil.Stopper
}

func New(opts *Options) *Store {
	userDeviceStore := opts.UserDeviceStore
	if userDeviceStore == nil {
		if hybrid, ok := opts.DB.(*HybridDB); ok {
			userDeviceStore = NewSlotUserDeviceStore(hybrid.slotDB, hybrid.routeSlot)
		}
	}
	if userDeviceStore == nil && opts.DB != nil {
		userDeviceStore = opts.DB
	}
	conversationStore := opts.ConversationStore
	if conversationStore == nil {
		if hybrid, ok := opts.DB.(*HybridDB); ok {
			conversationStore = NewSlotConversationStore(hybrid.slotDB, hybrid.slotCount, hybrid.routeSlot, func() uint64 {
				return hybrid.NextPrimaryKey()
			})
		}
	}
	if conversationStore == nil && opts.DB != nil {
		conversationStore = opts.DB
	}
	channelStateStore := opts.ChannelStateStore
	if channelStateStore == nil {
		if hybrid, ok := opts.DB.(*HybridDB); ok {
			channelStateStore = NewSlotChannelStateStore(hybrid.slotDB, hybrid.routeSlot)
		}
	}
	if channelStateStore == nil && opts.DB != nil {
		channelStateStore = opts.DB
	}
	messageStore := opts.MessageStore
	if messageStore == nil && opts.DB != nil {
		messageStore = NewLegacyMessageStore(opts.DB)
	}
	eventStore := opts.MessageEventStore
	if eventStore == nil && opts.DB != nil {
		eventStore = opts.DB
	}
	metaStore := opts.MetaStore
	if metaStore == nil && opts.DB != nil {
		metaStore = opts.DB
	}
	metaCommandProposer := opts.MetaCommandProposer
	if metaCommandProposer == nil && opts.Slot != nil {
		metaCommandProposer = NewSlotZeroMetaCommandProposer(opts.Slot, 0)
	}
	adminSearchStore := opts.AdminSearchStore
	if adminSearchStore == nil {
		if hybrid, ok := opts.DB.(*HybridDB); ok {
			adminSearchStore = NewSlotAdminSearchStore(hybrid.slotDB, hybrid.slotCount, hybrid.routeSlot)
		}
	}
	s := &Store{
		opts:                opts,
		Log:                 wklog.NewWKLog("store"),
		wdb:                 opts.DB,
		userDeviceStore:     userDeviceStore,
		conversationStore:   conversationStore,
		channelStateStore:   channelStateStore,
		messageStore:        messageStore,
		eventStore:          eventStore,
		metaStore:           metaStore,
		metaCommandProposer: metaCommandProposer,
		adminSearchStore:    adminSearchStore,
		channelCfgCh:        make(chan *channelCfgReq, 2048),
		stopper:             syncutil.NewStopper(),
	}

	return s
}

func (s *Store) NextPrimaryKey() uint64 {
	return s.wdb.NextPrimaryKey()
}

// GetShardNum 获取数据库分片数量
func (s *Store) GetShardNum() int {
	return s.wdb.GetShardNum()
}

// GetChannelShardIndex 获取频道所在的分片索引
func (s *Store) GetChannelShardIndex(channelId string, channelType uint8) uint32 {
	return s.wdb.GetChannelShardIndex(channelId, channelType)
}

func (s *Store) Start() error {
	for i := 0; i < 50; i++ {
		go s.loopSaveChannelClusterConfig()
	}
	// s.stopper.RunWorker(s.loopSaveChannelClusterConfig)
	return nil
}

func (s *Store) Stop() {
	s.stopper.Stop()
}

type channelCfgReq struct {
	cfg   wkdb.ChannelClusterConfig
	errCh chan error
}
