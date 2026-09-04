import { Button, Card, Checkbox, Form, Input, message, Typography } from 'antd';

import { useAppStore } from '@/framework/store';

type LoginForm = {
  username: string;
  password: string;
  remember: boolean;
};

export default function LoginPage() {
  const login = useAppStore((state) => state.login);
  const loading = useAppStore((state) => state.loading);

  const submit = async (values: LoginForm) => {
    const result = await login(values);
    if (!result.success) {
      message.error(result.message || '登录失败');
    }
  };

  return (
    <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', background: '#f5f5f5' }}>
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
  );
}
