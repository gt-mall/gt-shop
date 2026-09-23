import { useEffect, useState } from 'react';
import { Button, Card, Checkbox, Form, Input, message, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';

import { api } from '@/framework/http';
import { getDefaultAuthorizedPathFromPermissions } from '@/framework/router';
import { useAppStore, useAuth } from '@/framework/store';

type LoginForm = {
  username: string;
  password: string;
  remember: boolean;
};

export default function LoginPage() {
  const { login, loading } = useAuth();
  const navigate = useNavigate();
  const [version, setVersion] = useState<string>(
    typeof __APP_VERSION__ !== 'undefined' ? __APP_VERSION__ : ''
  );

  useEffect(() => {
    let active = true;
    const fetchVersion = async () => {
      const res = await api.api.getSystemVersion();
      if (active && res.code === 1 && res.data?.version) {
        setVersion(res.data.version);
      }
    };
    void fetchVersion();
    return () => {
      active = false;
    };
  }, []);

  const submit = async (values: LoginForm) => {
    const result = await login(values);
    if (!result.success) {
      message.error(result.message || '登录失败');
      return;
    }
    const current = useAppStore.getState().current;
    const defaultPath = getDefaultAuthorizedPathFromPermissions(current?.permissions);
    navigate(defaultPath, { replace: true });
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center',
        background: '#f5f5f5',
        padding: '24px 0',
      }}
    >
      <div style={{ flex: 1, display: 'flex', alignItems: 'center' }}>
        <Card style={{ width: 380 }}>
          <Typography.Title level={3} style={{ textAlign: 'center', marginBottom: 28 }}>
            Mall Admin
          </Typography.Title>
          <Form<LoginForm>
            layout="vertical"
            requiredMark={false}
            initialValues={{ remember: false }}
            onFinish={submit}
          >
            <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
              <Input autoComplete="username" />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password autoComplete="current-password" />
            </Form.Item>
            <Form.Item name="remember" valuePropName="checked">
              <Checkbox>记住登录 30 天</Checkbox>
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              登录
            </Button>
          </Form>
        </Card>
      </div>
      {version && (
        <div style={{ color: '#9ca3af', fontSize: 13, textAlign: 'center', paddingBottom: 16 }}>
          GT-Shop v{version}
        </div>
      )}
    </div>
  );
}
