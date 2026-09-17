package member

import "time"

// MemberQueryRequest represents query parameters for member analytics.
type MemberQueryRequest struct {
	TimeRange string `form:"time_range" json:"time_range" example:"last_7_days"`
	StartTime string `form:"start_time" json:"start_time" example:"2026-09-01 00:00:00"`
	EndTime   string `form:"end_time" json:"end_time" example:"2026-09-17 23:59:59"`
} // @name AnalyticsMemberQueryRequest

// MemberMetric represents summary metrics for member analytics.
type MemberMetric struct {
	NewMemberCount          int64  `json:"new_member_count" example:"45"`
	PaidMemberCount         int64  `json:"paid_member_count" example:"32"`
	NewPaidMemberCount      int64  `json:"new_paid_member_count" example:"18"`
	TotalMemberCount        int64  `json:"total_member_count" example:"520"`
	AverageMemberSpending   string `json:"average_member_spending" example:"350.00"`
	AverageMemberOrderCount string `json:"average_member_order_count" example:"1.50"`
} // @name AnalyticsMemberMetric

// MemberTrendItem represents daily member growth and activity.
type MemberTrendItem struct {
	Date            string `json:"date" example:"2026-09-10"`
	NewMemberCount  int64  `json:"new_member_count" example:"6"`
	PaidMemberCount int64  `json:"paid_member_count" example:"8"`
} // @name AnalyticsMemberTrendItem

// CustomerGroupAnalysis represents new vs returning customer comparison.
type CustomerGroupAnalysis struct {
	NewCustomerCount      int64  `json:"new_customer_count" example:"18"`
	NewCustomerPayAmount  string `json:"new_customer_pay_amount" example:"4500.00"`
	NewCustomerOrderCount int64  `json:"new_customer_order_count" example:"20"`
	NewCustomerPercentage string `json:"new_customer_percentage" example:"56.25%"`

	OldCustomerCount      int64  `json:"old_customer_count" example:"14"`
	OldCustomerPayAmount  string `json:"old_customer_pay_amount" example:"6700.00"`
	OldCustomerOrderCount int64  `json:"old_customer_order_count" example:"28"`
	OldCustomerPercentage string `json:"old_customer_percentage" example:"43.75%"`
} // @name AnalyticsCustomerGroupAnalysis

// MemberRankingItem represents top paying members.
type MemberRankingItem struct {
	MemberID       int64      `json:"member_id" example:"1"`
	Nickname       string     `json:"nickname" example:"张三"`
	Mobile         string     `json:"mobile" example:"13800138000"`
	AvatarURL      string     `json:"avatar_url" example:"https://example.com/a.jpg"`
	PaidOrderCount int64      `json:"paid_order_count" example:"5"`
	TotalPayAmount string     `json:"total_pay_amount" example:"1280.00"`
	LastPaidAt     *time.Time `json:"last_paid_at"`
} // @name AnalyticsMemberRankingItem

// MemberResponse represents full member analytics response.
type MemberResponse struct {
	Metrics         MemberMetric          `json:"metrics"`
	Trends          []MemberTrendItem     `json:"trends"`
	CustomerSegment CustomerGroupAnalysis `json:"customer_segment"`
	TopMembers      []MemberRankingItem   `json:"top_members"`
} // @name AnalyticsMemberResponse
