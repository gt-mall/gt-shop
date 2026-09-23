package miniapp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	miniapporder "mall/admin_core/internal/miniapp/order"
	miniappreview "mall/admin_core/internal/miniapp/review"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/response"
)

func createTestPNGBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func createCompletedOrderWithItem(t *testing.T, db *gorm.DB, memberID int64, reqID string) (int64, int64, int64) {
	t.Helper()
	prodID, skuID, addrID := createOrderTestData(t, db, memberID)
	order, err := miniapporder.NewService(db).Create(context.Background(), model.Member{ID: memberID}, miniapporder.OrderCreateRequest{
		RequestID: reqID,
		AddressID: addrID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("create test order failed: %v", err)
	}

	newWechatPaymentFixture(t, db, memberID).pay(t, order.ID)

	// Mark order as completed
	now := time.Now()
	if err := db.Model(&model.OrderMain{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status":       model.OrderStatusCompleted,
		"completed_at": &now,
	}).Error; err != nil {
		t.Fatalf("update order completed failed: %v", err)
	}

	var item model.OrderItem
	if err := db.Where("order_id = ?", order.ID).First(&item).Error; err != nil {
		t.Fatalf("find order item: %v", err)
	}

	return order.ID, item.ID, prodID
}

func TestMiniappReview_Upload(t *testing.T) {
	_, handler, tokenA, _, _, _ := setupOrderTestApp(t)

	// 1. Upload valid PNG image
	pngData := createTestPNGBytes()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "review.png")
	_, _ = part.Write(pngData)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp response.Response[miniappreview.MiniappReviewUploadResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal upload resp: %v", err)
	}
	if resp.Code != 1 || resp.Data.FileID <= 0 {
		t.Fatalf("expected valid file id, got %+v", resp)
	}

	// 2. Reject non-image file (e.g. .txt)
	body2 := &bytes.Buffer{}
	writer2 := multipart.NewWriter(body2)
	part2, _ := writer2.CreateFormFile("file", "exploit.txt")
	_, _ = part2.Write([]byte("plain text not image"))
	_ = writer2.Close()

	req2 := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/upload", body2)
	req2.Header.Set("Content-Type", writer2.FormDataContentType())
	req2.Header.Set("Authorization", "Bearer "+tokenA)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	var resp2 response.Response[any]
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Code == 1 {
		t.Fatalf("expected text file upload to be rejected, got success: %s", w2.Body.String())
	}

	// 3. Unauthorized upload
	req3 := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/upload", body)
	req3.Header.Set("Content-Type", writer.FormDataContentType())
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	var resp3 response.Response[any]
	_ = json.Unmarshal(w3.Body.Bytes(), &resp3)
	if resp3.Code != -1 {
		t.Fatalf("expected code -1 for unauthorized, got %d", resp3.Code)
	}
}

