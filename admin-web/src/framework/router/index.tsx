import { Route, Routes } from 'react-router-dom';

import DemoPage from '@/modules/demo';
import DashboardHomePage from '@/modules/dashboard/home';
import HelloPluginPage from '@/modules/plugin/hello';
import UserPage from '@/modules/system/user';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<DashboardHomePage />} />
      <Route path="/demo" element={<DemoPage />} />
      <Route path="/dashboard/home" element={<DashboardHomePage />} />
      <Route path="/plugin/hello" element={<HelloPluginPage />} />
      <Route path="/plugins/hello" element={<HelloPluginPage />} />
      <Route path="/system/user" element={<UserPage />} />
      <Route path="/user/users" element={<UserPage />} />
    </Routes>
  );
}
