import React from 'react';
import { Card, Col, Empty, Row, Typography, theme } from 'antd';
import {
  AppstoreOutlined,
  FileImageOutlined,
  PlusCircleOutlined,
  SettingOutlined,
  ShopOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { usePermission } from '@/framework/store';

const { Text } = Typography;

interface QuickLinkItem {
  key: string;
  name: string;
  path: string;
  permission: string;
  icon: React.ReactNode;
  desc: string;
}

export const QuickLinks: React.FC = () => {
  const { token } = theme.useToken();
  const navigate = useNavigate();
  const hasPermission = usePermission();

  const allLinks: QuickLinkItem[] = [
    {
      key: 'product-create',
      name: '新增商品',
      path: '/product/product/create',
      permission: 'product:product:create',
      icon: <PlusCircleOutlined style={{ fontSize: 20, color: token.colorPrimary }} />,
      desc: '发布新商品上架',
    },
    {
      key: 'product-list',
      name: '商品管理',
      path: '/product/product',
      permission: 'product:product:list',
      icon: <ShopOutlined style={{ fontSize: 20, color: '#722ed1' }} />,
      desc: '维护商品与规格',
    },
    {
      key: 'order-list',
      name: '订单管理',
      path: '/order/order',
      permission: 'order:order:list',
      icon: <AppstoreOutlined style={{ fontSize: 20, color: '#13c2c2' }} />,
      desc: '订单审核与发货',
    },
    {
      key: 'member-list',
      name: '会员管理',
      path: '/member/member',
      permission: 'member:member:list',
      icon: <TeamOutlined style={{ fontSize: 20, color: '#52c41a' }} />,
      desc: '查看会员与资料',
    },
    {
      key: 'file-list',
      name: '素材管理',
      path: '/system/file',
      permission: 'system:file:list',
      icon: <FileImageOutlined style={{ fontSize: 20, color: '#fa8c16' }} />,
      desc: '上传与管理图片',
    },
    {
      key: 'system-setting',
      name: '系统设置',
      path: '/system/setting/shop',
      permission: 'system:setting:view',
      icon: <SettingOutlined style={{ fontSize: 20, color: '#fa541c' }} />,
      desc: '商城与站点配置',
    },
  ];

  // 严格根据用户权限过滤，无权限时不显示入口
  const visibleLinks = allLinks.filter((item) => hasPermission(item.permission));

  if (visibleLinks.length === 0) {
    return null;
  }

  return (
    <Card variant="outlined" style={{ borderRadius: 8 }} title="快捷入口">
      <Row gutter={[12, 12]}>
        {visibleLinks.map((item) => (
          <Col xs={12} sm={8} md={8} lg={4} key={item.key}>
            <div
              onClick={() => navigate(item.path)}
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                padding: '16px 8px',
                borderRadius: 8,
                backgroundColor: token.colorFillAlter,
                border: `1px solid ${token.colorBorderSecondary}`,
                cursor: 'pointer',
                transition: 'all 0.2s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = token.colorFillSecondary;
                e.currentTarget.style.borderColor = token.colorPrimary;
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = token.colorFillAlter;
                e.currentTarget.style.borderColor = token.colorBorderSecondary;
              }}
            >
              <div style={{ marginBottom: 8 }}>{item.icon}</div>
              <Text strong style={{ fontSize: 13, marginBottom: 2 }}>
                {item.name}
              </Text>
              <Text type="secondary" style={{ fontSize: 11 }}>
                {item.desc}
              </Text>
            </div>
          </Col>
        ))}
      </Row>
    </Card>
  );
};
