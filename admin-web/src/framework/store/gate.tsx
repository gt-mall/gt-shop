import { useEffect, type ReactNode } from 'react';
import { Spin } from 'antd';

import { useAuth } from '@/framework/store';
import LoginPage from '@/modules/login';

export function StoreGate({ children }: { children: ReactNode }) {
  const { isLoggedIn, initialized, refreshCurrent, refreshSettings } = useAuth();

  useEffect(() => {
    void refreshCurrent();
    void refreshSettings();
  }, [refreshCurrent, refreshSettings]);

  if (!initialized) {
    return <Spin fullscreen description="正在加载用户权限" />;
  }
  return isLoggedIn ? children : <LoginPage />;
}
