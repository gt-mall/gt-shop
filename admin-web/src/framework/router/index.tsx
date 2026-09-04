import { Navigate, Route, Routes } from 'react-router-dom';

import { useAppStore } from '@/framework/store';
import DemoPage from '@/modules/demo';
import DashboardHomePage from '@/modules/dashboard/home';
import HelloPluginPage from '@/modules/plugin/hello';
import RolePage from '@/modules/system/role';
import UserPage from '@/modules/system/user';

export function AppRoutes() {
  const hasPermission = useAppStore((state) => state.hasPermission);
  return (
    <Routes>
      <Route path="/" element={<DashboardHomePage />} />
      <Route path="/demo" element={<DemoPage />} />
      <Route path="/dashboard/home" element={<DashboardHomePage />} />
      <Route path="/plugin/hello" element={<HelloPluginPage />} />
      <Route path="/plugins/hello" element={<HelloPluginPage />} />
      {hasPermission('system:role:list') ? <Route path="/system/role" element={<RolePage />} /> : null}
      {hasPermission('system:user:list') ? <Route path="/system/user" element={<UserPage />} /> : null}
      {hasPermission('system:user:list') ? <Route path="/user/users" element={<UserPage />} /> : null}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
