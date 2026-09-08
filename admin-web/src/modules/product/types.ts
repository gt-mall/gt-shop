import type { ProductCategoryOption, ProductCategoryRow, ProductSKURequest, ProductSKUValue, ProductSpec } from '@/framework/http';

export function getSpecValueKey(specID: number, valueID: number): string {
  return `${specID}:${valueID}`;
}

export interface CategoryTreeNode extends ProductCategoryRow {
  key: number;
  children?: CategoryTreeNode[];
}

export interface CategoryOptionNode extends ProductCategoryOption {
  key: number;
  value: number;
  title: string;
  disabled?: boolean;
  children?: CategoryOptionNode[];
}

// Convert flat category array to tree structure
export function buildCategoryTree(list: ProductCategoryRow[]): CategoryTreeNode[] {
  const map = new Map<number, CategoryTreeNode>();
  const roots: CategoryTreeNode[] = [];

  for (const item of list) {
    const id = item.id ?? 0;
    map.set(id, { ...item, key: id, children: [] });
  }

  for (const item of list) {
    const id = item.id ?? 0;
    const parentId = item.parent_id ?? 0;
    const node = map.get(id)!;
    if (parentId > 0 && map.has(parentId)) {
      map.get(parentId)!.children!.push(node);
    } else {
      roots.push(node);
    }
  }

  const cleanEmptyChildren = (nodes: CategoryTreeNode[]) => {
    for (const n of nodes) {
      if (n.children && n.children.length === 0) {
        delete n.children;
      } else if (n.children) {
        cleanEmptyChildren(n.children);
      }
    }
  };
  cleanEmptyChildren(roots);

  return roots;
}

// Convert flat options array to TreeSelect options
export function buildCategoryOptionTree(
  list: ProductCategoryOption[],
  disabledIDs: Set<number> = new Set(),
): CategoryOptionNode[] {
  const map = new Map<number, CategoryOptionNode>();
  const roots: CategoryOptionNode[] = [];

  for (const item of list) {
    const id = item.id ?? 0;
    map.set(id, {
      ...item,
      key: id,
      value: id,
      title: (item.name ?? '') + (item.status === 0 ? ' (已禁用)' : ''),
      disabled: disabledIDs.has(id),
      children: [],
    });
  }

  for (const item of list) {
    const id = item.id ?? 0;
    const parentId = item.parent_id ?? 0;
    const node = map.get(id)!;
    if (parentId > 0 && map.has(parentId)) {
      map.get(parentId)!.children!.push(node);
    } else {
      roots.push(node);
    }
  }

  const cleanEmptyChildren = (nodes: CategoryOptionNode[]) => {
    for (const n of nodes) {
      if (n.children && n.children.length === 0) {
        delete n.children;
      } else if (n.children) {
        cleanEmptyChildren(n.children);
      }
    }
  };
  cleanEmptyChildren(roots);

  return roots;
}

// Format money string to standard 2-decimal format without float conversions
export function normalizeMoney(value: string | number | null | undefined): string {
  if (value === null || value === undefined) return '0.00';
  const s = String(value).trim();
  if (!s) return '0.00';

  const parts = s.split('.');
  const intPart = parts[0] || '0';
  let decPart = parts[1] || '';

  if (decPart.length === 0) {
    decPart = '00';
  } else if (decPart.length === 1) {
    decPart += '0';
  } else if (decPart.length > 2) {
    decPart = decPart.slice(0, 2);
  }

  return `${intPart}.${decPart}`;
}

// Build canonical SKU combo key "1:1|2:4" for matching
export function getSKUComboKey(specValues: ProductSKUValue[]): string {
  const sorted = [...specValues].sort((a, b) => (a.spec_id ?? 0) - (b.spec_id ?? 0));
  return sorted.map((v) => `${v.spec_id ?? 0}:${v.value_id ?? 0}`).join('|');
}

// Generate SKU combinations via cartesian product of valid spec values
export function generateSKUCombinations(
  specs: ProductSpec[],
  existingSKUs: ProductSKURequest[] = [],
): { skus: ProductSKURequest[]; totalCount: number } {
  const validSpecs = specs.filter((s) => s.values && s.values.length > 0);
  if (validSpecs.length === 0) {
    return { skus: [], totalCount: 0 };
  }

  let totalCount = 1;
  for (const s of validSpecs) {
    totalCount *= s.values!.length;
  }

  if (totalCount > 200) {
    return { skus: [], totalCount };
  }

  let combos: ProductSKUValue[][] = [[]];
  for (const s of validSpecs) {
    const nextCombos: ProductSKUValue[][] = [];
    for (const prefix of combos) {
      for (const val of s.values!) {
        nextCombos.push([...prefix, { spec_id: s.id ?? 0, value_id: val.id ?? 0 }]);
      }
    }
    combos = nextCombos;
  }

  const existingMap = new Map<string, ProductSKURequest>();
  for (const sku of existingSKUs) {
    if (sku.spec_values) {
      const key = getSKUComboKey(sku.spec_values);
      existingMap.set(key, sku);
    }
  }

  const result: ProductSKURequest[] = [];
  for (let i = 0; i < combos.length; i++) {
    const c = combos[i];
    const key = getSKUComboKey(c);
    if (existingMap.has(key)) {
      result.push(existingMap.get(key)!);
    } else {
      result.push({
        id: 0,
        spec_values: c,
        price: '0.00',
        market_price: '0.00',
        cost_price: '0.00',
        sort: i,
        status: 1,
      });
    }
  }

  return { skus: result, totalCount };
}
