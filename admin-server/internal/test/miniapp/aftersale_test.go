package miniapp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/gorm"

	miniappaftersale "mall/admin_core/internal/miniapp/aftersale"
	miniapporder "mall/admin_core/internal/miniapp/order"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/internal/model"
	orderaftersale "mall/admin_core/internal/order/aftersale"
	"mall/admin_core/pkg/response"
)

func createMiniappTestPaidOrder(t *testing.T, db *gorm.DB, memberID int64, reqID string) int64 {
	t.Helper()
	_, skuID, addrID := createOrderTestData(t, db, memberID)
	order, err := miniapporder.NewService(db).Create(context.Background(), model.Member{ID: memberID}, miniapporder.OrderCreateRequest{
		RequestID: reqID,
		AddressID: addrID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("create test order failed: %v", err)
	}

	paySvc := miniapppayment.NewService(db)
	_, err = paySvc.MockPay(context.Background(), memberID, miniapppayment.MockPayRequest{OrderID: order.ID})
	if err != nil {
		t.Fatalf("mock pay failed: %v", err)
	}
	return order.ID
}

func TestMiniappAfterSaleHTTPEndpoints(t *testing.T) {
	db, handler, tokenA, memberIDA, tokenB, memberIDB := setupOrderTestApp(t)
	_ = memberIDB
	orderID := createMiniappTestPaidOrder(t, db, memberIDA, "as-miniapp-req-1")

	// 1. Unauthorized request
	applyReq := miniappaftersale.MiniappApplyAfterSaleRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeRefundOnly,
		Reason:    "不想要了",
		RequestID: "apply-req-1",
	}
	body, _ := json.Marshal(applyReq)
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var unauthResp response.Response[any]
	_ = json.Unmarshal(w.Body.Bytes(), &unauthResp)
	if unauthResp.Code != -1 {
		t.Fatalf("expected code -1 (unauthorized), got %d", unauthResp.Code)
	}

	// 2. Member A applies after-sale
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var applyResp response.Response[miniappaftersale.MiniappAfterSaleDetailResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &applyResp)
	if applyResp.Code != 1 || applyResp.Data.OrderID != orderID {
		t.Fatalf("apply failed: %+v", applyResp)
	}
	afterSaleID := applyResp.Data.ID

	// 3. Member B tries to view Member A's after-sale -> fails
	detailReq := miniappaftersale.MiniappAfterSaleDetailRequest{AfterSaleID: afterSaleID}
	bodyDetail, _ := json.Marshal(detailReq)
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/detail", bytes.NewReader(bodyDetail))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenB)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var forbiddenDetailResp response.Response[any]
	_ = json.Unmarshal(w.Body.Bytes(), &forbiddenDetailResp)
	if forbiddenDetailResp.Code != 0 {
		t.Fatalf("expected failure for other member detail, got: %+v", forbiddenDetailResp)
	}

	// 4. Member A views detail -> succeeds
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/detail", bytes.NewReader(bodyDetail))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var detailResp response.Response[miniappaftersale.MiniappAfterSaleDetailResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &detailResp)
	if detailResp.Code != 1 || detailResp.Data.ID != afterSaleID {
		t.Fatalf("expected member A detail success, got: %+v", detailResp)
	}

	// 5. Member A lists after-sales
	listReq := miniappaftersale.MiniappAfterSaleListRequest{Page: 1, PageSize: 10}
	bodyList, _ := json.Marshal(listReq)
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/list", bytes.NewReader(bodyList))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var listResp response.Response[[]miniappaftersale.MiniappAfterSaleListItemResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if listResp.Code != 1 || len(listResp.Data) != 1 {
		t.Fatalf("expected 1 item for member A list, got: %+v", listResp)
	}

	// Member B list is empty
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/list", bytes.NewReader(bodyList))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenB)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var listBResp response.Response[[]miniappaftersale.MiniappAfterSaleListItemResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &listBResp)
	if listBResp.Code != 1 || len(listBResp.Data) != 0 {
		t.Fatalf("expected 0 items for member B list, got: %+v", listBResp)
	}

	// 6. Member B tries to cancel Member A's after-sale -> fails
	cancelReq := miniappaftersale.MiniappCancelAfterSaleRequest{AfterSaleID: afterSaleID}
	bodyCancel, _ := json.Marshal(cancelReq)
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/cancel", bytes.NewReader(bodyCancel))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenB)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var cancelFailResp response.Response[any]
	_ = json.Unmarshal(w.Body.Bytes(), &cancelFailResp)
	if cancelFailResp.Code != 0 {
		t.Fatalf("expected cancel failure for member B, got: %+v", cancelFailResp)
	}

	// 7. Member A cancels after-sale -> succeeds
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/cancel", bytes.NewReader(bodyCancel))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var cancelOkResp response.Response[any]
	_ = json.Unmarshal(w.Body.Bytes(), &cancelOkResp)
	if cancelOkResp.Code != 1 {
		t.Fatalf("expected cancel success, got: %+v", cancelOkResp)
	}

	// Verify status is 70
	var asDb model.AfterSale
	_ = db.Where("id = ?", afterSaleID).First(&asDb)
	if asDb.Status != model.AfterSaleStatusCancelled {
		t.Fatalf("expected status 70, got %d", asDb.Status)
	}
}

