import React from 'react';
import { Badge, Button, Card, List, Space, Tag, Typography, theme } from 'antd';
import {
  AlertOutlined,
  CarryOutOutlined,
  CustomerServiceOutlined,
  InboxOutlined,
  RightOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { DashboardOverviewResponse } from '@/framework/http/generated_api';

const { Text, Title } = Typography;

interface TodoCardProps {
  data?: DashboardOverviewResponse;
  loading?: boolean;
}

export const TodoCard: React.FC<TodoCardProps> = ({ data, loading }) => {
  const { token } = theme.useToken();
  const navigate = useNavigate();

  const todoItems = [
    {
      key: 'delivery',
      title: '待发货订单',
      count: data?.pending_delivery_count ?? 0,
      icon: <CarryOutOutlined style={{ fontSize: 18, color: token.colorWarning }} />,
      desc: '买家已完成支付，请尽快核对并安排快递发货',
      actionText: '去发货',
      badgeColor: (data?.pending_delivery_count ?? 0) > 0 ? token.colorWarning : token.colorTextQuaternary,
      urgent: (data?.pending_delivery_count ?? 0) > 0,
      path: '/order/order',
    },
    {
      key: 'aftersale',
      title: '待处理售后',
      count: data?.pending_aftersale_count ?? 0,
      icon: <CustomerServiceOutlined style={{ fontSize: 18, color: token.colorError }} />,
      desc: '包含待审核申请、待商家确认收货及需重试退款记录',
      actionText: '去审核',
      badgeColor: (data?.pending_aftersale_count ?? 0) > 0 ? token.colorError : token.colorTextQuaternary,
      urgent: (data?.pending_aftersale_count ?? 0) > 0,
      path: '/order/aftersale',
    },
    {
      key: 'lowstock',
      title: '低库存 SKU',
      count: data?.low_stock_sku_count ?? 0,
      icon: <InboxOutlined style={{ fontSize: 18, color: token.colorError }} />,
      desc: '可售库存已低于预警阈值或默认安全线(<=10)，需及时补货',
      actionText: '去补货',
      badgeColor: (data?.low_stock_sku_count ?? 0) > 0 ? token.colorError : token.colorTextQuaternary,
      urgent: (data?.low_stock_sku_count ?? 0) > 0,
      path: '/product/inventory',
    },
  ];

  const totalPending = (data?.pending_delivery_count ?? 0) + (data?.pending_aftersale_count ?? 0) + (data?.low_stock_sku_count ?? 0);

  return (
    <Card
      variant="outlined"
      loading={loading}
      style={{ borderRadius: 8, height: '100%' }}
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Space>
            <span>待处理事项</span>
            {totalPending > 0 ? (
              <Badge count={totalPending} overflowCount={99} style={{ backgroundColor: token.colorError }} />
            ) : (
              <Tag color="success">全部已处理</Tag>
            )}
          </Space>
        </div>
      }
    >
      <List
        itemLayout="horizontal"
        dataSource={todoItems}
        renderItem={(item) => (
          <List.Item
            key={item.key}
            style={{
              padding: '12px 8px',
              borderRadius: 6,
              transition: 'background 0.2s',
              cursor: 'pointer',
            }}
            onClick={() => navigate(item.path)}
            actions={[
              <Button
                key="btn"
                type={item.urgent ? 'primary' : 'default'}
                size="small"
                danger={item.key === 'aftersale' && item.urgent}
                onClick={(e) => {
                  e.stopPropagation();
                  navigate(item.path);
                }}
              >
                {item.actionText}
                <RightOutlined style={{ fontSize: 10 }} />
              </Button>,
            ]}
          >
            <List.Item.Meta
              avatar={
                <div
                  style={{
                    width: 36,
                    height: 36,
                    borderRadius: 6,
                    backgroundColor: token.colorFillAlter,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  {item.icon}
                </div>
              }
              title={
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Text strong>{item.title}</Text>
                  <Tag
                    color={item.urgent ? (item.key === 'delivery' ? 'warning' : 'error') : 'default'}
                    style={{ borderRadius: 10, padding: '0 8px', fontWeight: 600 }}
                  >
                    {item.count}
                  </Tag>
                </div>
              }
              description={<Text type="secondary" style={{ fontSize: 12 }}>{item.desc}</Text>}
            />
          </List.Item>
        )}
      />
    </Card>
  );
};
