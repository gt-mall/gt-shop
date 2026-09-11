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
import ProductCategoryPage from '@/modules/product/category';
import ProductListPage from '@/modules/product/product';
import ProductEditPage from '@/modules/product/product/edit';
import ProductInventoryPage from '@/modules/product/inventory';
import ProductInventoryLogPage from '@/modules/product/inventory/log';
import FreightListPage from '@/modules/product/freight';
import FreightEditPage from '@/modules/product/freight/edit';
import MemberListPage from '@/modules/member/member';
import MemberDetailPage from '@/modules/member/member/detail';
import OrderListPage from '@/modules/order/order';
import OrderDetailPage from '@/modules/order/order/detail';

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
        <Route path="/system/setting/mp" element={<SystemSettingPage />} />
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
      <Route path="/product" element={<Navigate to="/product/product" replace />} />
      <Route element={<PermissionRoute permission="product:category:list" />}>
        <Route path="/product/category" element={<ProductCategoryPage />} />
      </Route>
      <Route element={<PermissionRoute permission="product:product:list" />}>
        <Route element={<PermissionRoute permission="product:product:create" />}>
          <Route path="/product/product/create" element={<ProductEditPage />} />
        </Route>
        <Route element={<PermissionRoute permission="product:product:update" />}>
          <Route path="/product/product/:id/edit" element={<ProductEditPage />} />
        </Route>
        <Route path="/product/product" element={<ProductListPage />} />
        <Route path="/product/product/:id" element={<ProductEditPage />} />
      </Route>
      <Route element={<PermissionRoute permission="product:inventory:list" />}>
        <Route path="/product/inventory" element={<ProductInventoryPage />} />
      </Route>
      <Route element={<PermissionRoute permission="product:inventory:log" />}>
        <Route path="/product/inventory/log" element={<ProductInventoryLogPage />} />
      </Route>
      <Route element={<PermissionRoute permission="product:freight:list" />}>
        <Route path="/product/freight" element={<FreightListPage />} />
        <Route element={<PermissionRoute permission="product:freight:create" />}>
          <Route path="/product/freight/create" element={<FreightEditPage />} />
        </Route>
        <Route element={<PermissionRoute permission="product:freight:update" />}>
          <Route path="/product/freight/:id/edit" element={<FreightEditPage />} />
        </Route>
        <Route path="/product/freight/:id" element={<FreightEditPage />} />
      </Route>
      <Route path="/order" element={<Navigate to="/order/order" replace />} />
      <Route element={<PermissionRoute permission="order:order:list" />}>
        <Route path="/order/order" element={<OrderListPage />} />
      </Route>
      <Route element={<PermissionRoute permission="order:order:detail" />}>
        <Route path="/order/order/:id" element={<OrderDetailPage />} />
      </Route>
      <Route path="/member" element={<Navigate to="/member/member" replace />} />
      <Route element={<PermissionRoute permission="member:member:list" />}>
        <Route path="/member/member" element={<MemberListPage />} />
        <Route path="/member/member/:id" element={<MemberDetailPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
