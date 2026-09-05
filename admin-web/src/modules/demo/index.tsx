import { Button, Card, Descriptions, message, Space, Tag, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';

import { useAuth, usePermission, usePermissions, useRoles, useUser } from '@/framework/store';

const usageExample = `import { useAuth, usePermission, usePermissions, useRoles, useUser } from '@/framework/store';

export default function UserSummary() {
  const user = useUser();
  const roles = useRoles();
  const permissions = usePermissions();
  const hasPermission = usePermission();
  const { isLoggedIn, loading, refreshCurrent, logout } = useAuth();

  return (
    <section>
      <p>{user?.nickname || user?.username || '未登录'}</p>
      <p>角色：{roles.map((role) => role.name).join('、') || '无'}</p>
      <p>权限数量：{permissions.length}</p>
      <p>可查看用户：{hasPermission('system:user:list') ? '是' : '否'}</p>
      <button disabled={!isLoggedIn || loading} onClick={refreshCurrent}>刷新用户信息</button>
      <button disabled={!isLoggedIn} onClick={logout}>退出登录</button>
    </section>
  );
}`;

export default function DemoPage() {
  const user = useUser();
  const roles = useRoles();
  const permissions = usePermissions();
  const hasPermission = usePermission();
  const { isLoggedIn, initialized, loading, refreshCurrent } = useAuth();
  const navigate = useNavigate();

  const refresh = async () => {
    const success = await refreshCurrent();
    if (success) message.success('用户、角色和权限已刷新');
    else message.error('刷新失败，请重新登录');
  };

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Card
        title="当前用户"
        extra={<Button loading={loading} disabled={!isLoggedIn} onClick={refresh}>刷新用户信息</Button>}
      >
        <Descriptions
          column={{ xs: 1, sm: 2, lg: 3 }}
          items={[
            { key: 'id', label: '用户 ID', children: user?.id ?? '—' },
            { key: 'username', label: '用户名', children: user?.username || '—' },
            { key: 'nickname', label: '昵称', children: user?.nickname || '—' },
            { key: 'login', label: '登录状态', children: isLoggedIn ? '已登录' : '未登录' },
            { key: 'initialized', label: '登录态恢复', children: initialized ? '已完成' : '进行中' },
          ]}
        />
      </Card>

      <Card title="角色与权限">
        <Typography.Paragraph>角色：{roles.length ? roles.map((role) => (
          <Tag key={role.id} color="blue">{role.name || role.code}</Tag>
        )) : '暂无角色'}</Typography.Paragraph>
        <Typography.Paragraph>权限（{permissions.length} 项）：</Typography.Paragraph>
        <Space wrap>
          {permissions.map((permission) => <Tag key={permission}>{permission}</Tag>)}
          {permissions.length === 0 && <Typography.Text type="secondary">暂无权限</Typography.Text>}
        </Space>
      </Card>

      <Card title="权限控制示例">
        <Typography.Paragraph type="secondary">按钮根据当前权限启用，点击后进入对应页面。</Typography.Paragraph>
        <Space wrap>
          <Button disabled={!hasPermission('system:user:list')} onClick={() => navigate('/system/user')}>用户管理</Button>
          <Button disabled={!hasPermission('system:role:list')} onClick={() => navigate('/system/role')}>权限管理</Button>
        </Space>
      </Card>

      <Card title="使用方法">
        <Typography.Paragraph type="secondary">
          Hook 读取同一份登录状态，数据变化后会自动更新页面；手动刷新调用 refreshCurrent()。
        </Typography.Paragraph>
        <Typography.Paragraph copyable={{ text: usageExample }}>
          <pre style={{ overflowX: 'auto', margin: 0 }}>{usageExample}</pre>
        </Typography.Paragraph>
      </Card>
    </Space>
  );
}
