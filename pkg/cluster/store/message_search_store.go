package store

import "github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"

type MessageSearchStore interface {
	SearchMessages(req wkdb.MessageSearchReq) ([]wkdb.Message, error)
	CountMessages() (int, error)
}

type LegacyMessageSearchStore struct {
	db wkdb.DB
}

func NewLegacyMessageSearchStore(db wkdb.DB) *LegacyMessageSearchStore {
	if db == nil {
		return nil
	}
	return &LegacyMessageSearchStore{db: db}
}

func (l *LegacyMessageSearchStore) SearchMessages(req wkdb.MessageSearchReq) ([]wkdb.Message, error) {
	return l.db.SearchMessages(req)
}

func (l *LegacyMessageSearchStore) CountMessages() (int, error) {
	return l.db.GetTotalMessageCount()
}
