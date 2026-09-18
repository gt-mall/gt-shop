package analytics

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/util"
)

// TimeRange constants
const (
	TimeRangeToday      = "today"
	TimeRangeYesterday  = "yesterday"
	TimeRangeLast7Days  = "last_7_days"
	TimeRangeLast30Days = "last_30_days"
	TimeRangeThisMonth  = "this_month"
	TimeRangeLastMonth  = "last_month"
	TimeRangeCustom     = "custom"
)

// ParseTimeRange parses standard time range options or custom dates.
// Returns [startTime, endTime), [prevStartTime, prevEndTime), and the slice of date strings in YYYY-MM-DD.
func ParseTimeRange(timeRange, startTimeStr, endTimeStr string) (
	startTime, endTime, prevStartTime, prevEndTime time.Time,
	dates []string,
	err error,
) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	switch strings.TrimSpace(timeRange) {
	case "", TimeRangeToday:
		startTime = todayStart
		endTime = tomorrowStart
		prevStartTime = todayStart.AddDate(0, 0, -1)
		prevEndTime = todayStart
		dates = []string{todayStart.Format("2006-01-02")}

	case TimeRangeYesterday:
		startTime = todayStart.AddDate(0, 0, -1)
		endTime = todayStart
		prevStartTime = todayStart.AddDate(0, 0, -2)
		prevEndTime = startTime
		dates = []string{startTime.Format("2006-01-02")}

	case TimeRangeLast7Days:
		startTime = todayStart.AddDate(0, 0, -6)
		endTime = tomorrowStart
		prevStartTime = startTime.AddDate(0, 0, -7)
		prevEndTime = startTime
		dates = buildDateSlice(startTime, endTime)

	case TimeRangeLast30Days:
		startTime = todayStart.AddDate(0, 0, -29)
		endTime = tomorrowStart
		prevStartTime = startTime.AddDate(0, 0, -30)
		prevEndTime = startTime
		dates = buildDateSlice(startTime, endTime)

	case TimeRangeThisMonth:
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endTime = tomorrowStart
		// Previous cycle: align Month-to-date with natural days of previous month
		prevStartTime = startTime.AddDate(0, -1, 0)
		lastDayOfPrevMonth := startTime.AddDate(0, 0, -1).Day()
		targetDay := now.Day()
		if targetDay > lastDayOfPrevMonth {
			targetDay = lastDayOfPrevMonth
		}
		prevEndTime = time.Date(prevStartTime.Year(), prevStartTime.Month(), targetDay, 0, 0, 0, 0, prevStartTime.Location()).AddDate(0, 0, 1)
		dates = buildDateSlice(startTime, endTime)

	case TimeRangeLastMonth:
		thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		startTime = thisMonthStart.AddDate(0, -1, 0)
		endTime = thisMonthStart
		prevStartTime = startTime.AddDate(0, -1, 0)
		prevEndTime = startTime
		dates = buildDateSlice(startTime, endTime)

	case TimeRangeCustom:
		startTime, endTime, err = parseCustomRange(startTimeStr, endTimeStr, now.Location())
		if err != nil {
			return time.Time{}, time.Time{}, time.Time{}, time.Time{}, nil, err
		}
		duration := endTime.Sub(startTime)
		prevEndTime = startTime
		prevStartTime = startTime.Add(-duration)
		dates = buildDateSlice(startTime, endTime)

	default:
		// Unknown range, fallback to today
		startTime = todayStart
		endTime = tomorrowStart
		prevStartTime = todayStart.AddDate(0, 0, -1)
		prevEndTime = todayStart
		dates = []string{todayStart.Format("2006-01-02")}
	}

	return startTime, endTime, prevStartTime, prevEndTime, dates, nil
}

