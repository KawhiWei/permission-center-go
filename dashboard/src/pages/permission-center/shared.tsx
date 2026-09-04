import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Space } from 'tdesign-react';
import {
  APPLICATION_CHANGE_EVENT,
  listApplications,
  type Application,
  getStoredApplication,
  setStoredApplication,
} from '../../api/permission';

export const APPLICATION_CATALOG_CHANGE_EVENT = 'permission-center-application-catalog-change';

export const notifyApplicationCatalogChanged = () => {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event(APPLICATION_CATALOG_CHANGE_EVENT));
  }
};

export const getRequestErrorMessage = (error: unknown, fallback: string): string => {
  if (error && typeof error === 'object') {
    const response = (error as { response?: { data?: unknown } }).response;
    const data = response?.data;
    if (data && typeof data === 'object') {
      const record = data as Record<string, unknown>;
      const message = record.message || record.detail || record.errorMessage || record.error;
      if (typeof message === 'string' && message.trim()) {
        return message;
      }
    }
    const message = (error as { message?: unknown }).message;
    if (typeof message === 'string' && message.trim()) {
      return message;
    }
  }
  return fallback;
};

type ApplicationCatalog = {
  applications: Application[];
  applicationsLoading: boolean;
  applicationsError: string | null;
  reloadApplications: () => Promise<void>;
};

export const useApplicationCatalog = (): ApplicationCatalog => {
  const [applications, setApplications] = useState<Application[]>([]);
  // The initial request must be considered loading so a cached application is
  // not cleared before the catalog has been checked.
  const [applicationsLoading, setApplicationsLoading] = useState(true);
  const [applicationsError, setApplicationsError] = useState<string | null>(null);

  const reloadApplications = useCallback(async () => {
    setApplicationsLoading(true);
    setApplicationsError(null);
    try {
      setApplications(await listApplications());
    } catch (error) {
      setApplications([]);
      setApplicationsError(getRequestErrorMessage(error, '加载应用列表失败'));
    } finally {
      setApplicationsLoading(false);
    }
  }, []);

  useEffect(() => {
    void reloadApplications();
    const refresh = () => {
      void reloadApplications();
    };
    window.addEventListener(APPLICATION_CATALOG_CHANGE_EVENT, refresh);
    return () => {
      window.removeEventListener(APPLICATION_CATALOG_CHANGE_EVENT, refresh);
    };
  }, [reloadApplications]);

  return { applications, applicationsLoading, applicationsError, reloadApplications };
};

export const useApplicationScope = () => {
  const [application, setApplication] = useState(getStoredApplication);
  const catalog = useApplicationCatalog();

  useEffect(() => {
    if (catalog.applicationsLoading || catalog.applicationsError) {
      return;
    }

    if (!application || catalog.applications.some((item) => (
      item.application === application && item.enabled && !item.isDeleted
    ))) {
      return;
    }

    setApplication('');
    setStoredApplication('');
  }, [application, catalog.applications, catalog.applicationsError, catalog.applicationsLoading]);

  useEffect(() => {
    const sync = () => {
      const next = getStoredApplication();
      setApplication(next);
    };
    window.addEventListener(APPLICATION_CHANGE_EVENT, sync);
    window.addEventListener('storage', sync);
    return () => {
      window.removeEventListener(APPLICATION_CHANGE_EVENT, sync);
      window.removeEventListener('storage', sync);
    };
  }, []);

  return {
    application,
    ...catalog,
  };
};

export const PageHeader = ({
  title,
  description,
  actions,
}: {
  title: string;
  description: string;
  actions?: ReactNode;
}) => (
  <div className="permission-page-header">
    <div>
      <h1>{title}</h1>
      <p>{description}</p>
    </div>
    {actions ? <Space>{actions}</Space> : null}
  </div>
);

export const formatDateTime = (value?: string): string => {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false });
};
