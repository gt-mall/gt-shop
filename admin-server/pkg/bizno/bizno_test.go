package bizno_test

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"mall/admin_core/pkg/bizno"
)

// 1. 格式测试
func TestGenerator_Format(t *testing.T) {
	g, err := bizno.NewGenerator(0)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	orderNo, err := g.Next(bizno.BizPrefixOrder)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	if len(orderNo) != 15 {
		t.Fatalf("expected length 15, got %d (%s)", len(orderNo), orderNo)
	}

	pattern := `^D[0-9]{6}[0-9ABCDEFGHJKMNPQRSTVWXYZ]{8}$`
	matched, err := regexp.MatchString(pattern, orderNo)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Errorf("orderNo %s does not match pattern %s", orderNo, pattern)
	}
}

// 2. Prefix测试 (D, P, R, A, S)
func TestGenerator_AllowedPrefixes(t *testing.T) {
	g, err := bizno.NewGenerator(1)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	testCases := []struct {
		prefix string
	}{
		{bizno.BizPrefixOrder},
		{bizno.BizPrefixPayment},
		{bizno.BizPrefixRefund},
		{bizno.BizPrefixAfterSale},
		{bizno.BizPrefixShipment},
	}

	for _, tc := range testCases {
		t.Run("Prefix_"+tc.prefix, func(t *testing.T) {
			no, err := g.Next(tc.prefix)
			if err != nil {
				t.Fatalf("Next(%s) failed: %v", tc.prefix, err)
			}
			if len(no) != 15 {
				t.Fatalf("expected length 15, got %d (%s)", len(no), no)
			}
			if string(no[0]) != tc.prefix {
				t.Errorf("expected prefix %s, got %s", tc.prefix, string(no[0]))
			}
			pattern := fmt.Sprintf(`^%s[0-9]{6}[0-9ABCDEFGHJKMNPQRSTVWXYZ]{8}$`, tc.prefix)
			matched, err := regexp.MatchString(pattern, no)
			if err != nil {
				t.Fatalf("regex error: %v", err)
			}
			if !matched {
				t.Errorf("bizNo %s does not match pattern %s", no, pattern)
			}
		})
	}
}

// 3. 非法 Prefix 测试
func TestGenerator_InvalidPrefix(t *testing.T) {
	g, err := bizno.NewGenerator(0)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	invalidPrefixes := []string{"X", "d", "p", "O", "B", "", "DD", "ORDER", "1"}
	for _, p := range invalidPrefixes {
		t.Run("Invalid_"+p, func(t *testing.T) {
			no, err := g.Next(p)
			if err == nil {
				t.Fatalf("expected error for invalid prefix %q, got no=%s", p, no)
			}
			if !errors.Is(err, bizno.ErrInvalidPrefix) {
				t.Errorf("expected ErrInvalidPrefix, got: %v", err)
			}
		})
	}
}

// 4. Worker 边界测试 (允许 0, 15; 拒绝 16, 255)
func TestGenerator_WorkerBoundary(t *testing.T) {
	validWorkers := []uint8{0, 1, 7, 14, 15}
	for _, w := range validWorkers {
		g, err := bizno.NewGenerator(w)
		if err != nil {
			t.Errorf("expected valid worker %d, got err: %v", w, err)
		}
		if g.WorkerID() != w {
			t.Errorf("expected workerID %d, got %d", w, g.WorkerID())
		}
	}

	invalidWorkers := []uint8{16, 17, 32, 100, 255}
	for _, w := range invalidWorkers {
		g, err := bizno.NewGenerator(w)
		if err == nil {
			t.Errorf("expected error for worker %d, got generator: %v", w, g)
		}
		if !errors.Is(err, bizno.ErrInvalidWorkerID) {
			t.Errorf("expected ErrInvalidWorkerID, got: %v", err)
		}
	}
}

// 5. 单线程唯一性 (连续生成 100000 个订单号，0 重复)
func TestGenerator_SingleThreadUniqueness(t *testing.T) {
	// Use FakeClock for fast, deterministic 100k generation
	initial := time.Date(2026, 9, 11, 10, 0, 0, 0, time.Local)
	fake := bizno.NewFakeClock(initial)
	g, err := bizno.NewGenerator(0, bizno.WithClock(fake))
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	const count = 100000
	seen := make(map[string]struct{}, count)

	for i := 0; i < count; i++ {
		no, err := g.Next(bizno.BizPrefixOrder)
		if err != nil {
			t.Fatalf("iteration %d failed: %v", i, err)
		}
		if _, exists := seen[no]; exists {
			t.Fatalf("duplicate orderNo detected at iteration %d: %s", i, no)
		}
		seen[no] = struct{}{}
	}

	if len(seen) != count {
		t.Fatalf("expected %d unique orderNos, got %d", count, len(seen))
	}
}

