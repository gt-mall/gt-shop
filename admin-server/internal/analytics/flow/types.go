package flow

import "time"

// FlowListRequest represents query filters and pagination for unified finance flows.
type FlowListRequest struct {
	Page      int    `json:"page" example:"1"`
	PageSize  int    `json:"page_size" example:"10"`
	FlowType  string `json:"flow_type" example:"all"` // all, payment, refund
	BizNo     string `json:"biz_no" example:"PAY202609170001"`
	OrderNo   string `json:"order_no" example:"ORD202609170001"`
	Channel   string `json:"channel" example:"wechat"`
	Status    *int8  `json:"status" example:"20"`
	StartTime string `json:"start_time" example:"2026-09-01 00:00:00"`
	EndTime   string `json:"end_time" example:"2026-09-17 23:59:59"`
} // @name AnalyticsFlowListRequest

// FinanceFlowRow represents a single unified record of payment or refund flow.
type FinanceFlowRow struct {
	ID            int64     `json:"id" example:"1"`
	BizNo         string    `json:"biz_no" example:"PAY202609170001"`
	FlowType      string    `json:"flow_type" example:"payment"` // payment 或 refund
	Direction     string    `json:"direction" example:"income"`  // income 或 expense
	OrderID       int64     `json:"order_id" example:"100"`
	OrderNo       string    `json:"order_no" example:"ORD202609170001"`
	Channel       string    `json:"channel" example:"wechat"`
	Amount        string    `json:"amount" example:"199.00"` // 正向十进制金额
	TransactionNo string    `json:"transaction_no" example:"WX_TRANS_123456"`
	Status        int8      `json:"status" example:"20"`
	StatusText    string    `json:"status_text" example:"支付成功"`
	OccurredAt    time.Time `json:"occurred_at"`
	CreatedAt     time.Time `json:"created_at"`
} // @name AnalyticsFinanceFlowRow

// FlowListResponse represents paginated flow records along with financial aggregate summaries.
type FlowListResponse struct {
	Total              int64            `json:"total" example:"150"`
	Items              []FinanceFlowRow `json:"items"`
	TotalIncomeAmount  string           `json:"total_income_amount" example:"18500.00"`
	TotalExpenseAmount string           `json:"total_expense_amount" example:"1200.00"`
	NetFlowAmount      string           `json:"net_flow_amount" example:"17300.00"`
} // @name AnalyticsFlowListResponse
