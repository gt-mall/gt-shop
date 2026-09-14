import { Navigate, Outlet } from 'react-router-dom';

import { usePermission } from '@/framework/store';
import { getDefaultAuthorizedPath } from './authorized-pages';

export function PermissionRoute({ permission }: { permission: string }) {
  const hasPermission = usePermission();
  const defaultPath = getDefaultAuthorizedPath(hasPermission);
  return hasPermission(permission) ? <Outlet /> : <Navigate to={defaultPath} replace />;
}