// 6. 高并发唯一性 (100 goroutine，每个生成 1000 个，总共 100000 个，0 重复，go test -race)
func TestGenerator_HighConcurrencyUniqueness(t *testing.T) {
	initial := time.Date(2026, 9, 11, 10, 0, 0, 0, time.Local)
	fake := bizno.NewFakeClock(initial)
	g, err := bizno.NewGenerator(2, bizno.WithClock(fake))
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	const goroutines = 100
	const perGoroutine = 1000
	const total = goroutines * perGoroutine

	var wg sync.WaitGroup
	results := make([][]string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ids := make([]string, perGoroutine)
			for j := 0; j < perGoroutine; j++ {
				no, err := g.Next(bizno.BizPrefixOrder)
				if err != nil {
					t.Errorf("goroutine %d iter %d failed: %v", idx, j, err)
					return
				}
				ids[j] = no
			}
			results[idx] = ids
		}(i)
	}
	wg.Wait()

	seen := make(map[string]struct{}, total)
	for i := 0; i < goroutines; i++ {
		for _, id := range results[i] {
			if id == "" {
				t.Fatalf("empty ID encountered in goroutine %d", i)
			}
			if _, exists := seen[id]; exists {
				t.Fatalf("duplicate ID detected: %s", id)
			}
			seen[id] = struct{}{}
		}
	}

	if len(seen) != total {
		t.Fatalf("expected %d unique IDs, got %d", total, len(seen))
	}
}

// 7. 同毫秒 sequence 测试 (固定同一个毫秒，连续生成多个编号)
func TestGenerator_SameMillisecondSequence(t *testing.T) {
	fixedTime := time.Date(2026, 9, 11, 12, 0, 0, 500000000, time.Local)
	fake := bizno.NewFakeClock(fixedTime)
	g, err := bizno.NewGenerator(5, bizno.WithClock(fake))
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	const n = 10
	nos := make([]string, n)
	for i := 0; i < n; i++ {
		no, err := g.Next(bizno.BizPrefixOrder)
		if err != nil {
			t.Fatalf("Next(%d) failed: %v", i, err)
		}
		nos[i] = no

		prefix, dateStr, _, workerID, seq, err := bizno.DeconstructBizNo(no)
		if err != nil {
			t.Fatalf("DeconstructBizNo failed: %v", err)
		}
		if prefix != "D" {
			t.Errorf("expected prefix D, got %s", prefix)
		}
		if dateStr != "260911" {
			t.Errorf("expected date 260911, got %s", dateStr)
		}
		if workerID != 5 {
			t.Errorf("expected workerID 5, got %d", workerID)
		}
		if int(seq) != i {
			t.Errorf("expected sequence %d, got %d", i, seq)
		}
	}

	// 确认编码结果不同，且严格单调递增
	for i := 0; i < n-1; i++ {
		if nos[i] == nos[i+1] {
			t.Errorf("expected nos[%d] != nos[%d], both are %s", i, i+1, nos[i])
		}
		if nos[i] >= nos[i+1] {
			t.Errorf("expected nos[%d] < nos[%d], got %s >= %s", i, i+1, nos[i], nos[i+1])
		}
	}
}

// 8. sequence 溢出测试 (生成 256 个 ID，第 257 个必须等待到下一毫秒后继续，sequence 归零)
func TestGenerator_SequenceOverflow(t *testing.T) {
	fixedTime := time.Date(2026, 9, 11, 14, 30, 0, 100000000, time.Local)
	fake := bizno.NewFakeClock(fixedTime)
	g, err := bizno.NewGenerator(3, bizno.WithClock(fake))
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	// 生成 256 个 ID (sequence 0 ~ 255)
	for i := 0; i < 256; i++ {
		no, err := g.Next(bizno.BizPrefixOrder)
		if err != nil {
			t.Fatalf("generate %d failed: %v", i, err)
		}
		_, _, _, _, seq, err := bizno.DeconstructBizNo(no)
		if err != nil {
			t.Fatalf("deconstruct %d failed: %v", i, err)
		}
		if int(seq) != i {
			t.Fatalf("expected sequence %d, got %d", i, seq)
		}
	}

	// 第 257 个 ID：由于 FakeClock.Sleep(1ms) 会自动将时间推移 1ms，因此能顺利生成且 sequence 归零
	no257, err := g.Next(bizno.BizPrefixOrder)
	if err != nil {
		t.Fatalf("257th ID generation failed: %v", err)
	}

	_, _, dayMillis257, workerID257, seq257, err := bizno.DeconstructBizNo(no257)
	if err != nil {
		t.Fatalf("deconstruct 257th ID failed: %v", err)
	}
	if workerID257 != 3 {
		t.Errorf("expected workerID 3, got %d", workerID257)
	}
	if seq257 != 0 {
		t.Errorf("expected sequence 0 after overflow wait, got %d", seq257)
	}

	// 初始时间是 14:30:00.100 (100ms)，溢出等待后应该前进了 1ms 变成 14:30:00.101 (101ms)
	expectedDayMillis := int64((14*3600+30*60)*1000 + 101)
	if dayMillis257 != expectedDayMillis {
		t.Errorf("expected dayMillis %d, got %d", expectedDayMillis, dayMillis257)
	}
}

