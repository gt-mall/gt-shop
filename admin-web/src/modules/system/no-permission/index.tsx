import React from 'react';
import { Button, Card, Result } from 'antd';
import { LockOutlined, LogoutOutlined } from '@ant-design/icons';

import { useAuth } from '@/framework/store';

export default function NoPermissionPage() {
  const { logout } = useAuth();

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: 'calc(100vh - 200px)',
        padding: '24px',
      }}
    >
      <Card
        style={{
          maxWidth: 520,
          width: '100%',
          boxShadow: '0 4px 20px rgba(0, 0, 0, 0.05)',
          borderRadius: 12,
          textAlign: 'center',
        }}
      >
        <Result
          status="warning"
          icon={<LockOutlined style={{ fontSize: 56, color: '#FAAD14' }} />}
          title="暂无可访问功能"
          subTitle="当前账号尚未分配后台功能权限，请联系管理员为当前角色授权。"
          extra={
            <Button
              type="primary"
              danger
              icon={<LogoutOutlined />}
              size="middle"
              onClick={logout}
            >
              退出登录
            </Button>
          }
        />
      </Card>
    </div>
  );
}