func parseCustomRange(startTimeStr, endTimeStr string, loc *time.Location) (time.Time, time.Time, error) {
	startTimeStr = strings.TrimSpace(startTimeStr)
	endTimeStr = strings.TrimSpace(endTimeStr)
	if startTimeStr == "" || endTimeStr == "" {
		return time.Time{}, time.Time{}, errors.New("自定义时间范围缺少开始或结束时间")
	}

	parseDate := func(s string, isEnd bool) (time.Time, error) {
		layouts := []string{
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
			"2006-01-02",
			time.RFC3339,
		}
		for _, l := range layouts {
			if t, err := time.ParseInLocation(l, s, loc); err == nil {
				if isEnd {
					if l == "2006-01-02" {
						// End date without time is inclusive of the whole day -> add 1 day
						return t.AddDate(0, 0, 1), nil
					}
					// If end time is 23:59:59, advance to next day 00:00:00 for [start, end) half-open interval
					if t.Hour() == 23 && t.Minute() == 59 && t.Second() == 59 {
						return t.Add(time.Second), nil
					}
				}
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("无法解析时间格式: %s", s)
	}

	st, err := parseDate(startTimeStr, false)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	et, err := parseDate(endTimeStr, true)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if et.Before(st) {
		return time.Time{}, time.Time{}, errors.New("结束时间不能早于开始时间")
	}
	return st, et, nil
}

func buildDateSlice(start, end time.Time) []string {
	var dates []string
	cur := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	if end.After(endDay) {
		endDay = endDay.AddDate(0, 0, 1)
	}

	for cur.Before(endDay) {
		dates = append(dates, cur.Format("2006-01-02"))
		cur = cur.AddDate(0, 0, 1)
		// Limit to at most 366 dates to prevent extreme queries
		if len(dates) >= 366 {
			break
		}
	}
	return dates
}

// FormatSignedCentToYuan formats integer cents (which can be negative) into a 2-decimal string.
// Example: 19990 -> "199.90", -5000 -> "-50.00", 0 -> "0.00"
func FormatSignedCentToYuan(cents int64) string {
	if cents == 0 {
		return "0.00"
	}
	neg := cents < 0
	if neg {
		cents = -cents
	}
	intPart := cents / 100
	decPart := cents % 100
	if neg {
		return fmt.Sprintf("-%d.%02d", intPart, decPart)
	}
	return fmt.Sprintf("%d.%02d", intPart, decPart)
}

// CalcGrowthRate computes ((curr - prev) / prev) * 100% using integer math.
// When baseline prevCents <= 0, returns "--" as growth rate has no valid base.
func CalcGrowthRate(currCents, prevCents int64) string {
	if prevCents <= 0 {
		return "--"
	}

	diff := currCents - prevCents
	if diff == 0 {
		return "0.00%"
	}

	pct := (diff * 10000) / prevCents
	neg := pct < 0
	if neg {
		pct = -pct
	}
	intPart := pct / 100
	decPart := pct % 100

	if neg {
		return fmt.Sprintf("-%d.%02d%%", intPart, decPart)
	}
	return fmt.Sprintf("+%d.%02d%%", intPart, decPart)
}

// CalcCountGrowthRate computes count growth rate.
func CalcCountGrowthRate(currCount, prevCount int64) string {
	return CalcGrowthRate(currCount, prevCount)
}

// CalcRefundRate computes refund amount ratio = (refundCents / salesCents) * 100%.
// When salesCents <= 0, returns "--" instead of hardcoding 100.00%.
func CalcRefundRate(refundCents, salesCents int64) string {
	if salesCents <= 0 {
		return "--"
	}
	if refundCents <= 0 {
		return "0.00%"
	}
	pct := (refundCents * 10000) / salesCents
	return fmt.Sprintf("%d.%02d%%", pct/100, pct%100)
}

// SafeAOV calculates average order value = payCents / paidOrders.
func SafeAOV(payCents, paidOrders int64) string {
	if paidOrders <= 0 || payCents <= 0 {
		return "0.00"
	}
	aovCent := payCents / paidOrders
	return util.FormatCentToYuan(aovCent)
}

// SafePercentage returns percentage string = (numerator * 10000 / denominator)%
func SafePercentage(num, denom int64) string {
	if denom <= 0 || num <= 0 {
		return "0.00%"
	}
	pct := (num * 10000) / denom
	return fmt.Sprintf("%d.%02d%%", pct/100, pct%100)
}

// SafeRate is an alias for SafePercentage.
func SafeRate(num, denom int64) string {
	return SafePercentage(num, denom)
}

// PaymentStatusText returns display name for order_payment.status
func PaymentStatusText(status int8) string {
	return model.PaymentStatusText(status)
}

// RefundStatusText returns display name for order_refund.status
func RefundStatusText(status int8) string {
	return model.RefundStatusText(status)
}

// ParseDateTime safely parses date-time strings from SQLite/MySQL driver scans.
// Uses time.Local for formats without explicit timezone offset to avoid 8-hour UTC shift.
func ParseDateTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil {
			return t
		}
	}
	return time.Time{}
}
