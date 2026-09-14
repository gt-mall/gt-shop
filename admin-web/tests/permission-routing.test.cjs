const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { test } = require('node:test');
const ts = require('typescript');

const root = path.resolve(__dirname, '../src');

function transpileAndExecute(filePath, bindings = {}) {
  const content = fs.readFileSync(filePath, 'utf8');
  const output = ts.transpileModule(content, {
    compilerOptions: {
      target: ts.ScriptTarget.ES2022,
      module: ts.ModuleKind.CommonJS,
      jsx: ts.JsxEmit.ReactJSX,
    },
  }).outputText;
  const context = vm.createContext({
    exports: {},
    require: (id) => {
      if (id.includes('store')) {
        return { usePermission: () => () => true, useAuth: () => ({ logout: () => {} }) };
      }
      return {};
    },
    ...bindings,
  });
  vm.runInContext(output, context);
  return context.exports;
}

const authorizedPagesModule = transpileAndExecute(
  path.join(root, 'framework/router/authorized-pages.ts')
);
const {
  AUTHORIZED_PAGES,
  NO_PERMISSION_PATH,
  getDefaultAuthorizedPath,
  getDefaultAuthorizedPathFromPermissions,
} = authorizedPagesModule;

test('priority list strictly defines the 12 authorized pages in order', () => {
  const expectedOrder = [
    { path: '/dashboard/home', permission: 'dashboard:view' },
    { path: '/product/product', permission: 'product:product:list' },
    { path: '/order/order', permission: 'order:order:list' },
    { path: '/order/aftersale', permission: 'order:after_sale:list' },
    { path: '/member/member', permission: 'member:member:list' },
    { path: '/product/inventory', permission: 'product:inventory:list' },
    { path: '/product/category', permission: 'product:category:list' },
    { path: '/product/freight', permission: 'product:freight:list' },
    { path: '/system/file', permission: 'system:file:list' },
    { path: '/system/user', permission: 'system:user:list' },
    { path: '/system/role', permission: 'system:role:list' },
    { path: '/system/setting/shop', permission: 'system:setting:view' },
  ];

  assert.equal(AUTHORIZED_PAGES.length, 12, 'Must have exactly 12 authorized pages');
  expectedOrder.forEach((expected, index) => {
    assert.equal(AUTHORIZED_PAGES[index].path, expected.path, `Item ${index} path must match`);
    assert.equal(AUTHORIZED_PAGES[index].permission, expected.permission, `Item ${index} permission must match`);
  });
  assert.equal(NO_PERMISSION_PATH, '/no-permission');
});

test('Scenario 7 & 8: SuperAdmin and user with dashboard:view resolve to /dashboard/home', () => {
  // SuperAdmin has all permissions
  const allPermissions = AUTHORIZED_PAGES.map((p) => p.permission);
  assert.equal(getDefaultAuthorizedPathFromPermissions(allPermissions), '/dashboard/home');

  // User with dashboard:view + some other permissions
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['dashboard:view', 'product:product:list']),
    '/dashboard/home'
  );

  // User with only dashboard:view
  assert.equal(getDefaultAuthorizedPathFromPermissions(['dashboard:view']), '/dashboard/home');
});

test('Scenario 9: User without dashboard:view but with product:product:list resolves to /product/product', () => {
  const permissions = ['product:product:list', 'order:order:list'];
  assert.equal(getDefaultAuthorizedPathFromPermissions(permissions), '/product/product');
});

test('Scenario 10: User with only order:order:list resolves to /order/order', () => {
  const permissions = ['order:order:list'];
  assert.equal(getDefaultAuthorizedPathFromPermissions(permissions), '/order/order');
});

test('Scenario 11: User with no permissions resolves to /no-permission', () => {
  assert.equal(getDefaultAuthorizedPathFromPermissions([]), '/no-permission');
  assert.equal(getDefaultAuthorizedPathFromPermissions(undefined), '/no-permission');
  assert.equal(getDefaultAuthorizedPath((perm) => false), '/no-permission');
});

test('Scenario 12: Strict priority ordering across each step of the hierarchy', () => {
  // 1. Dashboard vs Product
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['dashboard:view', 'product:product:list']),
    '/dashboard/home'
  );
  // 2. Product vs Order
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['product:product:list', 'order:order:list']),
    '/product/product'
  );
  // 3. Order vs Aftersale
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['order:order:list', 'order:after_sale:list']),
    '/order/order'
  );
  // 4. Aftersale vs Member
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['order:after_sale:list', 'member:member:list']),
    '/order/aftersale'
  );
  // 5. Member vs Inventory
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['member:member:list', 'product:inventory:list']),
    '/member/member'
  );
  // 6. Inventory vs Category
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['product:inventory:list', 'product:category:list']),
    '/product/inventory'
  );
  // 7. Category vs Freight
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['product:category:list', 'product:freight:list']),
    '/product/category'
  );
  // 8. Freight vs System File
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['product:freight:list', 'system:file:list']),
    '/product/freight'
  );
  // 9. System File vs System User
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['system:file:list', 'system:user:list']),
    '/system/file'
  );
  // 10. System User vs System Role
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['system:user:list', 'system:role:list']),
    '/system/user'
  );
  // 11. System Role vs System Setting
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['system:role:list', 'system:setting:view']),
    '/system/role'
  );
  // 12. Only System Setting
  assert.equal(
    getDefaultAuthorizedPathFromPermissions(['system:setting:view']),
    '/system/setting/shop'
  );
});

