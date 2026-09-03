import { ProCard } from '@ant-design/pro-components';
import { Route, Routes, useParams } from 'react-router-dom';

import { DemoPage } from '@/pages/DemoPage';
import { HelloPluginPage } from '@/pages/HelloPluginPage';
import { HomePage } from '@/pages/HomePage';

function MenuPage() {
  const { moduleKey, groupKey, menuKey } = useParams();

  return (
    <ProCard title={menuKey ?? 'Menu'} variant="outlined">
      <p>
        当前页面属于 {moduleKey ?? 'module'} / {groupKey ?? 'menu'}，可以在这里继续开发业务内容。
      </p>
    </ProCard>
  );
}

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/demo" element={<DemoPage />} />
      <Route path="/plugins/hello" element={<HelloPluginPage />} />
      <Route path="/:moduleKey/:groupKey/:menuKey" element={<MenuPage />} />
    </Routes>
  );
}
