package cmd

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	wkinspect "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/inspect"
	"github.com/WuKongIM/WuKongIM/pkg/wkutil"
)

type slotFlags struct {
	selector string
}

type slotInterval struct {
	start uint32
	end   uint32
}

type slotSelector struct {
	all       bool
	intervals []slotInterval
}

type slotRouteHint struct {
	key   string
	label string
}

type slotScopedValue[T any] struct {
	SlotID   uint32 `json:"slot_id"`
	BucketID uint32 `json:"bucket_id"`
	Data     T      `json:"data"`
}

func (d *dbCMD) resolveSelectedSlots(ctx context.Context, inspector *wkinspect.Inspector, opts *dbCommandOptions, flags slotFlags, hint slotRouteHint) ([]uint32, error) {
	if strings.TrimSpace(flags.selector) == "" {
		if strings.TrimSpace(hint.key) != "" {
			slotID, err := d.inferSlotFromKey(opts, hint.key)
			if err != nil {
				return nil, fmt.Errorf("infer slot from %s: %w", hint.label, err)
			}
			return []uint32{slotID}, nil
		}
		return nil, fmt.Errorf("slot selector is required; provide --slot or a routable key")
	}
	selector, err := parseSlotSelector(flags.selector)
	if err != nil {
		return nil, err
	}
	if selector.all {
		return inspector.ExistingSlots(ctx)
	}
	seen := make(map[uint32]struct{})
	slots := make([]uint32, 0)
	for _, interval := range selector.intervals {
		for slotID := interval.start; slotID <= interval.end; slotID++ {
			if _, ok := seen[slotID]; ok {
				continue
			}
			seen[slotID] = struct{}{}
			slots = append(slots, slotID)
		}
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	return slots, nil
}

func (d *dbCMD) inferSlotFromKey(opts *dbCommandOptions, key string) (uint32, error) {
	slotCount, err := d.resolveSlotCount(opts)
	if err != nil {
		return 0, err
	}
	return wkutil.GetSlotNum(slotCount, key), nil
}

func (d *dbCMD) resolveSlotCount(opts *dbCommandOptions) (int, error) {
	if opts != nil && opts.slotCount > 0 {
		return opts.slotCount, nil
	}
	if serverOpts != nil && serverOpts.Cluster.SlotCount > 0 {
		return serverOpts.Cluster.SlotCount, nil
	}
	return 0, fmt.Errorf("slot count is required for automatic routing; provide --slot-count or configure cluster.slotCount")
}

func parseSlotSelector(raw string) (slotSelector, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return slotSelector{}, fmt.Errorf("slot selector is required")
	}
	if strings.EqualFold(raw, "all") || raw == "*" {
		return slotSelector{all: true}, nil
	}
	parts := strings.Split(raw, ",")
	selector := slotSelector{intervals: make([]slotInterval, 0, len(parts))}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return slotSelector{}, fmt.Errorf("invalid slot selector: %q", raw)
		}
		if strings.Contains(part, "-") {
			bounds := strings.Split(part, "-")
			if len(bounds) != 2 {
				return slotSelector{}, fmt.Errorf("invalid slot range: %q", part)
			}
			start, err := parseSlotUint(bounds[0])
			if err != nil {
				return slotSelector{}, fmt.Errorf("invalid slot range start %q: %w", bounds[0], err)
			}
			end, err := parseSlotUint(bounds[1])
			if err != nil {
				return slotSelector{}, fmt.Errorf("invalid slot range end %q: %w", bounds[1], err)
			}
			if start > end {
				return slotSelector{}, fmt.Errorf("invalid slot range %q: start is greater than end", part)
			}
			selector.intervals = append(selector.intervals, slotInterval{start: start, end: end})
			continue
		}
		slotID, err := parseSlotUint(part)
		if err != nil {
			return slotSelector{}, fmt.Errorf("invalid slot id %q: %w", part, err)
		}
		selector.intervals = append(selector.intervals, slotInterval{start: slotID, end: slotID})
	}
	return selector, nil
}

func parseSlotUint(raw string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("slot is empty")
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(value), nil
}

func collectSlotValues[T any](slots []uint32, inspector *wkinspect.Inspector, limit int, fetch func(slotID uint32, currentLimit int) ([]T, error)) ([]slotScopedValue[T], error) {
	items := make([]slotScopedValue[T], 0)
	remaining := limit
	for _, slotID := range slots {
		currentLimit := 0
		if limit > 0 {
			if remaining <= 0 {
				break
			}
			currentLimit = remaining
		}
		values, err := fetch(slotID, currentLimit)
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			items = append(items, slotScopedValue[T]{
				SlotID:   slotID,
				BucketID: inspector.BucketForSlot(slotID),
				Data:     value,
			})
		}
		if limit > 0 {
			remaining -= len(values)
		}
	}
	return items, nil
}

func findSlotValues[T any](slots []uint32, inspector *wkinspect.Inspector, fetch func(slotID uint32) (T, bool, error)) ([]slotScopedValue[T], error) {
	items := make([]slotScopedValue[T], 0)
	for _, slotID := range slots {
		value, found, err := fetch(slotID)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		items = append(items, slotScopedValue[T]{
			SlotID:   slotID,
			BucketID: inspector.BucketForSlot(slotID),
			Data:     value,
		})
	}
	return items, nil
}
