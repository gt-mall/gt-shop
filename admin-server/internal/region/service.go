package region

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"mall/admin_core/data"
)

const DataVersion = "2026.09"

type Service struct {
	version string
	etag    string
	data    RegionsData
	byCode  map[string]*RegionNode
}

// NewService creates a new Region Service using embedded data.FS.
func NewService() (*Service, error) {
	return NewServiceWithFS(data.FS)
}

// NewServiceWithFS creates a new Region Service using the provided filesystem.
func NewServiceWithFS(fsys fs.FS) (*Service, error) {
	if fsys == nil {
		return nil, errors.New("file system cannot be nil")
	}

	content, err := fs.ReadFile(fsys, "region/regions.json")
	if err != nil {
		return nil, fmt.Errorf("read region/regions.json: %w", err)
	}

	sum := sha256.Sum256(content)
	etag := fmt.Sprintf(`"%x"`, sum)

	var regions []RegionNode
	if err := json.Unmarshal(content, &regions); err != nil {
		return nil, fmt.Errorf("unmarshal region/regions.json: %w", err)
	}

	byCode := make(map[string]*RegionNode)
	var indexNode func(node *RegionNode)
	indexNode = func(node *RegionNode) {
		byCode[node.Code] = node
		for i := range node.Children {
			indexNode(&node.Children[i])
		}
	}

	for i := range regions {
		indexNode(&regions[i])
	}

	s := &Service{
		version: DataVersion,
		etag:    etag,
		data: RegionsData{
			Version: DataVersion,
			Regions: regions,
		},
		byCode: byCode,
	}

	return s, nil
}

func (s *Service) GetVersion() string {
	if s == nil {
		return ""
	}
	return s.version
}

func (s *Service) GetETag() string {
	if s == nil {
		return ""
	}
	return s.etag
}

func (s *Service) GetData() RegionsData {
	if s == nil {
		return RegionsData{}
	}
	return s.data
}

func (s *Service) GetRegions() []RegionNode {
	if s == nil {
		return nil
	}
	return s.data.Regions
}

func (s *Service) FindByCode(code string) (*RegionNode, bool) {
	if s == nil {
		return nil, false
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, false
	}
	node, ok := s.byCode[code]
	return node, ok
}

// Resolve resolves province, city, and district names to their corresponding codes.
// Resolution strictly follows the hierarchy: province -> city -> district.
func (s *Service) Resolve(province, city, district string) (*ResolvedRegion, error) {
	if s == nil {
		return nil, errors.New("region service is not initialized")
	}

	province = strings.TrimSpace(province)
	city = strings.TrimSpace(city)
	district = strings.TrimSpace(district)

	if province == "" {
		return nil, errors.New("省份名称不能为空")
	}
	if city == "" {
		return nil, errors.New("城市名称不能为空")
	}
	if district == "" {
		return nil, errors.New("区县名称不能为空")
	}

	// 1. Match province
	var provinceNode *RegionNode
	for i := range s.data.Regions {
		p := &s.data.Regions[i]
		if p.Name == province {
			provinceNode = p
			break
		}
	}
	if provinceNode == nil {
		return nil, fmt.Errorf("未找到省份: %s", province)
	}

	// 2. Match city under this province
	var cityNode *RegionNode
	for i := range provinceNode.Children {
		c := &provinceNode.Children[i]
		if c.Name == city || (c.Name == "市辖区" && (city == province || strings.TrimSuffix(city, "市") == strings.TrimSuffix(province, "市") || city == "市辖区")) {
			cityNode = c
			break
		}
	}
	if cityNode == nil {
		return nil, fmt.Errorf("在省份 [%s] 下未找到城市: %s", province, city)
	}

	// 3. Match district under this city
	var districtNode *RegionNode
	for i := range cityNode.Children {
		d := &cityNode.Children[i]
		if d.Name == district {
			districtNode = d
			break
		}
	}
	if districtNode == nil {
		return nil, fmt.Errorf("在城市 [%s/%s] 下未找到区县: %s", province, city, district)
	}

	return &ResolvedRegion{
		ProvinceCode: provinceNode.Code,
		Province:     provinceNode.Name,
		CityCode:     cityNode.Code,
		City:         cityNode.Name,
		DistrictCode: districtNode.Code,
		District:     districtNode.Name,
	}, nil
}

func cleanRegionName(name string) string {
	name = strings.TrimSpace(name)
	for _, suffix := range []string{
		"维吾尔自治区", "回族自治区", "壮族自治区", "自治区",
		"特别行政区", "省", "市", "地区", "自治州", "区", "县",
	} {
		if strings.HasSuffix(name, suffix) && len(name) > len(suffix) {
			name = strings.TrimSuffix(name, suffix)
			break
		}
	}
	return strings.TrimSpace(name)
}

func matchRegionName(input, standard string) bool {
	input = strings.TrimSpace(input)
	standard = strings.TrimSpace(standard)
	if input == "" || standard == "" {
		return true
	}
	if input == standard {
		return true
	}
	cleanIn := cleanRegionName(input)
	cleanStd := cleanRegionName(standard)
	if cleanIn != "" && cleanIn == cleanStd {
		return true
	}
	if strings.HasPrefix(standard, input) || strings.HasPrefix(input, cleanStd) {
		return true
	}
	return false
}

