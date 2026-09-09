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
