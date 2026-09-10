package order_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
	orderorder "mall/admin_core/internal/order/order"
	"mall/admin_core/pkg/response"
	"mall/admin_core/routers"

	"github.com/gin-gonic/gin"
)

const testAdminSecret = "test_admin_jwt_secret_1234567890"

func setupAdminOrderApp(t *testing.T) (*gorm.DB, http.Handler, model.SystemUser, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(
		&model.SystemUser{},
		&model.SystemRole{},
		&model.Member{},
		&model.MemberAddress{},
		&model.OrderMain{},
		&model.OrderItem{},
		&model.OrderAddress{},
		&model.OrderLog{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Create super admin user
	admin := model.SystemUser{
		Username: "superadmin",
		Nickname: "超级管理员",
		Status:   1,
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	role := model.SystemRole{
		Code:   auth.SuperAdminRoleCode,
		Name:   "超级管理员",
		Status: 1,
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	// Create casbin enforcer
	m := casbinmodel.NewModel()
	m.AddDef("r", "r", "sub, perm")
	m.AddDef("p", "p", "sub, perm")
	m.AddDef("g", "g", "_, _")
	m.AddDef("e", "e", "some(where (p_eft == allow))")
	m.AddDef("m", "m", "g(r_sub, p_sub) && r_perm == p_perm")
	enforcer, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	if _, err := enforcer.AddRoleForUser(auth.UserSubject(admin.ID), auth.SuperAdminRoleCode); err != nil {
		t.Fatalf("add super admin role: %v", err)
	}

	authService := auth.NewService(db, enforcer, testAdminSecret, 24*time.Hour)

	// Issue admin token
	adminClaims := jwt.RegisteredClaims{
		Issuer:    "mall-admin-core",
		Subject:   strconv.FormatInt(admin.ID, 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, adminClaims).SignedString([]byte(testAdminSecret))
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}

	bConfig := bootstrap.Config{
		JWTSecret:     "miniapp_jwt_secret_12345",
		JWTExpireHour: 24,
	}
	router := routers.NewRouter(bConfig, db, authService)

	return db, router, admin, token
}

func createAdminOrderData(t *testing.T, db *gorm.DB) (orderID int64, orderNo string) {
	t.Helper()
	now := time.Now()

	// Create member
	mobile := "13912345678"
	member := model.Member{
		Nickname: "测试买家",
		Mobile:   &mobile,
		Status:   1,
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	// Create order main
	orderNo = "202609101000000001"
	orderMain := model.OrderMain{
		OrderNo:       orderNo,
		MemberID:      member.ID,
		RequestID:     "req-admin-001",
		Status:        model.OrderStatusPendingShipment,
		TotalAmount:   "199.90",
		PayAmount:     "199.90",
		FreightAmount: "0.00",
		Remark:        "加急配送",
		PaidAt:        nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := db.Create(&orderMain).Error; err != nil {
		t.Fatalf("create order_main: %v", err)
	}
	orderID = orderMain.ID

	// Create order address
	orderAddr := model.OrderAddress{
		OrderID:      orderID,
		Name:         "王小二",
		Mobile:       "13800001111",
		ProvinceCode: "310000",
		Province:     "上海市",
		CityCode:     "310100",
		City:         "上海市",
		DistrictCode: "310104",
		District:     "徐汇区",
		Address:      "宜山路 888 号",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(&orderAddr).Error; err != nil {
		t.Fatalf("create order_address: %v", err)
	}

	// Create order item
	orderItem := model.OrderItem{
		OrderID:     orderID,
		ProductID:   1,
		SKUID:       10,
		ProductName: "降噪无线耳机",
		SKUCode:     "EAR-BLK",
		UnitPrice:   "199.90",
		Quantity:    1,
		TotalAmount: "199.90",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.Create(&orderItem).Error; err != nil {
		t.Fatalf("create order_item: %v", err)
	}

	return orderID, orderNo
}

func TestAdminOrderService(t *testing.T) {
	db, _, admin, _ := setupAdminOrderApp(t)
	orderID, orderNo := createAdminOrderData(t, db)
	svc := orderorder.NewService(db)
	ctx := context.Background()

	t.Run("List with keyword matching order_no", func(t *testing.T) {
		rows, total, err := svc.List(ctx, orderorder.OrderListRequest{
			Keyword: orderNo,
		})
		if err != nil {
			t.Fatalf("list err: %v", err)
		}
		if total != 1 || len(rows) != 1 {
			t.Fatalf("expected 1 order, got total=%d, len=%d", total, len(rows))
		}
		if rows[0].OrderNo != orderNo {
			t.Errorf("expected %s, got %s", orderNo, rows[0].OrderNo)
		}
		if rows[0].MemberNickname != "测试买家" {
			t.Errorf("expected member nickname 测试买家, got %s", rows[0].MemberNickname)
		}
		if rows[0].PayAmount != "199.90" {
			t.Errorf("expected 199.90, got %s", rows[0].PayAmount)
		}
	})

	t.Run("List with keyword matching recipient name", func(t *testing.T) {
		rows, total, err := svc.List(ctx, orderorder.OrderListRequest{
			Keyword: "王小二",
		})
		if err != nil {
			t.Fatalf("list err: %v", err)
		}
		if total != 1 || len(rows) != 1 {
			t.Errorf("expected 1 order matching 王小二, got total=%d", total)
		}
	})

	t.Run("Detail returns full details including address, items, logs", func(t *testing.T) {
		detail, err := svc.Detail(ctx, orderID)
		if err != nil {
			t.Fatalf("detail err: %v", err)
		}
		if detail.ID != orderID || detail.OrderNo != orderNo {
			t.Errorf("unexpected detail ID or OrderNo")
		}
		if detail.PaidAt != nil {
			t.Errorf("expected paid_at == nil for unpaid order, got %v", detail.PaidAt)
		}
		if detail.Address == nil || detail.Address.Name != "王小二" {
			t.Errorf("expected address name 王小二")
		}
		if len(detail.Items) != 1 || detail.Items[0].ProductName != "降噪无线耳机" {
			t.Errorf("expected 1 item with name 降噪无线耳机")
		}
	})

	t.Run("Ship order successfully updates status and records log", func(t *testing.T) {
		err := svc.Ship(ctx, admin, orderorder.OrderShipRequest{
			ID:             orderID,
			ExpressCompany: "顺丰速运",
			ExpressNo:      "SF987654321",
		})
		if err != nil {
			t.Fatalf("ship order err: %v", err)
		}

		// Verify order status is 30 (pending_receipt)
		detail, err := svc.Detail(ctx, orderID)
		if err != nil {
			t.Fatalf("detail after ship err: %v", err)
		}
		if detail.Status != model.OrderStatusPendingReceipt {
			t.Errorf("expected status=30 (pending_receipt), got %d", detail.Status)
		}
		if detail.ExpressCompany == nil || *detail.ExpressCompany != "顺丰速运" {
			t.Errorf("expected express company 顺丰速运")
		}
		if detail.ExpressNo == nil || *detail.ExpressNo != "SF987654321" {
			t.Errorf("expected express no SF987654321")
		}
		if detail.ShippedAt == nil {
			t.Errorf("expected shipped_at to be set")
		}

		// Verify order log
		if len(detail.Logs) != 1 || detail.Logs[0].Action != "ship" {
			t.Errorf("expected 1 ship log, got %v", detail.Logs)
		}
		if detail.Logs[0].FromStatus != model.OrderStatusPendingShipment || detail.Logs[0].ToStatus != model.OrderStatusPendingReceipt {
			t.Errorf("expected ship log from 20 to 30, got from=%d to=%d", detail.Logs[0].FromStatus, detail.Logs[0].ToStatus)
		}
		if detail.Logs[0].FromStatusText != "待发货" || detail.Logs[0].ToStatusText != "待收货" {
			t.Errorf("expected text 待发货 -> 待收货, got %s -> %s", detail.Logs[0].FromStatusText, detail.Logs[0].ToStatusText)
		}
	})

	t.Run("Cannot ship already shipped order", func(t *testing.T) {
		err := svc.Ship(ctx, admin, orderorder.OrderShipRequest{
			ID:             orderID,
			ExpressCompany: "圆通快递",
			ExpressNo:      "YT123456",
		})
		if err == nil {
			t.Errorf("expected error when re-shipping already shipped order")
		}
	})

	t.Run("Cannot ship cancelled or completed orders", func(t *testing.T) {
		// Cancelled order
		cOrder := model.OrderMain{
			OrderNo:       "202609101000000099",
			MemberID:      1,
			RequestID:     "req-cancelled-01",
			Status:        model.OrderStatusCancelled,
			TotalAmount:   "50.00",
			PayAmount:     "50.00",
			FreightAmount: "0.00",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		_ = db.Create(&cOrder)
		err := svc.Ship(ctx, admin, orderorder.OrderShipRequest{
			ID:             cOrder.ID,
			ExpressCompany: "圆通快递",
			ExpressNo:      "YT123456",
		})
		if err == nil {
			t.Errorf("expected error when shipping cancelled order")
		}

		// Empty express params
		pOrder := model.OrderMain{
			OrderNo:       "202609101000000100",
			MemberID:      1,
			RequestID:     "req-pending-ship-01",
			Status:        model.OrderStatusPendingShipment,
			TotalAmount:   "50.00",
			PayAmount:     "50.00",
			FreightAmount: "0.00",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		_ = db.Create(&pOrder)
		errEmpty := svc.Ship(ctx, admin, orderorder.OrderShipRequest{
			ID:             pOrder.ID,
			ExpressCompany: "",
			ExpressNo:      "",
		})
		if errEmpty == nil {
			t.Errorf("expected error when shipping with empty express info")
		}
	})

	t.Run("Detail returns error for non-existent order", func(t *testing.T) {
		_, err := svc.Detail(ctx, 9999999)
		if err == nil {
			t.Errorf("expected error for non-existent order detail")
		}
	})
}

func TestAdminOrderHTTPAPI(t *testing.T) {
	db, router, _, token := setupAdminOrderApp(t)
	orderID, _ := createAdminOrderData(t, db)

	t.Run("POST /api/order/orders/list", func(t *testing.T) {
		body, _ := json.Marshal(orderorder.OrderListRequest{
			Page:     1,
			PageSize: 10,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/order/orders/list", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp response.Response[[]orderorder.OrderListRow]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 || len(resp.Data) != 1 {
			t.Fatalf("expected 1 order in list, got %d: %s", len(resp.Data), resp.Message)
		}
	})

	t.Run("POST /api/order/orders/detail", func(t *testing.T) {
		body, _ := json.Marshal(orderorder.OrderDetailRequest{ID: orderID})
		req := httptest.NewRequest(http.MethodPost, "/api/order/orders/detail", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[orderorder.OrderDetailResponse]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 || resp.Data.ID != orderID {
			t.Fatalf("expected detail for order %d, got msg=%s", orderID, resp.Message)
		}
	})

	t.Run("POST /api/order/orders/ship", func(t *testing.T) {
		body, _ := json.Marshal(orderorder.OrderShipRequest{
			ID:             orderID,
			ExpressCompany: "中通快递",
			ExpressNo:      "ZT123456789",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/order/orders/ship", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[any]
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != 1 {
			t.Fatalf("ship failed: %s", resp.Message)
		}
	})
}
