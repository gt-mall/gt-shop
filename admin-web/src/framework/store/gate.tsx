import { useEffect, type ReactNode } from 'react';
import { Spin } from 'antd';

import { useAppStore } from '@/framework/store';
import LoginPage from '@/modules/login';

export function StoreGate({ children }: { children: ReactNode }) {
  const token = useAppStore((state) => state.token);
  const current = useAppStore((state) => state.current);
  const loading = useAppStore((state) => state.loading);
  const initialized = useAppStore((state) => state.initialized);
  const refreshCurrent = useAppStore((state) => state.refreshCurrent);

  useEffect(() => {
    void refreshCurrent();
  }, [refreshCurrent]);

  if (!initialized || (token && loading && !current)) {
    return <Spin fullscreen tip="正在加载用户权限" />;
  }
  if (!token || !current) {
    return <LoginPage />;
  }
  return children;
}
