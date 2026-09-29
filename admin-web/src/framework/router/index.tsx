import { lazy, Suspense } from 'react';
import { Spin } from 'antd';
import { Navigate, Route, Routes } from 'react-router-dom';

import { PermissionRoute } from './permission';
import {
  AUTHORIZED_PAGES,
  NO_PERMISSION_PATH,
  getDefaultAuthorizedPath,
  getDefaultAuthorizedPathFromPermissions,
  useDefaultAuthorizedPath,
  type AuthorizedPageConfig,
} from './authorized-pages';

const DemoPage = lazy(() => import('@/modules/demo'));
const DashboardHomePage = lazy(() => import('@/modules/dashboard/home'));
const HelloPluginPage = lazy(() => import('@/modules/plugin/hello'));
const RolePage = lazy(() => import('@/modules/system/role'));
const RolePermissionPage = lazy(() => import('@/modules/system/role/permission'));
const UserPage = lazy(() => import('@/modules/system/user'));
const FilePage = lazy(() => import('@/modules/system/file'));
const FileStoragePage = lazy(() => import('@/modules/system/file/storage'));
const SystemJobPage = lazy(() => import('@/modules/system/job'));
const SystemSettingPage = lazy(() => import('@/modules/system/setting'));
const ProductDictServicePage = lazy(() => import('@/modules/system/product-service'));
const ProductDictUnitPage = lazy(() => import('@/modules/system/product-unit'));
const ImageDemoPage = lazy(() => import('@/modules/demo/image'));
const EditorDemoPage = lazy(() => import('@/modules/demo/editor'));
const ProductCategoryPage = lazy(() => import('@/modules/product/category'));
const ProductListPage = lazy(() => import('@/modules/product/product'));
const ProductEditPage = lazy(() => import('@/modules/product/product/edit'));
const ProductInventoryPage = lazy(() => import('@/modules/product/inventory'));
const ProductInventoryLogPage = lazy(() => import('@/modules/product/inventory/log'));
const FreightListPage = lazy(() => import('@/modules/product/freight'));
const FreightEditPage = lazy(() => import('@/modules/product/freight/edit'));
const ProductReviewPage = lazy(() => import('@/modules/product/review'));
const MemberListPage = lazy(() => import('@/modules/member/member'));
const MemberDetailPage = lazy(() => import('@/modules/member/member/detail'));
const OrderListPage = lazy(() => import('@/modules/order/order'));
const OrderDetailPage = lazy(() => import('@/modules/order/order/detail'));
const AfterSaleListPage = lazy(() => import('@/modules/order/aftersale'));
const AfterSaleDetailPage = lazy(() => import('@/modules/order/aftersale/detail'));
const AnalyticsOverviewPage = lazy(() => import('@/modules/analytics/overview'));
const AnalyticsTradePage = lazy(() => import('@/modules/analytics/trade'));
const AnalyticsProductPage = lazy(() => import('@/modules/analytics/product'));
const AnalyticsMemberPage = lazy(() => import('@/modules/analytics/member'));
const AnalyticsFlowPage = lazy(() => import('@/modules/analytics/flow'));
const NoPermissionPage = lazy(() => import('@/modules/system/no-permission'));

export {
  AUTHORIZED_PAGES,
  NO_PERMISSION_PATH,
  getDefaultAuthorizedPath,
  getDefaultAuthorizedPathFromPermissions,
  useDefaultAuthorizedPath,
  type AuthorizedPageConfig,
};

function RootIndexRedirect() {
  const defaultPath = useDefaultAuthorizedPath();
  return <Navigate to={defaultPath} replace />;
}

