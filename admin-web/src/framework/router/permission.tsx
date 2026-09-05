import { Navigate, Outlet } from 'react-router-dom';

import { usePermission } from '@/framework/store';

export function PermissionRoute({ permission }: { permission: string }) {
  const hasPermission = usePermission();
  return hasPermission(permission) ? <Outlet /> : <Navigate to="/" replace />;
}