func TestMiniappReview_CreationAndEligibility(t *testing.T) {
	db, handler, tokenA, memberIDA, tokenB, memberIDB := setupOrderTestApp(t)
	_ = memberIDB

	orderID, itemID, prodID := createCompletedOrderWithItem(t, db, memberIDA, "rev-test-ord-1")

	// 1. Upload review image
	pngData := createTestPNGBytes()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "photo.png")
	_, _ = part.Write(pngData)
	_ = writer.Close()

	upReq := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/upload", body)
	upReq.Header.Set("Content-Type", writer.FormDataContentType())
	upReq.Header.Set("Authorization", "Bearer "+tokenA)
	upW := httptest.NewRecorder()
	handler.ServeHTTP(upW, upReq)

	var upResp response.Response[miniappreview.MiniappReviewUploadResponse]
	_ = json.Unmarshal(upW.Body.Bytes(), &upResp)
	fileID := upResp.Data.FileID

	// 2. Cross-user attempt: User B trying to review User A's completed item
	createReq := miniappreview.MiniappReviewCreateRequest{
		OrderItemID: itemID,
		Rating:      5,
		Content:     "非常满意，物美价廉！",
		ImageIDs:    []int64{fileID},
		IsAnonymous: 0,
		RequestID:   "rev-req-b-1",
	}
	bJSON, _ := json.Marshal(createReq)
	reqB := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(bJSON))
	reqB.Header.Set("Content-Type", "application/json")
	reqB.Header.Set("Authorization", "Bearer "+tokenB)
	wB := httptest.NewRecorder()
	handler.ServeHTTP(wB, reqB)

	var respB response.Response[any]
	_ = json.Unmarshal(wB.Body.Bytes(), &respB)
	if respB.Code == 1 {
		t.Fatalf("user B should not be able to review user A's order: %s", wB.Body.String())
	}

	// 3. User A review validation: rating out of bounds
	badRatingReq := createReq
	badRatingReq.Rating = 6
	badJSON, _ := json.Marshal(badRatingReq)
	reqBad := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(badJSON))
	reqBad.Header.Set("Content-Type", "application/json")
	reqBad.Header.Set("Authorization", "Bearer "+tokenA)
	wBad := httptest.NewRecorder()
	handler.ServeHTTP(wBad, reqBad)
	var respBad response.Response[any]
	_ = json.Unmarshal(wBad.Body.Bytes(), &respBad)
	if respBad.Code == 1 {
		t.Fatalf("expected rating 6 to be rejected")
	}

	// 4. User A review validation: empty or whitespace content
	emptyReq := createReq
	emptyReq.Content = "   "
	emptyJSON, _ := json.Marshal(emptyReq)
	reqEmpty := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(emptyJSON))
	reqEmpty.Header.Set("Content-Type", "application/json")
	reqEmpty.Header.Set("Authorization", "Bearer "+tokenA)
	wEmpty := httptest.NewRecorder()
	handler.ServeHTTP(wEmpty, reqEmpty)
	var respEmpty response.Response[any]
	_ = json.Unmarshal(wEmpty.Body.Bytes(), &respEmpty)
	if respEmpty.Code == 1 {
		t.Fatalf("expected empty content to be rejected")
	}

	// 5. User A review validation: content over 500 runes
	longReq := createReq
	longReq.Content = strings.Repeat("好", 501)
	longJSON, _ := json.Marshal(longReq)
	reqLong := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(longJSON))
	reqLong.Header.Set("Content-Type", "application/json")
	reqLong.Header.Set("Authorization", "Bearer "+tokenA)
	wLong := httptest.NewRecorder()
	handler.ServeHTTP(wLong, reqLong)
	var respLong response.Response[any]
	_ = json.Unmarshal(wLong.Body.Bytes(), &respLong)
	if respLong.Code == 1 {
		t.Fatalf("expected 501 runes content to be rejected")
	}

	// 6. User A successful review creation
	successReq := createReq
	successReq.Content = "质量非常棒，包装精美，发货速度极快！"
	successReq.RequestID = "rev-req-a-1"
	succJSON, _ := json.Marshal(successReq)
	reqSucc := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(succJSON))
	reqSucc.Header.Set("Content-Type", "application/json")
	reqSucc.Header.Set("Authorization", "Bearer "+tokenA)
	wSucc := httptest.NewRecorder()
	handler.ServeHTTP(wSucc, reqSucc)

	if wSucc.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wSucc.Code, wSucc.Body.String())
	}
	var respSucc response.Response[miniappreview.MiniappReviewCreateResponse]
	if err := json.Unmarshal(wSucc.Body.Bytes(), &respSucc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if respSucc.Code != 1 || respSucc.Data.ID <= 0 {
		t.Fatalf("expected review id > 0, got %+v", respSucc)
	}
	reviewID := respSucc.Data.ID

	// 7. Idempotency test: Replay identical request with same request_id
	reqReplay := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(succJSON))
	reqReplay.Header.Set("Content-Type", "application/json")
	reqReplay.Header.Set("Authorization", "Bearer "+tokenA)
	wReplay := httptest.NewRecorder()
	handler.ServeHTTP(wReplay, reqReplay)
	var respReplay response.Response[miniappreview.MiniappReviewCreateResponse]
	_ = json.Unmarshal(wReplay.Body.Bytes(), &respReplay)
	if respReplay.Code != 1 || respReplay.Data.ID != reviewID {
		t.Fatalf("expected idempotent response with same ID %d, got %+v", reviewID, respReplay)
	}

	// 8. Idempotency conflict test: Same request_id with different content
	conflictReq := successReq
	conflictReq.Content = "不同内容的冲突请求"
	conflictJSON, _ := json.Marshal(conflictReq)
	reqConflict := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(conflictJSON))
	reqConflict.Header.Set("Content-Type", "application/json")
	reqConflict.Header.Set("Authorization", "Bearer "+tokenA)
	wConflict := httptest.NewRecorder()
	handler.ServeHTTP(wConflict, reqConflict)
	var respConflict response.Response[any]
	_ = json.Unmarshal(wConflict.Body.Bytes(), &respConflict)
	if respConflict.Code == 1 {
		t.Fatalf("expected conflict error on mismatched payload, got success")
	}

	// 9. Duplicate review test on same item with new request_id
	dupReq := successReq
	dupReq.RequestID = "rev-req-a-2"
	dupJSON, _ := json.Marshal(dupReq)
	reqDup := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(dupJSON))
	reqDup.Header.Set("Content-Type", "application/json")
	reqDup.Header.Set("Authorization", "Bearer "+tokenA)
	wDup := httptest.NewRecorder()
	handler.ServeHTTP(wDup, reqDup)
	var respDup response.Response[any]
	_ = json.Unmarshal(wDup.Body.Bytes(), &respDup)
	if respDup.Code == 1 {
		t.Fatalf("expected duplicate order item review rejection")
	}

	// 10. Check order detail and order list: can_review must be false, review_id set
	ordDetailReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/orders/%d", orderID), nil)
	ordDetailReq.Header.Set("Authorization", "Bearer "+tokenA)
	wOrd := httptest.NewRecorder()
	handler.ServeHTTP(wOrd, ordDetailReq)

	var ordResp response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(wOrd.Body.Bytes(), &ordResp)
	if len(ordResp.Data.Items) == 0 {
		t.Fatalf("expected order items")
	}
	item0 := ordResp.Data.Items[0]
	if item0.CanReview {
		t.Fatalf("expected can_review to be false for reviewed item")
	}
	if item0.ReviewID == nil || *item0.ReviewID != reviewID {
		t.Fatalf("expected review_id %d, got %v", reviewID, item0.ReviewID)
	}
	if item0.ReviewBlockReason != "已评价" {
		t.Fatalf("expected block reason '已评价', got %s", item0.ReviewBlockReason)
	}

	// 11. Mine reviews check
	mineReq := httptest.NewRequest(http.MethodGet, "/api/miniapp/reviews/mine", nil)
	mineReq.Header.Set("Authorization", "Bearer "+tokenA)
	wMine := httptest.NewRecorder()
	handler.ServeHTTP(wMine, mineReq)

	var mineResp response.Response[[]miniappreview.MiniappMineReviewItemResponse]
	_ = json.Unmarshal(wMine.Body.Bytes(), &mineResp)
	if len(mineResp.Data) == 0 || mineResp.Data[0].ID != reviewID {
		t.Fatalf("expected mine review list to contain review %d, got %+v", reviewID, mineResp)
	}
	if mineResp.Data[0].Status != model.ReviewStatusPendingReview {
		t.Fatalf("expected initial status 10, got %d", mineResp.Data[0].Status)
	}
	if len(mineResp.Data[0].Images) == 0 {
		t.Fatalf("expected author to see their own private image URLs")
	}

	// 12. Public product review: unapproved review must NOT be visible to public!
	pubReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/products/%d/reviews", prodID), nil)
	wPub := httptest.NewRecorder()
	handler.ServeHTTP(wPub, pubReq)

	var pubResp response.Response[miniappreview.MiniappProductReviewsResponse]
	_ = json.Unmarshal(wPub.Body.Bytes(), &pubResp)
	if pubResp.Data.Summary.TotalCount != 0 || len(pubResp.Data.Items) != 0 {
		t.Fatalf("unapproved review must not appear in public product reviews! got %d items", len(pubResp.Data.Items))
	}

	// 13. Approve review in DB directly to test public display & anonymization
	now := time.Now()
	_ = db.Model(&model.ProductReview{}).Where("id = ?", reviewID).Updates(map[string]any{
		"status":     model.ReviewStatusApproved,
		"audited_at": now,
	})
	_ = db.Model(&model.SystemFile{}).Where("id = ?", fileID).Update("visibility", model.VisibilityPublic)

	wPub2 := httptest.NewRecorder()
	handler.ServeHTTP(wPub2, pubReq)
	var pubResp2 response.Response[miniappreview.MiniappProductReviewsResponse]
	_ = json.Unmarshal(wPub2.Body.Bytes(), &pubResp2)
	if pubResp2.Data.Summary.TotalCount != 1 || len(pubResp2.Data.Items) != 1 {
		t.Fatalf("expected approved review to appear in public product reviews, got %d", len(pubResp2.Data.Items))
	}
	if pubResp2.Data.Summary.AverageRating != "5.0" || pubResp2.Data.Summary.GoodRate != "100%" {
		t.Fatalf("expected 5.0 and 100%%, got %+v", pubResp2.Data.Summary)
	}
	pubItem := pubResp2.Data.Items[0]
	if len(pubItem.Images) == 0 {
		t.Fatalf("expected public image URLs in approved review")
	}
}

