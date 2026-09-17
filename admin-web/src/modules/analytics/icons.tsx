import React from 'react';
import {
  ChartNoAxesCombined,
  LayoutDashboard,
  ChartLine,
  PackageSearch,
  Users,
  WalletCards,
} from 'lucide-react';

/**
 * 数据财务模块菜单图标映射
 * 统一尺寸 size = 18, strokeWidth = 1.8，颜色继承当前主题
 */
export const financeMenuIcons = {
  finance: <ChartNoAxesCombined size={18} strokeWidth={1.8} />,
  overview: <LayoutDashboard size={18} strokeWidth={1.8} />,
  trade: <ChartLine size={18} strokeWidth={1.8} />,
  product: <PackageSearch size={18} strokeWidth={1.8} />,
  member: <Users size={18} strokeWidth={1.8} />,
  flow: <WalletCards size={18} strokeWidth={1.8} />,
};