// 9. 时钟回拨测试 (上一毫秒 10000，下一次 9999，必须返回 clock moved backwards 错误，不返回编号)
func TestGenerator_ClockMovedBackwards(t *testing.T) {
	baseTime := time.UnixMilli(10000).In(time.Local)
	fake := bizno.NewFakeClock(baseTime)
	g, err := bizno.NewGenerator(0, bizno.WithClock(fake))
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	no1, err := g.Next(bizno.BizPrefixOrder)
	if err != nil {
		t.Fatalf("first Next failed: %v", err)
	}
	if no1 == "" {
		t.Fatalf("expected non-empty no1")
	}

	// 回拨到 9999ms
	fake.Set(time.UnixMilli(9999).In(time.Local))

	no2, err := g.Next(bizno.BizPrefixOrder)
	if err == nil {
		t.Fatalf("expected error on clock backwards, got no2=%s", no2)
	}
	if no2 != "" {
		t.Errorf("expected empty number on clock rollback, got %s", no2)
	}
	if !strings.Contains(err.Error(), "clock moved backwards") {
		t.Errorf("expected 'clock moved backwards' in error message, got: %v", err)
	}
}

// 10. 跨天测试 (模拟 2026-09-11 23:59:59.999 和 2026-09-12 00:00:00.000)
func TestGenerator_CrossDay(t *testing.T) {
	loc := time.Local
	t1 := time.Date(2026, 9, 11, 23, 59, 59, 999000000, loc)
	t2 := time.Date(2026, 9, 12, 0, 0, 0, 0, loc)

	fake := bizno.NewFakeClock(t1)
	g, err := bizno.NewGenerator(0, bizno.WithClock(fake))
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	no1, err := g.Next(bizno.BizPrefixOrder)
	if err != nil {
		t.Fatalf("Next(t1) failed: %v", err)
	}

	fake.Set(t2)
	no2, err := g.Next(bizno.BizPrefixOrder)
	if err != nil {
		t.Fatalf("Next(t2) failed: %v", err)
	}

	p1, d1, _, _, _, _ := bizno.DeconstructBizNo(no1)
	p2, d2, _, _, _, _ := bizno.DeconstructBizNo(no2)

	if p1 != "D" || p2 != "D" {
		t.Errorf("expected prefixes D, got %s and %s", p1, p2)
	}
	if d1 != "260911" {
		t.Errorf("expected date 260911, got %s", d1)
	}
	if d2 != "260912" {
		t.Errorf("expected date 260912, got %s", d2)
	}
	if d1 == d2 {
		t.Errorf("expected different dates across day boundary, got %s", d1)
	}
	if no1 == no2 {
		t.Errorf("expected different IDs across day boundary, got same %s", no1)
	}
	// 跨天依然满足字符串递增
	if no1 >= no2 {
		t.Errorf("expected no1 < no2 across day, got no1=%s >= no2=%s", no1, no2)
	}
}

// 11. 顺序测试 (在同一 worker 下按时间递增生成 no1 < no2 < no3)
func TestGenerator_ChronologicalOrder(t *testing.T) {
	loc := time.Local
	t1 := time.Date(2026, 9, 11, 10, 0, 0, 100000000, loc)
	t2 := time.Date(2026, 9, 11, 10, 0, 0, 200000000, loc)
	t3 := time.Date(2026, 9, 11, 10, 0, 0, 300000000, loc)

	fake := bizno.NewFakeClock(t1)
	g, err := bizno.NewGenerator(7, bizno.WithClock(fake))
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	no1, err := g.Next(bizno.BizPrefixOrder)
	if err != nil {
		t.Fatalf("Next(t1) failed: %v", err)
	}

	fake.Set(t2)
	no2, err := g.Next(bizno.BizPrefixOrder)
	if err != nil {
		t.Fatalf("Next(t2) failed: %v", err)
	}

	fake.Set(t3)
	no3, err := g.Next(bizno.BizPrefixOrder)
	if err != nil {
		t.Fatalf("Next(t3) failed: %v", err)
	}

	if !(no1 < no2 && no2 < no3) {
		t.Errorf("expected no1 < no2 < no3, got no1=%s, no2=%s, no3=%s", no1, no2, no3)
	}
}

// 12. Base32 字符测试 (后 8 位不得出现 I, L, O, U)
func TestGenerator_Base32Characters(t *testing.T) {
	initial := time.Date(2026, 9, 11, 8, 0, 0, 0, time.Local)
	fake := bizno.NewFakeClock(initial)
	g, err := bizno.NewGenerator(15, bizno.WithClock(fake))
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	forbiddenChars := "ILOU"
	const testCount = 100000

	for i := 0; i < testCount; i++ {
		no, err := g.Next(bizno.BizPrefixOrder)
		if err != nil {
			t.Fatalf("Next at %d failed: %v", i, err)
		}
		base32Part := no[7:15]
		if strings.ContainsAny(base32Part, forbiddenChars) {
			t.Fatalf("forbidden character found in base32 code: %s in %s", base32Part, no)
		}
	}
}
