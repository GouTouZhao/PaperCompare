package snowflake

import (
	"errors"
	"net"
	"sync"
	"time"
)

func NewPodIPHashSnowflake() (*Snowflake, error) {
	nodeID := getPodIPHashBits()
	return NewSnowflake(nodeID)
}

func getPodIPHashBits() int64 {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return 1
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip := ipnet.IP.To4(); ip != nil {
				return int64((uint16(ip[2])<<8 | uint16(ip[3])) & 1023)
			}
		}
	}
	return 1
}

type Snowflake struct {
	mu                 sync.Mutex
	epoch              int64
	nodeID             int64
	sequence           int64
	nodeIDBits         uint
	sequenceBits       uint
	nodeIDShift        uint
	timestampLeftShift uint
	sequenceMask       int64
	lastTimestamp      int64
}

func NewSnowflake(nodeID int64) (*Snowflake, error) {
	s := &Snowflake{
		epoch:        1609459200000,
		nodeID:       nodeID,
		nodeIDBits:   10,
		sequenceBits: 12,
	}

	maxNodeID := int64(-1 ^ (-1 << s.nodeIDBits))
	if nodeID < 0 || nodeID > maxNodeID {
		return nil, errors.New("node ID exceeds limit")
	}

	s.nodeIDShift = s.sequenceBits
	s.timestampLeftShift = s.sequenceBits + s.nodeIDBits
	s.sequenceMask = -1 ^ (-1 << s.sequenceBits)

	return s, nil
}

func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixNano() / 1e6

	if now < s.lastTimestamp {
		now = s.lastTimestamp
	}

	if now == s.lastTimestamp {
		s.sequence = (s.sequence + 1) & s.sequenceMask
		if s.sequence == 0 {
			for now <= s.lastTimestamp {
				now = time.Now().UnixNano() / 1e6
			}
		}
	} else {
		s.sequence = 0
	}

	s.lastTimestamp = now

	id := ((now - s.epoch) << s.timestampLeftShift) |
		(s.nodeID << s.nodeIDShift) |
		s.sequence

	return id, nil
}

