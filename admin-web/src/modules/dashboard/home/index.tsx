import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Col, Result, Row, Space, Typography, message, theme } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

import {
  api,
  type DashboardOverviewResponse,
  type DashboardTrendResponse,
} from '@/framework/http';
import { usePermission } from '@/framework/store';

import { OverviewCards } from './overview-cards';
import { SalesChart } from './sales-chart';
import { TodoCard } from './todo-card';
import { QuickLinks } from './quick-links';

const { Title, Text } = Typography;

export default function DashboardHomePage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();

  const canViewDashboard = hasPermission('dashboard:view');

  const [overviewData, setOverviewData] = useState<DashboardOverviewResponse>();
  const [trendData, setTrendData] = useState<DashboardTrendResponse>();
  const [overviewLoading, setOverviewLoading] = useState(false);
  const [trendLoading, setTrendLoading] = useState(false);
  const [days, setDays] = useState<number>(7);
  const [lastUpdated, setLastUpdated] = useState<string>('');

  const overviewSeqRef = useRef(0);
  const trendSeqRef = useRef(0);

  const fetchOverview = useCallback(async () => {
    if (!canViewDashboard) return;
    const currentSeq = ++overviewSeqRef.current;
    setOverviewLoading(true);

    const res = await api.api.getDashboardOverview();

    if (currentSeq !== overviewSeqRef.current) return;
    setOverviewLoading(false);

    if (res.code === 1) {
      setOverviewData(res.data);
      setLastUpdated(dayjs().format('YYYY-MM-DD HH:mm:ss'));
    } else {
      message.error(res.message || '获取看板概览数据失败');
    }
  }, [canViewDashboard]);

  const fetchTrend = useCallback(
    async (targetDays: number) => {
      if (!canViewDashboard) return;
      const currentSeq = ++trendSeqRef.current;
      setTrendLoading(true);

      const res = await api.api.getDashboardTrend({ days: targetDays });

      if (currentSeq !== trendSeqRef.current) return;
      setTrendLoading(false);

      if (res.code === 1) {
        setTrendData(res.data);
      } else {
        message.error(res.message || '获取销售趋势数据失败');
      }
    },
    [canViewDashboard],
  );

  const handleRefresh = useCallback(() => {
    void fetchOverview();
    void fetchTrend(days);
  }, [fetchOverview, fetchTrend, days]);

  const handleDaysChange = useCallback(
    (newDays: number) => {
      setDays(newDays);
      void fetchTrend(newDays);
    },
    [fetchTrend],
  );

  useEffect(() => {
    if (canViewDashboard) {
      void fetchOverview();
      void fetchTrend(days);
    }
  }, [canViewDashboard, fetchOverview, fetchTrend, days]);

  if (!canViewDashboard) {
    return (
      <div style={{ padding: '60px 0' }}>
        <Result
          status="403"
          title="暂无数据看板访问权限"
          subTitle="您当前所在的角色未被授予数据看板的查看权限，请联系超级管理员在【角色管理】中授权。"
        />
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      {/* 顶部标题与操作栏 */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: 12,
        }}
      >
        <div>
          <Title level={4} style={{ margin: 0 }}>
            商城数据看板
          </Title>
          <Text type="secondary" style={{ fontSize: 13 }}>
            实时监控商城日常经营核心数据、业务走势与待办事项
          </Text>
        </div>

        <Space align="center">
          {lastUpdated && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              最后更新：{lastUpdated}
            </Text>
          )}
          <Button
            icon={<ReloadOutlined spin={overviewLoading || trendLoading} />}
            loading={overviewLoading || trendLoading}
            onClick={handleRefresh}
          >
            刷新数据
          </Button>
        </Space>
      </div>

      {/* 顶部核心指标与业务概况卡片 */}
      <OverviewCards data={overviewData} loading={overviewLoading} />

      {/* 趋势图表与待办事项 */}
      <Row gutter={[20, 20]}>
        <Col xs={24} lg={16} xl={17}>
          <SalesChart
            data={trendData}
            loading={trendLoading}
            days={days}
            onDaysChange={handleDaysChange}
          />
        </Col>
        <Col xs={24} lg={8} xl={7}>
          <TodoCard data={overviewData} loading={overviewLoading} />
        </Col>
      </Row>

      {/* 快捷入口 */}
      <QuickLinks />
    </div>
  );
}