func TestMiniappReview_AfterSaleBlock(t *testing.T) {
	db, handler, tokenA, memberIDA, _, _ := setupOrderTestApp(t)

	// Create completed order
	orderID, itemID, _ := createCompletedOrderWithItem(t, db, memberIDA, "as-block-ord-1")

	// 1. Create an active after-sale for this order item
	now := time.Now()
	as := model.AfterSale{
		AfterSaleNo:        "AS202609200001",
		RequestID:          "as-req-1",
		OrderID:            orderID,
		MemberID:           memberIDA,
		Type:               model.AfterSaleTypeRefundOnly,
		Status:             model.AfterSaleStatusPendingReview, // Active after-sale (10)
		OrderStatusAtApply: model.OrderStatusCompleted,
		AppliedAt:          now,
	}
	if err := db.Create(&as).Error; err != nil {
		t.Fatalf("create aftersale failed: %v", err)
	}

	asItem := model.AfterSaleItem{
		AfterSaleID:  as.ID,
		OrderItemID:  itemID,
		ProductID:    1,
		SKUID:        1,
		Quantity:     1,
		RefundAmount: "10.00",
		CreatedAt:    now,
	}
	if err := db.Create(&asItem).Error; err != nil {
		t.Fatalf("create aftersale item failed: %v", err)
	}

	// 2. Check order detail: can_review must be false, reason "售后处理中暂不可评价"
	ordDetailReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/orders/%d", orderID), nil)
	ordDetailReq.Header.Set("Authorization", "Bearer "+tokenA)
	wOrd := httptest.NewRecorder()
	handler.ServeHTTP(wOrd, ordDetailReq)

	var ordResp response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(wOrd.Body.Bytes(), &ordResp)
	if ordResp.Data.Items[0].CanReview {
		t.Fatalf("item under active aftersale should not be reviewable")
	}
	if ordResp.Data.Items[0].ReviewBlockReason != "售后处理中暂不可评价" {
		t.Fatalf("expected '售后处理中暂不可评价', got %s", ordResp.Data.Items[0].ReviewBlockReason)
	}

	// 3. Attempt to submit review during active after-sale -> must be rejected
	createReq := miniappreview.MiniappReviewCreateRequest{
		OrderItemID: itemID,
		Rating:      5,
		Content:     "尝试在售后中评价",
		RequestID:   "rev-as-block-1",
	}
	bJSON, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(bJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp response.Response[any]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code == 1 {
		t.Fatalf("review submission during active after-sale must be rejected, got: %s", w.Body.String())
	}

	// 4. Update after-sale to Completed (Refunded) -> must still be blocked
	_ = db.Model(&model.AfterSale{}).Where("id = ?", as.ID).Update("status", model.AfterSaleStatusCompleted)
	wOrd2 := httptest.NewRecorder()
	handler.ServeHTTP(wOrd2, ordDetailReq)
	var ordResp2 response.Response[miniapporder.OrderDetailResponse]
	_ = json.Unmarshal(wOrd2.Body.Bytes(), &ordResp2)
	if ordResp2.Data.Items[0].CanReview {
		t.Fatalf("refunded item should not be reviewable")
	}
	if ordResp2.Data.Items[0].ReviewBlockReason != "已退款不可评价" {
		t.Fatalf("expected '已退款不可评价', got %s", ordResp2.Data.Items[0].ReviewBlockReason)
	}
}

func TestMiniappReview_DetailFieldBoundaryAndAnonymity(t *testing.T) {
	db, handler, tokenA, memberIDA, tokenB, _ := setupOrderTestApp(t)

	orderID, itemID, _ := createCompletedOrderWithItem(t, db, memberIDA, "field-boundary-ord-1")

	// 1. Create an anonymous review by Member A
	createReq := miniappreview.MiniappReviewCreateRequest{
		OrderItemID: itemID,
		Rating:      5,
		Content:     "非常棒的匿名体验！",
		IsAnonymous: 1,
		RequestID:   "field-boundary-rev-1",
	}
	bJSON, _ := json.Marshal(createReq)
	reqA := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(bJSON))
	reqA.Header.Set("Content-Type", "application/json")
	reqA.Header.Set("Authorization", "Bearer "+tokenA)
	wA := httptest.NewRecorder()
	handler.ServeHTTP(wA, reqA)

	var createResp response.Response[miniappreview.MiniappReviewCreateResponse]
	_ = json.Unmarshal(wA.Body.Bytes(), &createResp)
	if createResp.Code != 1 {
		t.Fatalf("create review failed: %s", wA.Body.String())
	}
	revID := createResp.Data.ID

	// Set audit reason in DB to simulate internal moderation note
	_ = db.Model(&model.ProductReview{}).Where("id = ?", revID).Update("audit_reason", "internal moderation note")

	// 2. Member B requests unapproved review detail -> MUST fail
	reqB := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/reviews/%d", revID), nil)
	reqB.Header.Set("Authorization", "Bearer "+tokenB)
	wB := httptest.NewRecorder()
	handler.ServeHTTP(wB, reqB)

	var respB response.Response[any]
	_ = json.Unmarshal(wB.Body.Bytes(), &respB)
	if respB.Code == 1 {
		t.Fatalf("Member B should not be able to view unapproved review detail: %s", wB.Body.String())
	}

	// 3. Member A (Author) requests review detail -> MUST succeed and have private fields
	reqAuthor := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/reviews/%d", revID), nil)
	reqAuthor.Header.Set("Authorization", "Bearer "+tokenA)
	wAuthor := httptest.NewRecorder()
	handler.ServeHTTP(wAuthor, reqAuthor)

	var authorResp response.Response[miniappreview.MiniappReviewDetailResponse]
	_ = json.Unmarshal(wAuthor.Body.Bytes(), &authorResp)
	if authorResp.Code != 1 {
		t.Fatalf("author should be able to view own review: %s", wAuthor.Body.String())
	}
	if !authorResp.Data.IsAuthor {
		t.Fatalf("expected is_author=true for author")
	}
	if authorResp.Data.OrderID == nil || *authorResp.Data.OrderID != orderID {
		t.Fatalf("expected author to see order_id %d, got %v", orderID, authorResp.Data.OrderID)
	}
	if authorResp.Data.OrderItemID == nil || *authorResp.Data.OrderItemID != itemID {
		t.Fatalf("expected author to see order_item_id %d, got %v", itemID, authorResp.Data.OrderItemID)
	}
	if authorResp.Data.AuditReason == nil || *authorResp.Data.AuditReason != "internal moderation note" {
		t.Fatalf("expected author to see audit_reason, got %v", authorResp.Data.AuditReason)
	}

	// 4. Admin approves the review (status 20)
	_ = db.Model(&model.ProductReview{}).Where("id = ?", revID).Update("status", model.ReviewStatusApproved)

	// 5. Member B requests approved review detail -> MUST succeed, but order/audit fields MUST NOT be present
	wB2 := httptest.NewRecorder()
	handler.ServeHTTP(wB2, reqB)

	if wB2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wB2.Code, wB2.Body.String())
	}
	var respB2 response.Response[miniappreview.MiniappReviewDetailResponse]
	_ = json.Unmarshal(wB2.Body.Bytes(), &respB2)
	if respB2.Code != 1 {
		t.Fatalf("expected code 1, got %+v", respB2)
	}
	if respB2.Data.IsAuthor {
		t.Fatalf("expected is_author=false for non-author")
	}
	if respB2.Data.OrderID != nil {
		t.Fatalf("SECURITY VIOLATION: non-author read order_id: %v", *respB2.Data.OrderID)
	}
	if respB2.Data.OrderItemID != nil {
		t.Fatalf("SECURITY VIOLATION: non-author read order_item_id: %v", *respB2.Data.OrderItemID)
	}
	if respB2.Data.AuditReason != nil {
		t.Fatalf("SECURITY VIOLATION: non-author read audit_reason: %v", *respB2.Data.AuditReason)
	}

	// Verify that raw JSON doesn't even contain "order_id" or "audit_reason"
	var rawMap map[string]interface{}
	_ = json.Unmarshal(wB2.Body.Bytes(), &rawMap)
	dataMap := rawMap["data"].(map[string]interface{})
	if _, ok := dataMap["order_id"]; ok {
		t.Fatalf("raw JSON data should omit order_id key for non-author")
	}
	if _, ok := dataMap["audit_reason"]; ok {
		t.Fatalf("raw JSON data should omit audit_reason key for non-author")
	}

	// Anonymous check for non-author
	if respB2.Data.UserNickname != "匿***名" {
		t.Fatalf("expected anonymous nickname '匿***名', got '%s'", respB2.Data.UserNickname)
	}
	if respB2.Data.UserAvatar != "" {
		t.Fatalf("expected empty avatar for anonymous review, got '%s'", respB2.Data.UserAvatar)
	}
}

