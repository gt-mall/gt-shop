import { useEffect, type ReactNode } from 'react';
import { Spin } from 'antd';

import { useAuth } from '@/framework/store';
import LoginPage from '@/modules/login';

export function StoreGate({ children }: { children: ReactNode }) {
  const { isLoggedIn, initialized, refreshCurrent } = useAuth();

  useEffect(() => {
    void refreshCurrent();
  }, [refreshCurrent]);

  if (!initialized) {
    return <Spin fullscreen tip="正在加载用户权限" />;
  }
  return isLoggedIn ? children : <LoginPage />;
}
