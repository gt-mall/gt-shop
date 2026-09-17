package member

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"mall/admin_core/internal/analytics"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/util"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type memberPayAggRow struct {
	MemberID       int64  `gorm:"column:member_id"`
	PaidOrderCount int64  `gorm:"column:paid_order_count"`
	TotalPayAmount string `gorm:"column:total_pay_amount"`
	LastPaidAt     string `gorm:"column:last_paid_at"`
}

type dateCountRow struct {
	D   string `gorm:"column:d"`
	Cnt int64  `gorm:"column:cnt"`
}

func (s *Service) Member(ctx context.Context, req MemberQueryRequest) (MemberResponse, error) {
	if s.db == nil {
		return MemberResponse{}, errors.New("数据库服务不可用")
	}

	startTime, endTime, _, _, dates, err := analytics.ParseTimeRange(req.TimeRange, req.StartTime, req.EndTime)
	if err != nil {
		return MemberResponse{}, err
	}

	// 1. New members & total members
	var newMemberCount int64
	err = s.db.WithContext(ctx).Table("member").
		Where("deleted_at IS NULL AND created_at >= ? AND created_at < ?", startTime, endTime).
		Count(&newMemberCount).Error
	if err != nil {
		return MemberResponse{}, err
	}

	var totalMemberCount int64
	err = s.db.WithContext(ctx).Table("member").
		Where("deleted_at IS NULL AND created_at < ?", endTime).
		Count(&totalMemberCount).Error
	if err != nil {
		return MemberResponse{}, err
	}

	// 2. Paying members in the period
	var payingRows []memberPayAggRow
	err = s.db.WithContext(ctx).Table("order_payment AS p").
		Select(`
			o.member_id,
			COUNT(DISTINCT p.order_id) AS paid_order_count,
			COALESCE(SUM(p.amount), 0) AS total_pay_amount,
			MAX(p.paid_at) AS last_paid_at
		`).
		Joins("INNER JOIN order_main AS o ON p.order_id = o.id").
		Where("p.status = ? AND p.paid_at >= ? AND p.paid_at < ?", model.PaymentStatusSuccess, startTime, endTime).
		Group("o.member_id").
		Scan(&payingRows).Error
	if err != nil {
		return MemberResponse{}, err
	}

	paidMemberCount := int64(len(payingRows))
	var totalPayCents int64
	var totalPaidOrders int64
	memberIDs := make([]int64, len(payingRows))
	for i, r := range payingRows {
		memberIDs[i] = r.MemberID
		cents, _ := util.ParseDecimalYuanToCent(r.TotalPayAmount)
		totalPayCents += cents
		totalPaidOrders += r.PaidOrderCount
	}

	// 3. New paid members vs Old paid members
	oldMemberSet := make(map[int64]bool)
	if len(memberIDs) > 0 {
		var oldMembers []int64
		err = s.db.WithContext(ctx).Table("order_payment AS p").
			Joins("INNER JOIN order_main AS o ON p.order_id = o.id").
			Where("p.status = ? AND p.paid_at < ? AND o.member_id IN (?)", model.PaymentStatusSuccess, startTime, memberIDs).
			Pluck("DISTINCT o.member_id", &oldMembers).Error
		if err != nil {
			return MemberResponse{}, err
		}
		for _, mID := range oldMembers {
			oldMemberSet[mID] = true
		}
	}

	var newCustCount, oldCustCount int64
	var newCustCents, oldCustCents int64
	var newCustOrders, oldCustOrders int64

	for _, r := range payingRows {
		cents, _ := util.ParseDecimalYuanToCent(r.TotalPayAmount)
		if oldMemberSet[r.MemberID] {
			oldCustCount++
			oldCustCents += cents
			oldCustOrders += r.PaidOrderCount
		} else {
			newCustCount++
			newCustCents += cents
			newCustOrders += r.PaidOrderCount
		}
	}

	customerSegment := CustomerGroupAnalysis{
		NewCustomerCount:      newCustCount,
		NewCustomerPayAmount:  util.FormatCentToYuan(newCustCents),
		NewCustomerOrderCount: newCustOrders,
		NewCustomerPercentage: analytics.SafePercentage(newCustCount, paidMemberCount),

		OldCustomerCount:      oldCustCount,
		OldCustomerPayAmount:  util.FormatCentToYuan(oldCustCents),
		OldCustomerOrderCount: oldCustOrders,
		OldCustomerPercentage: analytics.SafePercentage(oldCustCount, paidMemberCount),
	}

	var avgSpending string
	var avgOrders string
	if paidMemberCount > 0 {
		avgSpending = util.FormatCentToYuan(totalPayCents / paidMemberCount)
		avgOrderNum := float64(totalPaidOrders) / float64(paidMemberCount)
		avgOrders = fmt.Sprintf("%.2f", avgOrderNum)
	} else {
		avgSpending = "0.00"
		avgOrders = "0.00"
	}

	metrics := MemberMetric{
		NewMemberCount:          newMemberCount,
		PaidMemberCount:         paidMemberCount,
		NewPaidMemberCount:      newCustCount,
		TotalMemberCount:        totalMemberCount,
		AverageMemberSpending:   avgSpending,
		AverageMemberOrderCount: avgOrders,
	}

	// 4. Trends
	trends, err := s.queryMemberTrends(ctx, startTime, endTime, dates)
	if err != nil {
		return MemberResponse{}, err
	}

	// 5. Top Members (Sort payingRows by totalPayAmount DESC, limit 10)
	topMembers, err := s.queryTopMembers(ctx, payingRows)
	if err != nil {
		return MemberResponse{}, err
	}

	return MemberResponse{
		Metrics:         metrics,
		Trends:          trends,
		CustomerSegment: customerSegment,
		TopMembers:      topMembers,
	}, nil
}

