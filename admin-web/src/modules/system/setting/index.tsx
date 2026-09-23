import { useMemo } from 'react';
import { Card, Tabs } from 'antd';
import { useLocation, useNavigate } from 'react-router-dom';
import FileStoragePage from '@/modules/system/file/storage';
import SystemInfoPage from './info';
import MpSettingPage from './mp';
import SearchSettingPage from './search';
import ShopSettingPage from './shop';
import SiteSettingPage from './site';

const tabItems = [
  { key: 'shop', label: '商城基础' },
  { key: 'site', label: '站点信息' },
  { key: 'mp', label: '小程序设置' },
  { key: 'search', label: '搜索设置' },
  { key: 'storage', label: '文件存储' },
  { key: 'info', label: '系统信息' },
];

export default function SystemSettingPage() {
  const location = useLocation();
  const navigate = useNavigate();

  const activeTab = useMemo(() => {
    const pathname = location.pathname;
    if (pathname.includes('/site')) return 'site';
    if (pathname.includes('/mp')) return 'mp';
    if (pathname.includes('/search')) return 'search';
    if (pathname.includes('/storage') || pathname.includes('/file')) return 'storage';
    if (pathname.includes('/info')) return 'info';
    return 'shop';
  }, [location.pathname]);

  const handleTabChange = (key: string) => {
    navigate(`/system/setting/${key}`);
  };

  return (
    <Card>
      <Tabs
        activeKey={activeTab}
        onChange={handleTabChange}
        items={tabItems}
      />
      <div style={{ marginTop: 16 }}>
        {activeTab === 'shop' && <ShopSettingPage />}
        {activeTab === 'site' && <SiteSettingPage />}
        {activeTab === 'mp' && <MpSettingPage />}
        {activeTab === 'search' && <SearchSettingPage />}
        {activeTab === 'storage' && <FileStoragePage card={false} />}
        {activeTab === 'info' && <SystemInfoPage />}
      </div>
    </Card>
  );
}
