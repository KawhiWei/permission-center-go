import { Navigate, RouteObject, useLocation } from 'react-router-dom';

import ErrorPage from '../components/error';
import PublicLayout from '../layouts/layout';
import Login from '../pages/login';
import SelectServiceResource from '../pages/select-service-resource';
import {
  consumeLoginRedirect,
  getServiceResourceSelectionPath,
  getSafeRedirectPath,
  HomeRedirect,
  RedirectIfAuthenticated,
  RequireServiceResource,
  RequireAuth,
  setCachedAuthStatus,
} from './auth';

const AuthCallback = () => {
  const location = useLocation();
  const queryRedirect = getSafeRedirectPath(new URLSearchParams(location.search).get('redirect'));
  const redirect = queryRedirect || consumeLoginRedirect();
  setCachedAuthStatus(true);
  return <Navigate to={getServiceResourceSelectionPath(redirect)} replace />;
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
        path: '/select-service-resource',
        Component: SelectServiceResource,
        handle: { name: '选择服务资源' },
      },
      {
        element: <RequireServiceResource />,
        children: [
          {
            id: 'protected-layout',
            path: '/',
            Component: PublicLayout,
            children: [
              {
                path: '*',
                Component: ErrorPage,
              },
            ],
            errorElement: <ErrorPage />,
          },
        ],
      },
    ],
  },
];
