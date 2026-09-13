package region_test

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mall/admin_core/bootstrap"
	"mall/admin_core/data"
	"mall/admin_core/internal/auth"
	miniappregion "mall/admin_core/internal/miniapp/region"
	"mall/admin_core/internal/region"
	"mall/admin_core/routers"
)

func TestRegionEmbeddedAndHierarchy(t *testing.T) {
	// 1. embedded FS 能读取 region/regions.json
	content, err := fs.ReadFile(data.FS, "region/regions.json")
	if err != nil {
		t.Fatalf("read embedded region/regions.json: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("embedded region/regions.json is empty")
	}

	// 2. JSON 正常解析
	svc, err := region.NewService()
	if err != nil {
		t.Fatalf("initialize region service: %v", err)
	}

	regions := svc.GetRegions()
	if len(regions) == 0 {
		t.Fatal("expected non-empty regions list")
	}

	// 3. 数据只有省市区三级 & 4. code 是 string
	for _, p := range regions {
		if p.Code == "" || p.Name == "" {
			t.Fatalf("province has empty code or name: %+v", p)
		}
		for _, c := range p.Children {
			if c.Code == "" || c.Name == "" {
				t.Fatalf("city has empty code or name under %s: %+v", p.Name, c)
			}
			for _, d := range c.Children {
				if d.Code == "" || d.Name == "" {
					t.Fatalf("district has empty code or name under %s/%s: %+v", p.Name, c.Name, d)
				}
				if len(d.Children) > 0 {
					t.Fatalf("district %s has children (exceeds 3 levels)", d.Name)
				}
			}
		}
	}

	// 5. 山东省存在
	var shandong *region.RegionNode
	for i := range regions {
		if regions[i].Name == "山东省" {
			shandong = &regions[i]
			break
		}
	}
	if shandong == nil {
		t.Fatal("山东省 not found in regions")
	}
	if shandong.Code != "370000" {
		t.Errorf("expected 山东省 code '370000', got %q", shandong.Code)
	}

	// 6. 济南市属于山东省
	var jinan *region.RegionNode
	for i := range shandong.Children {
		if shandong.Children[i].Name == "济南市" {
			jinan = &shandong.Children[i]
			break
		}
	}
	if jinan == nil {
		t.Fatal("济南市 not found under 山东省")
	}
	if jinan.Code != "370100" {
		t.Errorf("expected 济南市 code '370100', got %q", jinan.Code)
	}

	// 7. 历下区属于济南市
	var lixia *region.RegionNode
	for i := range jinan.Children {
		if jinan.Children[i].Name == "历下区" {
			lixia = &jinan.Children[i]
			break
		}
	}
	if lixia == nil {
		t.Fatal("历下区 not found under 济南市")
	}
	if lixia.Code != "370102" {
		t.Errorf("expected 历下区 code '370102', got %q", lixia.Code)
	}

	// 8. FindByCode 正常
	nodeP, ok := svc.FindByCode("370000")
	if !ok || nodeP == nil || nodeP.Name != "山东省" {
		t.Errorf("FindByCode('370000') failed, got: %+v, ok=%v", nodeP, ok)
	}
	nodeC, ok := svc.FindByCode("370100")
	if !ok || nodeC == nil || nodeC.Name != "济南市" {
		t.Errorf("FindByCode('370100') failed, got: %+v, ok=%v", nodeC, ok)
	}
	nodeD, ok := svc.FindByCode("370102")
	if !ok || nodeD == nil || nodeD.Name != "历下区" {
		t.Errorf("FindByCode('370102') failed, got: %+v, ok=%v", nodeD, ok)
	}
	if _, ok := svc.FindByCode("999999"); ok {
		t.Error("FindByCode('999999') expected false, got true")
	}

	// 9. Resolve("山东省","济南市","历下区") 正常
	resolved, err := svc.Resolve("山东省", "济南市", "历下区")
	if err != nil {
		t.Fatalf("Resolve('山东省','济南市','历下区') failed: %v", err)
	}
	if resolved.ProvinceCode != "370000" || resolved.Province != "山东省" {
		t.Errorf("unexpected province in resolved: %+v", resolved)
	}
	if resolved.CityCode != "370100" || resolved.City != "济南市" {
		t.Errorf("unexpected city in resolved: %+v", resolved)
	}
	if resolved.DistrictCode != "370102" || resolved.District != "历下区" {
		t.Errorf("unexpected district in resolved: %+v", resolved)
	}

	// 10. 错误父子组合无法 Resolve
	if _, err := svc.Resolve("山东省", "青岛市", "历下区"); err == nil {
		t.Error("expected error resolving 历下区 under 青岛市, but got nil")
	}
	if _, err := svc.Resolve("广东省", "济南市", "历下区"); err == nil {
		t.Error("expected error resolving 济南市 under 广东省, but got nil")
	}
	if _, err := svc.Resolve("火星省", "济南市", "历下区"); err == nil {
		t.Error("expected error resolving unknown province, but got nil")
	}
}

func TestMiniappRegionsEndpoint(t *testing.T) {
	router := routers.NewRouter(bootstrap.Config{}, nil, auth.NewService(nil, nil, "", time.Hour))

	// 11. GET /api/miniapp/regions 无需登录
	req := httptest.NewRequest(http.MethodGet, "/api/miniapp/regions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	// 12. 返回 version
	var resp struct {
		Code    int                           `json:"code"`
		Message string                        `json:"message"`
		Total   int                           `json:"total"`
		Data    miniappregion.MiniappRegionsData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 1 {
		t.Fatalf("code = %d, want 1", resp.Code)
	}
	if resp.Data.Version != "2026.09" {
		t.Errorf("expected version '2026.09', got %q", resp.Data.Version)
	}
	if len(resp.Data.Regions) == 0 {
		t.Fatal("expected regions to be non-empty")
	}

	// 13. 返回 Cache-Control
	cacheControl := rec.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=86400" {
		t.Errorf("Cache-Control = %q, want 'public, max-age=86400'", cacheControl)
	}

	// 14. ETag 正常
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag header is missing or empty")
	}

	// 15. If-None-Match 命中时返回 304
	req304 := httptest.NewRequest(http.MethodGet, "/api/miniapp/regions", nil)
	req304.Header.Set("If-None-Match", etag)
	rec304 := httptest.NewRecorder()
	router.ServeHTTP(rec304, req304)

	if rec304.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304 Not Modified when If-None-Match matches", rec304.Code)
	}
	if rec304.Body.Len() > 0 {
		t.Errorf("expected empty body on 304, got %d bytes", rec304.Body.Len())
	}
}

func TestRegionNoRedisUsage(t *testing.T) {
	svc, err := region.NewService()
	if err != nil {
		t.Fatalf("initialize region service: %v", err)
	}

	data1 := svc.GetData()
	if len(data1.Regions) == 0 {
		t.Fatalf("expected non-empty regions")
	}
	if svc.GetETag() == "" {
		t.Fatalf("expected non-empty ETag")
	}
	if svc.GetVersion() != region.DataVersion {
		t.Fatalf("expected version %s, got %s", region.DataVersion, svc.GetVersion())
	}
}