func (s *Service) queryMemberTrends(ctx context.Context, start, end time.Time, dates []string) ([]MemberTrendItem, error) {
	dateNewMap := make(map[string]int64, len(dates))
	datePaidMap := make(map[string]int64, len(dates))
	for _, d := range dates {
		dateNewMap[d] = 0
		datePaidMap[d] = 0
	}

	var newRows []dateCountRow
	err := s.db.WithContext(ctx).Table("member").
		Select("DATE(created_at) AS d, COUNT(*) AS cnt").
		Where("deleted_at IS NULL AND created_at >= ? AND created_at < ?", start, end).
		Group("DATE(created_at)").
		Scan(&newRows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range newRows {
		d := strings.TrimSpace(r.D)
		if len(d) >= 10 {
			d = d[:10]
		}
		dateNewMap[d] = r.Cnt
	}

	var paidRows []dateCountRow
	err = s.db.WithContext(ctx).Table("order_payment AS p").
		Select("DATE(p.paid_at) AS d, COUNT(DISTINCT o.member_id) AS cnt").
		Joins("INNER JOIN order_main AS o ON p.order_id = o.id").
		Where("p.status = ? AND p.paid_at >= ? AND p.paid_at < ?", model.PaymentStatusSuccess, start, end).
		Group("DATE(p.paid_at)").
		Scan(&paidRows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range paidRows {
		d := strings.TrimSpace(r.D)
		if len(d) >= 10 {
			d = d[:10]
		}
		datePaidMap[d] = r.Cnt
	}

	items := make([]MemberTrendItem, len(dates))
	for i, d := range dates {
		items[i] = MemberTrendItem{
			Date:            d,
			NewMemberCount:  dateNewMap[d],
			PaidMemberCount: datePaidMap[d],
		}
	}
	return items, nil
}

func (s *Service) queryTopMembers(ctx context.Context, payingRows []memberPayAggRow) ([]MemberRankingItem, error) {
	if len(payingRows) == 0 {
		return []MemberRankingItem{}, nil
	}

	sortedRows := make([]memberPayAggRow, len(payingRows))
	copy(sortedRows, payingRows)
	sort.Slice(sortedRows, func(i, j int) bool {
		cA, _ := util.ParseDecimalYuanToCent(sortedRows[i].TotalPayAmount)
		cB, _ := util.ParseDecimalYuanToCent(sortedRows[j].TotalPayAmount)
		if cA != cB {
			return cA > cB
		}
		return sortedRows[i].MemberID < sortedRows[j].MemberID
	})

	if len(sortedRows) > 10 {
		sortedRows = sortedRows[:10]
	}

	topIDs := make([]int64, len(sortedRows))
	for i, r := range sortedRows {
		topIDs[i] = r.MemberID
	}

	var members []model.Member
	if err := s.db.WithContext(ctx).Where("id IN (?)", topIDs).Find(&members).Error; err != nil {
		return nil, err
	}
	memberInfoMap := make(map[int64]model.Member, len(members))
	for _, m := range members {
		memberInfoMap[m.ID] = m
	}

	items := make([]MemberRankingItem, len(sortedRows))
	for i, r := range sortedRows {
		m := memberInfoMap[r.MemberID]
		cents, _ := util.ParseDecimalYuanToCent(r.TotalPayAmount)
		var lastPaidAt *time.Time
		if r.LastPaidAt != "" {
			t := analytics.ParseDateTime(r.LastPaidAt)
			if !t.IsZero() {
				lastPaidAt = &t
			}
		}
		items[i] = MemberRankingItem{
			MemberID:       r.MemberID,
			Nickname:       m.Nickname,
			Mobile:         formatMemberMobile(m.Mobile),
			AvatarURL:      "",
			PaidOrderCount: r.PaidOrderCount,
			TotalPayAmount: util.FormatCentToYuan(cents),
			LastPaidAt:     lastPaidAt,
		}
	}
	return items, nil
}

func formatMemberMobile(m *string) string {
	if m == nil {
		return ""
	}
	return *m
}
