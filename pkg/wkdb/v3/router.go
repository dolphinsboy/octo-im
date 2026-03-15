package v3

import "fmt"

type BucketRouter interface {
	BucketCount() uint32
	BucketForSlot(slotID uint32) uint32
}

type StaticBucketRouter struct {
	bucketCount uint32
}

func NewStaticBucketRouter(bucketCount uint32) (*StaticBucketRouter, error) {
	if bucketCount == 0 {
		return nil, fmt.Errorf("bucket count must be greater than zero")
	}
	return &StaticBucketRouter{bucketCount: bucketCount}, nil
}

func (s *StaticBucketRouter) BucketCount() uint32 {
	return s.bucketCount
}

func (s *StaticBucketRouter) BucketForSlot(slotID uint32) uint32 {
	return slotID % s.bucketCount
}
