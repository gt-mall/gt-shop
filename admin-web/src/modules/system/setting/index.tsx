import { useMemo } from 'react';
import { Card, Tabs } from 'antd';
import { useLocation, useNavigate } from 'react-router-dom';
import FileStoragePage from '@/modules/system/file/storage';
import ShopSettingPage from './shop';
import SiteSettingPage from './site';

const tabItems = [
  { key: 'shop', label: '商城基础' },
  { key: 'site', label: '站点信息' },
  { key: 'storage', label: '文件存储' },
];

export default function SystemSettingPage() {
  const location = useLocation();
  const navigate = useNavigate();

  const activeTab = useMemo(() => {
    const pathname = location.pathname;
    if (pathname.includes('/site')) return 'site';
    if (pathname.includes('/storage') || pathname.includes('/file')) return 'storage';
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
        {activeTab === 'storage' && <FileStoragePage card={false} />}
      </div>
    </Card>
  );
}