func TestMiniappReturnLogisticsEndpoint(t *testing.T) {
	db, handler, tokenA, memberIDA, _, _ := setupOrderTestApp(t)

	// Create shipped order (status 30) for ReturnRefund
	orderID := createMiniappTestPaidOrder(t, db, memberIDA, "as-miniapp-return-1")
	now := time.Now()
	company := "顺丰速运"
	no := "SF111"
	db.Model(&model.OrderMain{}).Where("id = ?", orderID).Updates(map[string]any{
		"status":          model.OrderStatusPendingReceipt,
		"express_company": &company,
		"express_no":      &no,
		"shipped_at":      &now,
	})

	// Member A applies ReturnRefund
	applyReq := miniappaftersale.MiniappApplyAfterSaleRequest{
		OrderID:   orderID,
		Type:      model.AfterSaleTypeReturnRefund,
		Reason:    "不喜欢想退货",
		RequestID: "apply-return-1",
	}
	body, _ := json.Marshal(applyReq)
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var applyResp response.Response[miniappaftersale.MiniappAfterSaleDetailResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &applyResp)
	if applyResp.Code != 1 {
		t.Fatalf("apply return failed: %+v", applyResp)
	}
	afterSaleID := applyResp.Data.ID

	// Try to submit logistics while still in status 10 (PendingReview) -> fails
	logisticsReq := miniappaftersale.MiniappReturnLogisticsRequest{
		AfterSaleID: afterSaleID,
		Company:     "中通快递",
		TrackingNo:  "ZT123456",
	}
	bodyLogistics, _ := json.Marshal(logisticsReq)
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/return-logistics", bytes.NewReader(bodyLogistics))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var earlyLogisticsResp response.Response[any]
	_ = json.Unmarshal(w.Body.Bytes(), &earlyLogisticsResp)
	if earlyLogisticsResp.Code != 0 {
		t.Fatalf("expected early logistics submit to fail, got: %+v", earlyLogisticsResp)
	}

	// Admin approves return: 10 -> 20
	asCore := orderaftersale.NewService(db, nil, nil)
	adminUser := model.SystemUser{ID: 1, Username: "admin"}
	err := asCore.AdminApprove(context.Background(), adminUser, orderaftersale.AdminApproveRequest{
		ID:             afterSaleID,
		ReturnReceiver: "仓库老王",
		ReturnPhone:    "13800001111",
		ReturnAddress:  "北京市海淀区物流园3号库",
	})
	if err != nil {
		t.Fatalf("admin approve return failed: %v", err)
	}

	// Submit logistics in status 20 -> succeeds (20 -> 30)
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/return-logistics", bytes.NewReader(bodyLogistics))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var logisticsResp response.Response[any]
	_ = json.Unmarshal(w.Body.Bytes(), &logisticsResp)
	if logisticsResp.Code != 1 {
		t.Fatalf("expected logistics submit to succeed, got: %+v", logisticsResp)
	}

	var asDb model.AfterSale
	_ = db.Where("id = ?", afterSaleID).First(&asDb)
	if asDb.Status != model.AfterSaleStatusWaitingMerchantReceive {
		t.Fatalf("expected status 30, got %d", asDb.Status)
	}
	if asDb.BuyerReturnCompany != "中通快递" || asDb.BuyerReturnNo != "ZT123456" {
		t.Fatalf("buyer return info not saved properly: %+v", asDb)
	}
}

