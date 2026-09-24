import { useMemo } from 'react';
import { Card, Tabs } from 'antd';
import { useLocation, useNavigate } from 'react-router-dom';
import FileStoragePage from '@/modules/system/file/storage';
import AgreementSettingPage from './agreement';
import SystemInfoPage from './info';
import LogisticsSettingPage from './logistics';
import MemberSettingPage from './member';
import MpSettingPage from './mp';
import ReviewSettingPage from './review';
import SearchSettingPage from './search';
import ShopSettingPage from './shop';
import SiteSettingPage from './site';
import TradeSettingPage from './trade';
import WechatPaySetting from './wechat-pay';

const tabItems = [
  { key: 'shop', label: '商城设置' },
  { key: 'trade', label: '交易设置' },
  { key: 'wechat-pay', label: '支付设置' },
  { key: 'logistics', label: '配送设置' },
  { key: 'member', label: '用户设置' },
  { key: 'review', label: '评价设置' },
  { key: 'agreement', label: '协议设置' },
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
    if (pathname.includes('/trade')) return 'trade';
    if (pathname.includes('/wechat-pay') || pathname.includes('/payment')) return 'wechat-pay';
    if (pathname.includes('/logistics')) return 'logistics';
    if (pathname.includes('/member')) return 'member';
    if (pathname.includes('/review')) return 'review';
    if (pathname.includes('/agreement')) return 'agreement';
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
        {activeTab === 'trade' && <TradeSettingPage />}
        {activeTab === 'wechat-pay' && <WechatPaySetting />}
        {activeTab === 'logistics' && <LogisticsSettingPage />}
        {activeTab === 'member' && <MemberSettingPage />}
        {activeTab === 'review' && <ReviewSettingPage />}
        {activeTab === 'agreement' && <AgreementSettingPage />}
        {activeTab === 'site' && <SiteSettingPage />}
        {activeTab === 'mp' && <MpSettingPage />}
        {activeTab === 'search' && <SearchSettingPage />}
        {activeTab === 'storage' && <FileStoragePage card={false} />}
        {activeTab === 'info' && <SystemInfoPage />}
      </div>
    </Card>
  );
}
