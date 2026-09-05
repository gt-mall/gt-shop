# 用户与权限 Hooks

统一从 `@/framework/store` 导入，在 React 组件或自定义 Hook 的顶层调用。
这些 Hook 只订阅已有状态，不会自动发请求，也不需要页面自己维护用户、角色或权限副本。

运行示例：**首页 → Hooks 示例**（`/dashboard/demo`，原 `/demo` 也可访问）。

| Hook | 返回值 | 用途 |
| --- | --- | --- |
| `useUser()` | `CurrentUser \| undefined` | 当前用户的 ID、用户名、昵称等基本信息 |
| `useRoles()` | `readonly CurrentRole[]` | 当前用户的角色列表，默认 `[]` |
| `usePermissions()` | `readonly string[]` | 当前用户的权限 code 列表，默认 `[]` |
| `usePermission()` | `(code?: string) => boolean` | 判断指定权限；不传 code 表示无需权限 |
| `useAuth()` | 登录状态和操作，见下表 | 登录、退出、刷新用户信息 |

`useAuth()` 返回：

| 字段 | 说明 |
| --- | --- |
| `isLoggedIn` | 当前用户信息已获取，登录态可用；仅存在 Token 时仍为 `false` |
| `initialized` | 本次登录态恢复是否完成；未登录时也可为 `true` |
| `loading` | 正在登录或刷新当前用户信息 |
| `login(request)` | 接收生成的 `LoginRequest`，返回 `{ success, message? }` |
| `logout()` | 清空登录态和持久化信息 |
| `refreshCurrent()` | 重新获取用户、角色和权限，返回 `Promise<boolean>`；并发调用复用请求 |

## 获取用户、角色和权限

```tsx
import { usePermissions, useRoles, useUser } from '@/framework/store';

export default function UserInfo() {
  const user = useUser();
  const roles = useRoles();
  const permissions = usePermissions();

  return (
    <div>
      <p>{user?.nickname || user?.username || '未登录'}</p>
      <p>角色：{roles.map((role) => role.name).join('、') || '无'}</p>
      <p>权限数量：{permissions.length}</p>
    </div>
  );
}
```

登录态未恢复时，`useUser()` 返回 `undefined`，角色和权限返回空数组。列表是只读视图；
需要排序时使用 `[...roles].sort(...)`，不要直接修改 store 中的数组。

## 按钮权限

```tsx
import { Button } from 'antd';
import { usePermission } from '@/framework/store';

export default function CreateUserButton({ onClick }: { onClick: () => void }) {
  const hasPermission = usePermission();
  if (!hasPermission('system:user:create')) return null;
  return <Button onClick={onClick}>新增用户</Button>;
}
```

多个权限都需要满足时使用 `&&`，满足任意一个时使用 `||`。
操作权限使用权限 code 判断，角色列表用于展示；前端不根据角色名绕过权限。

路由权限使用已有 JSX 组件，可嵌套检查多项权限：

```tsx
<Route element={<PermissionRoute permission="system:user:list" />}>
  <Route path="/system/user" element={<UserPage />} />
</Route>
```

`PermissionRoute` 从 `@/framework/router/permission` 导入；这段配置放在 `<Routes>` 内。

## 登录、退出与刷新

现有登录页 `modules/login/index.tsx` 已使用 `useAuth()`：

```tsx
import { Button, Checkbox, Form, Input, message } from 'antd';
import type { LoginRequest } from '@/framework/http/generated_api';
import { useAuth } from '@/framework/store';

export default function LoginExample() {
  const { login, loading } = useAuth();

  const submit = async (values: LoginRequest) => {
    const result = await login(values);
    if (!result.success) message.error(result.message || '登录失败');
  };

  return (
    <Form<LoginRequest> onFinish={submit} initialValues={{ remember: false }}>
      <Form.Item name="username" rules={[{ required: true }]}>
        <Input placeholder="用户名" autoComplete="username" />
      </Form.Item>
      <Form.Item name="password" rules={[{ required: true }]}>
        <Input.Password placeholder="密码" autoComplete="current-password" />
      </Form.Item>
      <Form.Item name="remember" valuePropName="checked">
        <Checkbox>记住登录 30 天</Checkbox>
      </Form.Item>
      <Button htmlType="submit" loading={loading}>登录</Button>
    </Form>
  );
}
```

下面是可直接复用的刷新和退出示例：

```tsx
import { Button, message, Space } from 'antd';
import { useAuth } from '@/framework/store';

export default function SessionActions() {
  const { isLoggedIn, loading, refreshCurrent, logout } = useAuth();

  const refresh = async () => {
    const success = await refreshCurrent();
    if (success) message.success('用户信息已刷新');
    else message.error('刷新失败，请重新登录');
  };

  return (
    <Space>
      <Button disabled={!isLoggedIn} loading={loading} onClick={refresh}>刷新用户信息</Button>
      <Button disabled={!isLoggedIn} onClick={logout}>退出登录</Button>
    </Space>
  );
}
```

登录成功与应用重新加载时，框架会获取当前用户信息，页面不需要在 `useEffect` 中重复刷新。
当前用户的角色或权限变更后，可调用 `refreshCurrent()` 更新展示和访问权限。
刷新失败沿用现有 store 行为：清理登录态并返回 `false`；网络错误由公共 HTTP 层处理。
