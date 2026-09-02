import { Navigate, RouteObject, useLocation } from 'react-router-dom';

import ErrorPage from '../components/error';
import PublicLayout from '../layouts/layout';
import Login from '../pages/login';
import Dashboard from '../pages/permission-center/dashboard';
import ApplicationManagement from '../pages/permission-center/application-management';
import RoleManagement from '../pages/permission-center/role-management';
import MenuManagement from '../pages/permission-center/menu-management';
import UserRoleManagement from '../pages/permission-center/user-role-management';
import SelectApplication from '../pages/select-application';
import {
  consumeLoginRedirect,
  getApplicationSelectionPath,
  getSafeRedirectPath,
  HomeRedirect,
  RedirectIfAuthenticated,
  RequireApplication,
  RequireAuth,
  setCachedAuthStatus,
} from './auth';

const AuthCallback = () => {
  const location = useLocation();
  const queryRedirect = getSafeRedirectPath(new URLSearchParams(location.search).get('redirect'));
  const redirect = queryRedirect || consumeLoginRedirect();
  setCachedAuthStatus(true);
  return <Navigate to={getApplicationSelectionPath(redirect)} replace />;
};

export const routes: RouteObject[] = [
  {
    element: <RedirectIfAuthenticated />,
    children: [
      {
        path: '/login',
        Component: Login,
      },
    ],
  },
  {
    path: '/auth/callback',
    Component: AuthCallback,
  },
  {
    path: '/',
    element: <HomeRedirect />,
  },
  {
    element: <RequireAuth />,
    children: [
      {
        id: 'protected-layout',
        path: '/',
        Component: PublicLayout,
        children: [
          {
            path: 'select-application',
            Component: SelectApplication,
            handle: { name: '选择应用' },
          },
          {
            path: 'applications',
            Component: ApplicationManagement,
            handle: { name: '应用管理' },
          },
          {
            element: <RequireApplication />,
            children: [
              {
                path: 'dashboard',
                Component: Dashboard,
                handle: { name: '仪表盘' },
              },
              {
                path: 'roles',
                Component: RoleManagement,
                handle: { name: '角色管理' },
              },
              {
                path: 'menus',
                Component: MenuManagement,
                handle: { name: '菜单与按钮' },
              },
              {
                path: 'user-roles',
                Component: UserRoleManagement,
                handle: { name: '用户角色绑定' },
              },
            ],
          },
          {
            path: '*',
            Component: ErrorPage,
          },
        ],
        errorElement: <ErrorPage />,
      },
    ],
  },
];
