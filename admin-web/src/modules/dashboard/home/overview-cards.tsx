import React from 'react';
import { Card, Col, Row, Statistic, Tag, Typography, theme } from 'antd';
import {
  AlertOutlined,
  CarryOutOutlined,
  CustomerServiceOutlined,
  InboxOutlined,
  PayCircleOutlined,
  ShopOutlined,
  ShoppingOutlined,
  TeamOutlined,
  UserAddOutlined,
} from '@ant-design/icons';
import type { DashboardOverviewResponse } from '@/framework/http/generated_api';

const { Text } = Typography;

interface OverviewCardsProps {
  data?: DashboardOverviewResponse;
  loading?: boolean;
}

function formatMoney(amount?: string): string {
  if (!amount) return '0.00';
  const parts = amount.split('.');
  const intPart = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  const decPart = parts[1] || '00';
  return `${intPart}.${decPart}`;
}

export const OverviewCards: React.FC<OverviewCardsProps> = ({ data, loading }) => {
  const { token } = theme.useToken();

  const cardStyle: React.CSSProperties = {
    borderRadius: 8,
    transition: 'all 0.2s ease',
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Row 1: 核心指标卡片 (4张) */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={12} lg={6}>
          <Card
            variant="outlined"
            loading={loading}
            style={{
              ...cardStyle,
              borderLeft: `4px solid ${token.colorPrimary}`,
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div>
                <Text type="secondary" style={{ fontSize: 14 }}>
                  今日销售额
                </Text>
                <div style={{ marginTop: 8 }}>
                  <span style={{ fontSize: 16, color: token.colorPrimary, marginRight: 4 }}>¥</span>
                  <span style={{ fontSize: 28, fontWeight: 600, color: token.colorTextHeading }}>
                    {formatMoney(data?.today_sales_amount)}
                  </span>
                </div>
              </div>
              <div
                style={{
                  width: 44,
                  height: 44,
                  borderRadius: 8,
                  backgroundColor: token.colorPrimaryBg,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: token.colorPrimary,
                  fontSize: 22,
                }}
              >
                <PayCircleOutlined />
              </div>
            </div>
            <div style={{ marginTop: 12, paddingTop: 10, borderTop: `1px solid ${token.colorBorderSecondary}` }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                今日已支付订单：
              </Text>
              <Text strong style={{ fontSize: 12, marginLeft: 4 }}>
                {data?.today_paid_order_count ?? 0} 单
              </Text>
            </div>
          </Card>
        </Col>

        <Col xs={24} sm={12} md={12} lg={6}>
          <Card
            variant="outlined"
            loading={loading}
            style={{
              ...cardStyle,
              borderLeft: `4px solid ${token.colorSuccess}`,
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div>
                <Text type="secondary" style={{ fontSize: 14 }}>
                  今日订单
                </Text>
                <div style={{ marginTop: 8 }}>
                  <span style={{ fontSize: 28, fontWeight: 600, color: token.colorTextHeading }}>
                    {data?.today_order_count ?? 0}
                  </span>
                  <span style={{ fontSize: 14, color: token.colorTextSecondary, marginLeft: 6 }}>单</span>
                </div>
              </div>
              <div
                style={{
                  width: 44,
                  height: 44,
                  borderRadius: 8,
                  backgroundColor: token.colorSuccessBg,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: token.colorSuccess,
                  fontSize: 22,
                }}
              >
                <ShoppingOutlined />
              </div>
            </div>
            <div style={{ marginTop: 12, paddingTop: 10, borderTop: `1px solid ${token.colorBorderSecondary}` }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                待付款订单：
              </Text>
              <Text strong style={{ fontSize: 12, marginLeft: 4 }}>
                {data?.pending_payment_count ?? 0} 单
              </Text>
            </div>
          </Card>
        </Col>

        <Col xs={24} sm={12} md={12} lg={6}>
          <Card
            variant="outlined"
            loading={loading}
            style={{
              ...cardStyle,
              borderLeft: `4px solid ${token.colorWarning}`,
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div>
                <Text type="secondary" style={{ fontSize: 14 }}>
                  待发货订单
                </Text>
                <div style={{ marginTop: 8 }}>
                  <span style={{ fontSize: 28, fontWeight: 600, color: token.colorTextHeading }}>
                    {data?.pending_delivery_count ?? 0}
                  </span>
                  <span style={{ fontSize: 14, color: token.colorTextSecondary, marginLeft: 6 }}>单</span>
                </div>
              </div>
              <div
                style={{
                  width: 44,
                  height: 44,
                  borderRadius: 8,
                  backgroundColor: token.colorWarningBg,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: token.colorWarning,
                  fontSize: 22,
                }}
              >
                <CarryOutOutlined />
              </div>
            </div>
            <div style={{ marginTop: 12, paddingTop: 10, borderTop: `1px solid ${token.colorBorderSecondary}` }}>
              {(data?.pending_delivery_count ?? 0) > 0 ? (
                <Tag color="warning" style={{ margin: 0 }}>
                  急需处理发货
                </Tag>
              ) : (
                <Tag color="default" style={{ margin: 0 }}>
                  暂无待发货
                </Tag>
              )}
            </div>
          </Card>
        </Col>

        <Col xs={24} sm={12} md={12} lg={6}>
          <Card
            variant="outlined"
            loading={loading}
            style={{
              ...cardStyle,
              borderLeft: `4px solid ${token.colorError}`,
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div>
                <Text type="secondary" style={{ fontSize: 14 }}>
                  待处理售后
                </Text>
                <div style={{ marginTop: 8 }}>
                  <span style={{ fontSize: 28, fontWeight: 600, color: token.colorTextHeading }}>
                    {data?.pending_aftersale_count ?? 0}
                  </span>
                  <span style={{ fontSize: 14, color: token.colorTextSecondary, marginLeft: 6 }}>笔</span>
                </div>
              </div>
              <div
                style={{
                  width: 44,
                  height: 44,
                  borderRadius: 8,
                  backgroundColor: token.colorErrorBg,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: token.colorError,
                  fontSize: 22,
                }}
              >
                <CustomerServiceOutlined />
              </div>
            </div>
            <div style={{ marginTop: 12, paddingTop: 10, borderTop: `1px solid ${token.colorBorderSecondary}` }}>
              {(data?.pending_aftersale_count ?? 0) > 0 ? (
                <Tag color="error" style={{ margin: 0 }}>
                  需后台处理
                </Tag>
              ) : (
                <Tag color="default" style={{ margin: 0 }}>
                  售后已清空
                </Tag>
              )}
            </div>
          </Card>
        </Col>
      </Row>

      {/* Row 2: 业务概况卡片 (5个指标) */}
      <Card variant="outlined" style={cardStyle} loading={loading}>
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8} md={8} lg={4} xl={4} style={{ textAlign: 'center' }}>
            <Statistic
              title={
                <span>
                  <ShopOutlined style={{ marginRight: 6 }} />
                  商品总数
                </span>
              }
              value={data?.product_count ?? 0}
              suffix="件"
              valueStyle={{ fontSize: 22, fontWeight: 600 }}
            />
          </Col>

          <Col xs={12} sm={8} md={8} lg={5} xl={5} style={{ textAlign: 'center' }}>
            <Statistic
              title={
                <span>
                  <ShoppingOutlined style={{ marginRight: 6, color: token.colorSuccess }} />
                  在售商品
                </span>
              }
              value={data?.on_sale_product_count ?? 0}
              suffix="件"
              valueStyle={{ fontSize: 22, fontWeight: 600, color: token.colorSuccess }}
            />
          </Col>

          <Col xs={12} sm={8} md={8} lg={5} xl={5} style={{ textAlign: 'center' }}>
            <Statistic
              title={
                <span>
                  <TeamOutlined style={{ marginRight: 6 }} />
                  会员总数
                </span>
              }
              value={data?.member_count ?? 0}
              suffix="人"
              valueStyle={{ fontSize: 22, fontWeight: 600 }}
            />
          </Col>

          <Col xs={12} sm={12} md={12} lg={5} xl={5} style={{ textAlign: 'center' }}>
            <Statistic
              title={
                <span>
                  <UserAddOutlined style={{ marginRight: 6, color: token.colorPrimary }} />
                  今日新增会员
                </span>
              }
              value={data?.today_new_member_count ?? 0}
              suffix="人"
              valueStyle={{ fontSize: 22, fontWeight: 600, color: token.colorPrimary }}
            />
          </Col>

          <Col xs={12} sm={12} md={12} lg={5} xl={5} style={{ textAlign: 'center' }}>
            <Statistic
              title={
                <span>
                  <AlertOutlined
                    style={{
                      marginRight: 6,
                      color: (data?.low_stock_sku_count ?? 0) > 0 ? token.colorError : undefined,
                    }}
                  />
                  低库存 SKU
                </span>
              }
              value={data?.low_stock_sku_count ?? 0}
              suffix="个"
              valueStyle={{
                fontSize: 22,
                fontWeight: 600,
                color: (data?.low_stock_sku_count ?? 0) > 0 ? token.colorError : undefined,
              }}
            />
          </Col>
        </Row>
      </Card>
    </div>
  );
};
