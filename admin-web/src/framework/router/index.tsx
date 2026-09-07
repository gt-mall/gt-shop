import { Navigate, Route, Routes } from 'react-router-dom';

import DemoPage from '@/modules/demo';
import DashboardHomePage from '@/modules/dashboard/home';
import HelloPluginPage from '@/modules/plugin/hello';
import RolePage from '@/modules/system/role';
import RolePermissionPage from '@/modules/system/role/permission';
import UserPage from '@/modules/system/user';
import FilePage from '@/modules/system/file';
import FileStoragePage from '@/modules/system/file/storage';
import SystemSettingPage from '@/modules/system/setting';
import ImageDemoPage from '@/modules/demo/image';
import EditorDemoPage from '@/modules/demo/editor';

import { PermissionRoute } from './permission';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<DashboardHomePage />} />
      <Route path="/demo" element={<DemoPage />} />
      <Route path="/dashboard/home" element={<DashboardHomePage />} />
      <Route path="/dashboard/demo" element={<DemoPage />} />
      <Route path="/dashboard/editor-demo" element={<EditorDemoPage />} />
      <Route path="/plugin/hello" element={<HelloPluginPage />} />
      <Route path="/plugins/hello" element={<HelloPluginPage />} />
      <Route element={<PermissionRoute permission="system:file:list" />}>
        <Route path="/system/file" element={<FilePage />} />
        <Route path="/dashboard/image-demo" element={<ImageDemoPage />} />
      </Route>
      <Route element={<PermissionRoute permission="system:file:config" />}>
        <Route path="/system/file-storage" element={<FileStoragePage />} />
      </Route>
      <Route element={<PermissionRoute permission="system:setting:view" />}>
        <Route path="/system/setting" element={<Navigate to="/system/setting/shop" replace />} />
        <Route path="/system/setting/shop" element={<SystemSettingPage />} />
        <Route path="/system/setting/site" element={<SystemSettingPage />} />
        <Route path="/system/setting/storage" element={<SystemSettingPage />} />
        <Route path="/system/setting/file" element={<Navigate to="/system/setting/storage" replace />} />
      </Route>
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
