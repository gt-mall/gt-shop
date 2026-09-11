package bizno

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Business prefix definitions
// D = Order / 订单
// P = Payment / 支付
// R = Refund / 退款
// A = AfterSale / 售后
// S = Shipment / 发货
const (
	BizPrefixOrder     = "D"
	BizPrefixPayment   = "P"
	BizPrefixRefund    = "R"
	BizPrefixAfterSale = "A"
	BizPrefixShipment  = "S"
)

const (
	// CrockfordBase32Alphabet defines the 32 characters in Crockford's Base32.
	// Characters 'I', 'L', 'O', 'U' are intentionally omitted to avoid visual confusion.
	CrockfordBase32Alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

	// MaxWorkerID is the maximum allowed worker ID (0 ~ 15, 4 bits).
	MaxWorkerID = 15

	// MaxSequence is the maximum sequence value per millisecond (0 ~ 255, 8 bits).
	MaxSequence = 255

	// DayMillisecondsMask masks the 28 bits used for milliseconds elapsed since 00:00:00.000 of the day.
	DayMillisecondsMask = 0x0FFFFFFF
)

var (
	ErrInvalidWorkerID = errors.New("bizno: worker ID must be between 0 and 15")
	ErrInvalidPrefix   = errors.New("bizno: invalid business prefix")
	ErrClockBackwards  = errors.New("business number clock moved backwards")
)

// Clock abstracts time querying and sleeping for deterministic testing.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// RealClock implements Clock using the standard time package.
type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

func (RealClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// FakeClock provides an in-memory controllable clock for unit testing.
type FakeClock struct {
	mu      sync.Mutex
	current time.Time
}

// NewFakeClock creates a FakeClock initialized to the given time.
func NewFakeClock(initial time.Time) *FakeClock {
	return &FakeClock{current: initial}
}

// Now returns the current simulated time.
func (f *FakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.current
}

// Sleep advances the simulated time by d immediately without real delay.
func (f *FakeClock) Sleep(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current = f.current.Add(d)
}

// Set sets the simulated time to t.
func (f *FakeClock) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current = t
}

// Add adds duration d to the simulated time.
func (f *FakeClock) Add(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current = f.current.Add(d)
}

// Generator generates unique business numbers using a 40-bit Daily Snowflake:
// - 28 bits: milliseconds elapsed since 00:00:00.000 of current day (dayMilliseconds)
// -  4 bits: worker ID (0 ~ 15)
// -  8 bits: sequence (0 ~ 255)
// The 40-bit value is encoded into 8 Crockford Base32 characters.
// Final format: Prefix (1 char) + Date YYMMDD (6 chars) + Code (8 chars) = 15 chars.
type Generator struct {
	mu            sync.Mutex
	workerID      uint8
	lastUnixMilli int64
	sequence      uint16
	clock         Clock
}

// Option configures a Generator.
type Option func(*Generator)

// WithClock specifies a custom Clock (useful for testing).
func WithClock(clock Clock) Option {
	return func(g *Generator) {
		if clock != nil {
			g.clock = clock
		}
	}
}

// NewGenerator creates a new business number generator for the given worker ID (0 ~ 15).
func NewGenerator(workerID uint8, opts ...Option) (*Generator, error) {
	if workerID > MaxWorkerID {
		return nil, ErrInvalidWorkerID
	}
	g := &Generator{
		workerID:      workerID,
		lastUnixMilli: -1,
		clock:         RealClock{},
	}
	for _, opt := range opts {
		opt(g)
	}
	return g, nil
}

// WorkerID returns the configured worker ID.
func (g *Generator) WorkerID() uint8 {
	return g.workerID
}

// IsValidPrefix checks if the given prefix is among the allowed business prefixes.
func IsValidPrefix(prefix string) bool {
	switch prefix {
	case BizPrefixOrder, BizPrefixPayment, BizPrefixRefund, BizPrefixAfterSale, BizPrefixShipment:
		return true
	default:
		return false
	}
}