func TestMiniappReview_ImageUniquenessAndCrossPollution(t *testing.T) {
	db, handler, tokenA, memberIDA, _, _ := setupOrderTestApp(t)

	_, itemID1, _ := createCompletedOrderWithItem(t, db, memberIDA, "img-uniq-ord-1")
	_, itemID2, _ := createCompletedOrderWithItem(t, db, memberIDA, "img-uniq-ord-2")

	// Upload one image
	pngData := createTestPNGBytes()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test_unique.png")
	_, _ = part.Write(pngData)
	_ = writer.Close()

	upReq := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/upload", body)
	upReq.Header.Set("Content-Type", writer.FormDataContentType())
	upReq.Header.Set("Authorization", "Bearer "+tokenA)
	upW := httptest.NewRecorder()
	handler.ServeHTTP(upW, upReq)

	var upResp response.Response[miniappreview.MiniappReviewUploadResponse]
	_ = json.Unmarshal(upW.Body.Bytes(), &upResp)
	fileID := upResp.Data.FileID

	// Review 1 binds fileID
	req1 := miniappreview.MiniappReviewCreateRequest{
		OrderItemID: itemID1,
		Rating:      5,
		Content:     "第一条评价使用图片",
		ImageIDs:    []int64{fileID},
		RequestID:   "img-uniq-rev-1",
	}
	b1, _ := json.Marshal(req1)
	r1 := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(b1))
	r1.Header.Set("Content-Type", "application/json")
	r1.Header.Set("Authorization", "Bearer "+tokenA)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, r1)

	var resp1 response.Response[miniappreview.MiniappReviewCreateResponse]
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	if resp1.Code != 1 {
		t.Fatalf("review 1 create failed: %s", w1.Body.String())
	}

	// Review 2 tries to reuse fileID -> MUST fail
	req2 := miniappreview.MiniappReviewCreateRequest{
		OrderItemID: itemID2,
		Rating:      4,
		Content:     "第二条评价复用图片",
		ImageIDs:    []int64{fileID},
		RequestID:   "img-uniq-rev-2",
	}
	b2, _ := json.Marshal(req2)
	r2 := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(b2))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("Authorization", "Bearer "+tokenA)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r2)

	var resp2 response.Response[any]
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Code == 1 {
		t.Fatalf("reusing image file across reviews must be rejected, got: %s", w2.Body.String())
	}
	if !strings.Contains(resp2.Message, "已被其他评价使用") {
		t.Fatalf("expected error message mentioning image used, got: %s", resp2.Message)
	}
}