export function AppRoutes() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-64 items-center justify-center gap-3">
          <Spin size="large" />
          <span>页面加载中</span>
        </div>
      }
    >
      <Routes>
        <Route path="/" element={<RootIndexRedirect />} />
        <Route path="/no-permission" element={<NoPermissionPage />} />
        <Route path="/demo" element={<DemoPage />} />
        <Route element={<PermissionRoute permission="dashboard:view" />}>
          <Route path="/dashboard" element={<Navigate to="/dashboard/home" replace />} />
          <Route path="/dashboard/home" element={<DashboardHomePage />} />
        </Route>
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
          <Route path="/system/setting/trade" element={<SystemSettingPage />} />
          <Route path="/system/setting/wechat-pay" element={<SystemSettingPage />} />
          <Route path="/system/setting/payment" element={<Navigate to="/system/setting/wechat-pay" replace />} />
          <Route path="/system/setting/logistics" element={<SystemSettingPage />} />
          <Route path="/system/setting/member" element={<SystemSettingPage />} />
          <Route path="/system/setting/review" element={<SystemSettingPage />} />
          <Route path="/system/setting/agreement" element={<SystemSettingPage />} />
          <Route path="/system/setting/site" element={<SystemSettingPage />} />
          <Route path="/system/setting/mp" element={<SystemSettingPage />} />
          <Route path="/system/setting/search" element={<SystemSettingPage />} />
          <Route path="/system/setting/storage" element={<SystemSettingPage />} />
          <Route path="/system/setting/info" element={<SystemSettingPage />} />
          <Route path="/system/setting/file" element={<Navigate to="/system/setting/storage" replace />} />
        </Route>
        <Route element={<PermissionRoute permission="product:dict_service:list" />}>
          <Route path="/system/product-service" element={<ProductDictServicePage />} />
        </Route>
        <Route element={<PermissionRoute permission="product:dict_unit:list" />}>
          <Route path="/system/product-unit" element={<ProductDictUnitPage />} />
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
        <Route element={<PermissionRoute permission="system:job:list" />}>
          <Route path="/system/job" element={<SystemJobPage />} />
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
          <Route path="/product/inventory_log" element={<ProductInventoryLogPage />} />
          <Route path="/product/inventory/log" element={<Navigate to="/product/inventory_log" replace />} />
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
        <Route element={<PermissionRoute permission="product:review:list" />}>
          <Route path="/product/review" element={<ProductReviewPage />} />
        </Route>
        <Route path="/order" element={<Navigate to="/order/order" replace />} />
        <Route element={<PermissionRoute permission="order:order:list" />}>
          <Route path="/order/order" element={<OrderListPage />} />
        </Route>
        <Route element={<PermissionRoute permission="order:order:detail" />}>
          <Route path="/order/order/:id" element={<OrderDetailPage />} />
        </Route>
        <Route element={<PermissionRoute permission="order:after_sale:list" />}>
          <Route path="/order/aftersale" element={<AfterSaleListPage />} />
        </Route>
        <Route element={<PermissionRoute permission="order:after_sale:detail" />}>
          <Route path="/order/aftersale/:id" element={<AfterSaleDetailPage />} />
        </Route>
        <Route path="/member" element={<Navigate to="/member/member" replace />} />
        <Route element={<PermissionRoute permission="member:member:list" />}>
          <Route path="/member/member" element={<MemberListPage />} />
          <Route path="/member/member/:id" element={<MemberDetailPage />} />
        </Route>
        <Route path="/analytics" element={<Navigate to="/analytics/overview" replace />} />
        <Route element={<PermissionRoute permission="analytics:overview:view" />}>
          <Route path="/analytics/overview" element={<AnalyticsOverviewPage />} />
        </Route>
        <Route element={<PermissionRoute permission="analytics:trade:view" />}>
          <Route path="/analytics/trade" element={<AnalyticsTradePage />} />
        </Route>
        <Route element={<PermissionRoute permission="analytics:product:view" />}>
          <Route path="/analytics/product" element={<AnalyticsProductPage />} />
        </Route>
        <Route element={<PermissionRoute permission="analytics:member:view" />}>
          <Route path="/analytics/member" element={<AnalyticsMemberPage />} />
        </Route>
        <Route element={<PermissionRoute permission="analytics:flow:view" />}>
          <Route path="/analytics/flow" element={<AnalyticsFlowPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  );
}
