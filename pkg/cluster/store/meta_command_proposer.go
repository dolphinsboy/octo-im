package store

import "github.com/WuKongIM/WuKongIM/pkg/cluster/icluster"

type MetaCommandProposer interface {
	ProposeMetaCommandUntilApplied(data []byte) error
}

type SlotZeroMetaCommandProposer struct {
	slot   icluster.Slot
	slotID uint32
}

func NewSlotZeroMetaCommandProposer(slot icluster.Slot, slotID uint32) *SlotZeroMetaCommandProposer {
	if slot == nil {
		return nil
	}
	return &SlotZeroMetaCommandProposer{
		slot:   slot,
		slotID: slotID,
	}
}

func (s *SlotZeroMetaCommandProposer) ProposeMetaCommandUntilApplied(data []byte) error {
	_, err := s.slot.ProposeUntilApplied(s.slotID, data)
	return err
}