func TestMiniappReview_PlainTextAndCharBoundaries(t *testing.T) {
	db, handler, tokenA, memberIDA, _, _ := setupOrderTestApp(t)

	_, itemID, prodID := createCompletedOrderWithItem(t, db, memberIDA, "raw-text-ord-1")

	// 1. Submit content with special characters: A&B <3
	rawContent := "A&B <3"
	req := miniappreview.MiniappReviewCreateRequest{
		OrderItemID: itemID,
		Rating:      5,
		Content:     rawContent,
		RequestID:   "raw-text-rev-1",
	}
	b, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp response.Response[miniappreview.MiniappReviewCreateResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 1 {
		t.Fatalf("create review failed: %s", w.Body.String())
	}
	revID := resp.Data.ID

	// Check DB row directly: MUST NOT be HTML entity escaped
	var dbReview model.ProductReview
	_ = db.First(&dbReview, revID)
	if dbReview.Content != rawContent {
		t.Fatalf("database content was escaped! expected '%s', got '%s'", rawContent, dbReview.Content)
	}

	// 2. Fetch via Detail -> must return raw text
	rDetail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/reviews/%d", revID), nil)
	rDetail.Header.Set("Authorization", "Bearer "+tokenA)
	wDetail := httptest.NewRecorder()
	handler.ServeHTTP(wDetail, rDetail)

	var detailResp response.Response[miniappreview.MiniappReviewDetailResponse]
	_ = json.Unmarshal(wDetail.Body.Bytes(), &detailResp)
	if detailResp.Data.Content != rawContent {
		t.Fatalf("detail content was entity encoded: expected '%s', got '%s'", rawContent, detailResp.Data.Content)
	}

	// 3. Test 500-rune boundary with '&' character
	_, itemID500, _ := createCompletedOrderWithItem(t, db, memberIDA, "raw-text-ord-500")
	ampContent := strings.Repeat("&", 500)
	req500 := miniappreview.MiniappReviewCreateRequest{
		OrderItemID: itemID500,
		Rating:      5,
		Content:     ampContent,
		RequestID:   "raw-text-rev-500",
	}
	b500, _ := json.Marshal(req500)
	r500 := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(b500))
	r500.Header.Set("Content-Type", "application/json")
	r500.Header.Set("Authorization", "Bearer "+tokenA)
	w500 := httptest.NewRecorder()
	handler.ServeHTTP(w500, r500)

	var resp500 response.Response[miniappreview.MiniappReviewCreateResponse]
	_ = json.Unmarshal(w500.Body.Bytes(), &resp500)
	if resp500.Code != 1 {
		t.Fatalf("500-amp content should be accepted, got error: %s", w500.Body.String())
	}

	// Check DB length
	var dbReview500 model.ProductReview
	_ = db.First(&dbReview500, resp500.Data.ID)
	if len(dbReview500.Content) != 500 {
		t.Fatalf("expected DB content length 500, got %d", len(dbReview500.Content))
	}

	// 4. A literal entity and its decoded text are distinct payloads. Reusing
	// the same request_id for the latter must report an idempotency conflict.
	_, entityItemID, _ := createCompletedOrderWithItem(t, db, memberIDA, "raw-text-entity-order")
	entityReq := miniappreview.MiniappReviewCreateRequest{
		OrderItemID: entityItemID,
		Rating:      5,
		Content:     "A&amp;B",
		RequestID:   "raw-text-entity-request",
	}
	entityBody, _ := json.Marshal(entityReq)
	entityHTTP := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(entityBody))
	entityHTTP.Header.Set("Content-Type", "application/json")
	entityHTTP.Header.Set("Authorization", "Bearer "+tokenA)
	entityW := httptest.NewRecorder()
	handler.ServeHTTP(entityW, entityHTTP)
	var entityResp response.Response[miniappreview.MiniappReviewCreateResponse]
	_ = json.Unmarshal(entityW.Body.Bytes(), &entityResp)
	if entityResp.Code != 1 {
		t.Fatalf("literal entity creation failed: %s", entityW.Body.String())
	}

	decodedReq := entityReq
	decodedReq.Content = "A&B"
	decodedBody, _ := json.Marshal(decodedReq)
	decodedHTTP := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(decodedBody))
	decodedHTTP.Header.Set("Content-Type", "application/json")
	decodedHTTP.Header.Set("Authorization", "Bearer "+tokenA)
	decodedW := httptest.NewRecorder()
	handler.ServeHTTP(decodedW, decodedHTTP)
	var decodedResp response.Response[any]
	_ = json.Unmarshal(decodedW.Body.Bytes(), &decodedResp)
	if decodedResp.Code == 1 {
		t.Fatalf("distinct literal payload must conflict, got: %s", decodedW.Body.String())
	}
	_ = prodID
}