// ValidateAndResolveAddress validates that provinceCode, cityCode, and districtCode exist in the region hierarchy,
// checks that any passed names do not conflict with the codes, and returns the canonical resolved names.
func (s *Service) ValidateAndResolveAddress(provinceCode, provinceName, cityCode, cityName, districtCode, districtName string) (*ResolvedRegion, error) {
	if s == nil {
		return nil, errors.New("region service is not initialized")
	}

	pCode := strings.TrimSpace(provinceCode)
	cCode := strings.TrimSpace(cityCode)
	dCode := strings.TrimSpace(districtCode)
	pNameInput := strings.TrimSpace(provinceName)
	cNameInput := strings.TrimSpace(cityName)
	dNameInput := strings.TrimSpace(districtName)

	// 1. Locate province
	var provinceNode *RegionNode
	if pCode != "" {
		for i := range s.data.Regions {
			if s.data.Regions[i].Code == pCode {
				provinceNode = &s.data.Regions[i]
				break
			}
		}
		if provinceNode == nil {
			return nil, fmt.Errorf("无效的省份编码: %s", pCode)
		}
		if pNameInput != "" && !matchRegionName(pNameInput, provinceNode.Name) {
			return nil, fmt.Errorf("省份名称 [%s] 与编码 [%s(%s)] 不一致", pNameInput, pCode, provinceNode.Name)
		}
	} else if pNameInput != "" {
		for i := range s.data.Regions {
			if matchRegionName(pNameInput, s.data.Regions[i].Name) {
				provinceNode = &s.data.Regions[i]
				break
			}
		}
		if provinceNode == nil {
			return nil, fmt.Errorf("无效的省份名称: %s", pNameInput)
		}
	} else {
		return nil, errors.New("省份编码或名称不能为空")
	}

	// 2. Locate city under province
	var cityNode *RegionNode
	if cCode != "" {
		for i := range provinceNode.Children {
			if provinceNode.Children[i].Code == cCode {
				cityNode = &provinceNode.Children[i]
				break
			}
		}
		if cityNode == nil {
			return nil, fmt.Errorf("无效的城市编码 [%s] 或该城市不属于省份 [%s]", cCode, provinceNode.Name)
		}
	} else if cNameInput != "" || len(provinceNode.Children) == 1 {
		if len(provinceNode.Children) == 1 && (provinceNode.Children[0].Name == "市辖区" || matchRegionName(cNameInput, provinceNode.Name)) {
			cityNode = &provinceNode.Children[0]
		} else {
			for i := range provinceNode.Children {
				if matchRegionName(cNameInput, provinceNode.Children[i].Name) {
					cityNode = &provinceNode.Children[i]
					break
				}
			}
		}
		if cityNode == nil {
			return nil, fmt.Errorf("无效的城市名称 [%s] 或该城市不属于省份 [%s]", cNameInput, provinceNode.Name)
		}
	} else {
		return nil, errors.New("城市编码或名称不能为空")
	}

	// Municipality city check ("市辖区")
	resolvedCity := cityNode.Name
	if cityNode.Name == "市辖区" {
		if cNameInput != "" && !matchRegionName(cNameInput, provinceNode.Name) && cNameInput != "市辖区" {
			return nil, fmt.Errorf("城市名称 [%s] 与编码 [%s] 不一致", cNameInput, cityNode.Code)
		}
		if cNameInput != "" && cNameInput != "市辖区" {
			resolvedCity = cNameInput
		} else {
			resolvedCity = provinceNode.Name
		}
	} else {
		if cNameInput != "" && !matchRegionName(cNameInput, cityNode.Name) {
			return nil, fmt.Errorf("城市名称 [%s] 与编码 [%s(%s)] 不一致", cNameInput, cityNode.Code, cityNode.Name)
		}
	}

	// 3. Locate district
	var districtNode *RegionNode
	resolvedDCode := dCode
	if dCode != "" {
		for i := range cityNode.Children {
			if cityNode.Children[i].Code == dCode {
				districtNode = &cityNode.Children[i]
				break
			}
		}
		if districtNode == nil {
			return nil, fmt.Errorf("无效的区县编码 [%s] 或该区县不属于城市 [%s]", dCode, cityNode.Name)
		}
		if dNameInput != "" && !matchRegionName(dNameInput, districtNode.Name) {
			return nil, fmt.Errorf("区县名称 [%s] 与编码 [%s(%s)] 不一致", dNameInput, dCode, districtNode.Name)
		}
	} else if dNameInput != "" {
		for i := range cityNode.Children {
			if matchRegionName(dNameInput, cityNode.Children[i].Name) {
				districtNode = &cityNode.Children[i]
				resolvedDCode = districtNode.Code
				break
			}
		}
	}

	resolvedDistrict := ""
	if districtNode != nil {
		resolvedDistrict = districtNode.Name
	} else if dNameInput != "" {
		resolvedDistrict = dNameInput
	}

	return &ResolvedRegion{
		ProvinceCode: provinceNode.Code,
		Province:     provinceNode.Name,
		CityCode:     cityNode.Code,
		City:         resolvedCity,
		DistrictCode: resolvedDCode,
		District:     resolvedDistrict,
	}, nil
}