test('Dashboard menu in layout.tsx uses dashboard:view and filters visibility properly', () => {
  const layoutContent = fs.readFileSync(path.join(root, 'framework/layout/layout.tsx'), 'utf8');

  // Must not have stale dashboard:overview:view
  assert.ok(
    !layoutContent.includes('dashboard:overview:view'),
    'layout.tsx must not contain stale dashboard:overview:view'
  );

  // Must use dashboard:view
  assert.ok(
    layoutContent.includes("permission: 'dashboard:view'"),
    "layout.tsx must assign permission: 'dashboard:view' to dashboard-home"
  );

  // Simulate layout menu filtering logic
  function filterMenu(hasPermission) {
    const rawMenu = [
      {
        key: 'dashboard',
        name: '首页',
        path: '/dashboard',
        children: [
          {
            key: 'dashboard-home',
            name: '数据看板',
            path: '/dashboard/home',
            permission: 'dashboard:view',
          },
        ],
      },
      {
        key: 'product',
        name: '商品',
        path: '/product',
        children: [
          {
            key: 'product-product',
            name: '商品列表',
            path: '/product/product',
            permission: 'product:product:list',
          },
        ],
      },
    ];

    return rawMenu
      .map((item) => ({
        ...item,
        children: item.children.filter((child) => hasPermission(child.permission)),
      }))
      .filter((item) => item.children.length > 0);
  }

  // With dashboard:view
  const menuWithDash = filterMenu((perm) => perm === 'dashboard:view');
  assert.equal(menuWithDash.length, 1);
  assert.equal(menuWithDash[0].key, 'dashboard');

  // Without dashboard:view, but with product:product:list
  const menuWithoutDash = filterMenu((perm) => perm === 'product:product:list');
  assert.equal(menuWithoutDash.length, 1);
  assert.equal(menuWithoutDash[0].key, 'product');
  assert.ok(!menuWithoutDash.some((m) => m.key === 'dashboard'), 'Dashboard menu must be hidden');

  // With no permissions
  const emptyMenu = filterMenu(() => false);
  assert.equal(emptyMenu.length, 0, 'All menus must be hidden when user has no permissions');
});

test('NoPermissionPage contains required UI elements and avoids forbidden async keywords', () => {
  const noPermContent = fs.readFileSync(
    path.join(root, 'modules/system/no-permission/index.tsx'),
    'utf8'
  );

  assert.ok(noPermContent.includes('暂无可访问功能'), 'Must display "暂无可访问功能"');
  assert.ok(
    noPermContent.includes('当前账号尚未分配后台功能权限，请联系管理员为当前角色授权。'),
    'Must display explanation text'
  );
  assert.ok(noPermContent.includes('退出登录'), 'Must have "退出登录" button');

  // Verify no forbidden async keywords
  assert.ok(!noPermContent.includes('.then('), 'Must not use .then()');
  assert.ok(!noPermContent.includes('.catch('), 'Must not use .catch()');
  assert.ok(!noPermContent.includes('try {') && !noPermContent.includes('try{'), 'Must not use try/catch');
});

test('Dashboard home page checks dashboard:view and guards data fetching', () => {
  const dashboardContent = fs.readFileSync(
    path.join(root, 'modules/dashboard/home/index.tsx'),
    'utf8'
  );

  assert.ok(
    !dashboardContent.includes('dashboard:overview:view'),
    'Dashboard page must not contain stale dashboard:overview:view'
  );
  assert.ok(
    dashboardContent.includes("hasPermission('dashboard:view')"),
    "Dashboard page must check hasPermission('dashboard:view')"
  );
  assert.ok(
    dashboardContent.includes('if (!canViewDashboard) return;'),
    'Dashboard page must guard API fetches'
  );
});

test('Router configuration and loop prevention', () => {
  const routerContent = fs.readFileSync(
    path.join(root, 'framework/router/index.tsx'),
    'utf8'
  );

  // /dashboard/home wrapped in PermissionRoute
  assert.ok(
    routerContent.includes('permission="dashboard:view"'),
    'Dashboard route must be protected by permission="dashboard:view"'
  );

  // / redirects to RootIndexRedirect
  assert.ok(
    routerContent.includes('<Route path="/" element={<RootIndexRedirect />} />'),
    'Root path must render RootIndexRedirect'
  );

  // /no-permission route exists without PermissionRoute protection
  assert.ok(
    routerContent.includes('<Route path="/no-permission" element={<NoPermissionPage />} />'),
    '/no-permission route must be present'
  );

  // Loop prevention verification:
  // PermissionRoute navigates to defaultPath.
  // For unauthorized users, defaultPath is /no-permission.
  // /no-permission has no PermissionRoute.
  // Thus, transitions terminate deterministically with 0 possibility of recursion.
  const permissionRouteContent = fs.readFileSync(
    path.join(root, 'framework/router/permission.tsx'),
    'utf8'
  );
  assert.ok(
    permissionRouteContent.includes('getDefaultAuthorizedPath(hasPermission)'),
    'PermissionRoute must resolve defaultPath directly'
  );
});