func TestMiniappReview_IdempotencyWithImages(t *testing.T) {
	db, handler, tokenA, memberIDA, _, _ := setupOrderTestApp(t)

	_, itemID, _ := createCompletedOrderWithItem(t, db, memberIDA, "idem-img-ord-1")

	// Upload 2 images
	uploadOne := func(name string) int64 {
		pngData := createTestPNGBytes()
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", name)
		_, _ = part.Write(pngData)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+tokenA)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		var resp response.Response[miniappreview.MiniappReviewUploadResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp.Data.FileID
	}

	f1 := uploadOne("f1.png")
	f2 := uploadOne("f2.png")

	// 1. First submission with [f1, f2]
	createReq := miniappreview.MiniappReviewCreateRequest{
		OrderItemID: itemID,
		Rating:      5,
		Content:     "带图幂等测试",
		ImageIDs:    []int64{f1, f2},
		RequestID:   "idem-img-req-1",
	}
	b, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp response.Response[miniappreview.MiniappReviewCreateResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 1 {
		t.Fatalf("first creation failed: %s", w.Body.String())
	}
	firstID := resp.Data.ID

	// 2. Replay with exact same request -> must succeed with same ID
	rReplay := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(b))
	rReplay.Header.Set("Content-Type", "application/json")
	rReplay.Header.Set("Authorization", "Bearer "+tokenA)
	wReplay := httptest.NewRecorder()
	handler.ServeHTTP(wReplay, rReplay)

	var respReplay response.Response[miniappreview.MiniappReviewCreateResponse]
	_ = json.Unmarshal(wReplay.Body.Bytes(), &respReplay)
	if respReplay.Code != 1 || respReplay.Data.ID != firstID {
		t.Fatalf("expected idempotent response with ID %d, got %+v", firstID, respReplay)
	}

	// 3. Replay with same request_id but reversed image order [f2, f1] -> must be rejected
	conflictReq := createReq
	conflictReq.ImageIDs = []int64{f2, f1}
	bConflict, _ := json.Marshal(conflictReq)
	rConflict := httptest.NewRequest(http.MethodPost, "/api/miniapp/reviews/create", bytes.NewReader(bConflict))
	rConflict.Header.Set("Content-Type", "application/json")
	rConflict.Header.Set("Authorization", "Bearer "+tokenA)
	wConflict := httptest.NewRecorder()
	handler.ServeHTTP(wConflict, rConflict)

	var respConflict response.Response[any]
	_ = json.Unmarshal(wConflict.Body.Bytes(), &respConflict)
	if respConflict.Code == 1 {
		t.Fatalf("replay with reversed images must be rejected as conflict, got success: %s", wConflict.Body.String())
	}
}
