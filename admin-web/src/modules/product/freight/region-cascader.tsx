import { useEffect, useState } from 'react';
import { Cascader } from 'antd';
import { api } from '@/framework/http';
import type { RegionNode } from '@/framework/http';

export interface CascaderOption {
  value: string;
  label: string;
  children?: CascaderOption[];
}

let cachedRegions: RegionNode[] | null = null;
let regionNameMap: Record<string, string> = {};

export async function loadRegions(): Promise<RegionNode[]> {
  if (cachedRegions) {
    return cachedRegions;
  }
  const res = await api.api.getFreightRegions();
  if (res.code === 1 && res.data) {
    cachedRegions = res.data;
    buildRegionNameMap(res.data);
    return res.data;
  }
  return [];
}

function buildRegionNameMap(nodes: RegionNode[]) {
  regionNameMap = {};
  function traverse(list: RegionNode[]) {
    for (const node of list) {
      if (node.code && node.name) {
        regionNameMap[node.code] = node.name;
      }
      if (node.children && node.children.length > 0) {
        traverse(node.children);
      }
    }
  }
  traverse(nodes);
}

export function formatRegionLabel(provinceCode: string, cityCode?: string): string {
  const pName = regionNameMap[provinceCode] || provinceCode;
  if (!cityCode || cityCode === '' || cityCode === '0') {
    return pName;
  }
  const cName = regionNameMap[cityCode] || cityCode;
  return `${pName} / ${cName}`;
}

export function buildProvinceCityCascaderOptions(nodes: RegionNode[]): CascaderOption[] {
  return nodes.map((p) => {
    const provinceOption: CascaderOption = {
      value: p.code || '',
      label: p.name || '',
    };
    if (p.children && p.children.length > 0) {
      provinceOption.children = p.children.map((c) => ({
        value: c.code || '',
        label: c.name || '',
      }));
    }
    return provinceOption;
  });
}

interface RegionCascaderProps {
  value?: [string, string] | [string];
  onChange?: (val: [string, string] | [string] | undefined) => void;
  placeholder?: string;
  changeOnSelect?: boolean;
}

export function RegionCascader({
  value,
  onChange,
  placeholder = '请选择省份/城市',
  changeOnSelect = true,
}: RegionCascaderProps) {
  const [options, setOptions] = useState<CascaderOption[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let unmounted = false;
    async function init() {
      setLoading(true);
      const data = await loadRegions();
      if (!unmounted) {
        setOptions(buildProvinceCityCascaderOptions(data));
        setLoading(false);
      }
    }
    init();
    return () => {
      unmounted = true;
    };
  }, []);

  return (
    <Cascader
      options={options}
      value={value}
      onChange={(val) => {
        if (!val || val.length === 0) {
          onChange?.(undefined);
        } else if (val.length === 1) {
          onChange?.([String(val[0])]);
        } else {
          onChange?.([String(val[0]), String(val[1])]);
        }
      }}
      placeholder={placeholder}
      changeOnSelect={changeOnSelect}
      disabled={loading}
      style={{ width: '100%' }}
    />
  );
}