// Next generates the next unique 15-character business number for the specified prefix.
// It is thread-safe and guarantees monotonically increasing order per instance under normal clock conditions.
func (g *Generator) Next(prefix string) (string, error) {
	if !IsValidPrefix(prefix) {
		return "", ErrInvalidPrefix
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.clock.Now()
	currentUnixMilli := now.UnixMilli()

	if currentUnixMilli < g.lastUnixMilli {
		return "", ErrClockBackwards
	}

	if currentUnixMilli == g.lastUnixMilli {
		g.sequence++
		if g.sequence > MaxSequence {
			for currentUnixMilli <= g.lastUnixMilli {
				g.clock.Sleep(time.Millisecond)
				now = g.clock.Now()
				currentUnixMilli = now.UnixMilli()
				if currentUnixMilli < g.lastUnixMilli {
					return "", ErrClockBackwards
				}
			}
			g.lastUnixMilli = currentUnixMilli
			g.sequence = 0
		}
	} else {
		g.lastUnixMilli = currentUnixMilli
		g.sequence = 0
	}

	// Calculate milliseconds elapsed since 00:00:00.000 of the day in current location
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayMillis := now.Sub(startOfDay).Milliseconds()
	if dayMillis < 0 {
		dayMillis = 0
	}

	// 40 bits = [28 bit dayMillis] [4 bit worker] [8 bit sequence]
	rawID := (uint64(dayMillis&DayMillisecondsMask) << 12) |
		(uint64(g.workerID&0x0F) << 8) |
		uint64(g.sequence&MaxSequence)

	encoded := EncodeCrockfordBase32(rawID)
	dayStr := now.Format("060102")

	var buf [15]byte
	buf[0] = prefix[0]
	copy(buf[1:7], dayStr)
	copy(buf[7:15], encoded)

	return string(buf[:]), nil
}

// EncodeCrockfordBase32 encodes a 40-bit unsigned integer into an 8-character Crockford Base32 string.
// If the value requires fewer than 8 characters, it is zero-padded on the left.
func EncodeCrockfordBase32(val uint64) string {
	var buf [8]byte
	buf[0] = CrockfordBase32Alphabet[(val>>35)&0x1F]
	buf[1] = CrockfordBase32Alphabet[(val>>30)&0x1F]
	buf[2] = CrockfordBase32Alphabet[(val>>25)&0x1F]
	buf[3] = CrockfordBase32Alphabet[(val>>20)&0x1F]
	buf[4] = CrockfordBase32Alphabet[(val>>15)&0x1F]
	buf[5] = CrockfordBase32Alphabet[(val>>10)&0x1F]
	buf[6] = CrockfordBase32Alphabet[(val>>5)&0x1F]
	buf[7] = CrockfordBase32Alphabet[val&0x1F]
	return string(buf[:])
}

// DecodeCrockfordBase32 decodes an 8-character Crockford Base32 string into a 40-bit unsigned integer.
func DecodeCrockfordBase32(s string) (uint64, error) {
	if len(s) != 8 {
		return 0, fmt.Errorf("invalid Crockford Base32 string length: expected 8, got %d", len(s))
	}
	var val uint64
	for i := 0; i < 8; i++ {
		c := s[i]
		idx := -1
		for j := 0; j < len(CrockfordBase32Alphabet); j++ {
			if CrockfordBase32Alphabet[j] == c {
				idx = j
				break
			}
		}
		if idx == -1 {
			return 0, fmt.Errorf("invalid Crockford Base32 character: %q", c)
		}
		val = (val << 5) | uint64(idx)
	}
	return val, nil
}

// DeconstructBizNo decomposes a valid 15-character business number into its components:
// prefix, date string (YYMMDD), dayMillis (0 ~ 86399999), workerID (0 ~ 15), sequence (0 ~ 255).
func DeconstructBizNo(bizNo string) (prefix string, dateStr string, dayMillis int64, workerID uint8, sequence uint8, err error) {
	if len(bizNo) != 15 {
		return "", "", 0, 0, 0, fmt.Errorf("invalid bizNo length %d: expected 15", len(bizNo))
	}
	prefix = string(bizNo[0])
	if !IsValidPrefix(prefix) {
		return "", "", 0, 0, 0, ErrInvalidPrefix
	}
	dateStr = bizNo[1:7]
	rawID, err := DecodeCrockfordBase32(bizNo[7:15])
	if err != nil {
		return "", "", 0, 0, 0, err
	}
	dayMillis = int64((rawID >> 12) & DayMillisecondsMask)
	workerID = uint8((rawID >> 8) & 0x0F)
	sequence = uint8(rawID & MaxSequence)
	return prefix, dateStr, dayMillis, workerID, sequence, nil
}
