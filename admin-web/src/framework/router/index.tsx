import { Navigate, Route, Routes } from 'react-router-dom';

import DemoPage from '@/modules/demo';
import DashboardHomePage from '@/modules/dashboard/home';
import HelloPluginPage from '@/modules/plugin/hello';
import RolePage from '@/modules/system/role';
import RolePermissionPage from '@/modules/system/role/permission';
import UserPage from '@/modules/system/user';

import { PermissionRoute } from './permission';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<DashboardHomePage />} />
      <Route path="/demo" element={<DemoPage />} />
      <Route path="/dashboard/home" element={<DashboardHomePage />} />
      <Route path="/dashboard/demo" element={<DemoPage />} />
      <Route path="/plugin/hello" element={<HelloPluginPage />} />
      <Route path="/plugins/hello" element={<HelloPluginPage />} />
      <Route element={<PermissionRoute permission="system:role:list" />}>
        <Route path="/system/role" element={<RolePage />} />
        <Route element={<PermissionRoute permission="system:role:permission" />}>
          <Route path="/system/role/:roleId/permission" element={<RolePermissionPage />} />
        </Route>
      </Route>
      <Route element={<PermissionRoute permission="system:user:list" />}>
        <Route path="/system/user" element={<UserPage />} />
        <Route path="/user/users" element={<UserPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