func TestMiniappAfterSaleProofUploadAndDetail(t *testing.T) {
	db, handler, tokenA, memberIDA, tokenB, _ := setupOrderTestApp(t)
	orderID := createMiniappTestPaidOrder(t, db, memberIDA, "as-proof-upload-order")

	// 1. Member A uploads proof image
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{R: 200, G: 50, B: 50, A: 255})
	var pngBuf bytes.Buffer
	_ = png.Encode(&pngBuf, img)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", "proof.png")
	_, _ = part.Write(pngBuf.Bytes())
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var imgResp response.Response[miniappaftersale.MiniappAfterSaleProofUploadResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &imgResp)
	if imgResp.Code != 1 || imgResp.Data.ID <= 0 || imgResp.Data.FileType != "image" {
		t.Fatalf("unexpected upload image response: %+v", imgResp)
	}
	proofImgID := imgResp.Data.ID

	// 2. Member A uploads proof video
	buf.Reset()
	mw = multipart.NewWriter(&buf)
	part, _ = mw.CreateFormFile("file", "unboxing.mp4")
	_, _ = part.Write([]byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00isommp42\x00\x00\x00\x08free"))
	_ = mw.Close()

	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var videoResp response.Response[miniappaftersale.MiniappAfterSaleProofUploadResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &videoResp)
	if videoResp.Code != 1 || videoResp.Data.ID <= 0 || videoResp.Data.FileType != "video" {
		t.Fatalf("unexpected upload video response: %+v", videoResp)
	}
	proofVideoID := videoResp.Data.ID

	// 3. Member A applies after-sale with both proof files
	applyReq := miniappaftersale.MiniappApplyAfterSaleRequest{
		OrderID:      orderID,
		Type:         model.AfterSaleTypeRefundOnly,
		Reason:       "商品损坏及拆箱视频",
		ProofFileIDs: []int64{proofImgID, proofVideoID},
		RequestID:    "req-apply-with-proofs",
	}
	applyBody, _ := json.Marshal(applyReq)
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/apply", bytes.NewReader(applyBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var applyResult response.Response[miniappaftersale.MiniappAfterSaleDetailResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &applyResult)
	if applyResult.Code != 1 {
		t.Fatalf("apply failed: %+v", applyResult)
	}
	asID := applyResult.Data.ID

	// 4. Member A views detail -> verifies proof_files with signed URLs
	detailReq := miniappaftersale.MiniappAfterSaleDetailRequest{AfterSaleID: asID}
	detailBody, _ := json.Marshal(detailReq)
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/detail", bytes.NewReader(detailBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var detailResult response.Response[miniappaftersale.MiniappAfterSaleDetailResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &detailResult)
	if detailResult.Code != 1 || len(detailResult.Data.ProofFiles) != 2 {
		t.Fatalf("expected 2 proof files in detail, got: %+v", detailResult.Data)
	}
	if detailResult.Data.ProofFiles[0].FileType != "image" || detailResult.Data.ProofFiles[1].FileType != "video" {
		t.Fatalf("unexpected proof file types: %+v", detailResult.Data.ProofFiles)
	}

	// 5. Member B cannot access Member A's after-sale
	req = httptest.NewRequest(http.MethodPost, "/api/miniapp/after-sales/detail", bytes.NewReader(detailBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenB)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var bResult response.Response[any]
	_ = json.Unmarshal(w.Body.Bytes(), &bResult)
	if bResult.Code != 0 {
		t.Fatalf("expected Member B access to be denied, got: %+v", bResult)
	}
}
