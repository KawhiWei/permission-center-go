import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Space } from 'tdesign-react';
import {
  SERVICE_RESOURCE_CHANGE_EVENT,
  getStoredServiceResource,
  listServiceResources,
  setStoredServiceResource,
  type ServiceResource,
} from '../../api/permission';

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

type ServiceResourceCatalog = {
  serviceResources: ServiceResource[];
  serviceResourcesLoading: boolean;
  serviceResourcesError: string | null;
  reloadServiceResources: () => Promise<void>;
};

export const useServiceResourceCatalog = (): ServiceResourceCatalog => {
  const [serviceResources, setServiceResources] = useState<ServiceResource[]>([]);
  // Keep the cached scope usable until the catalog has been checked.
  const [serviceResourcesLoading, setServiceResourcesLoading] = useState(true);
  const [serviceResourcesError, setServiceResourcesError] = useState<string | null>(null);

  const reloadServiceResources = useCallback(async () => {
    setServiceResourcesLoading(true);
    setServiceResourcesError(null);
    try {
      setServiceResources(await listServiceResources());
    } catch (error) {
      setServiceResources([]);
      setServiceResourcesError(getRequestErrorMessage(error, '加载服务资源列表失败'));
    } finally {
      setServiceResourcesLoading(false);
    }
  }, []);

  useEffect(() => {
    void reloadServiceResources();
  }, [reloadServiceResources]);

  return {
    serviceResources,
    serviceResourcesLoading,
    serviceResourcesError,
    reloadServiceResources,
  };
};

export const useServiceResourceScope = () => {
  const [serviceResource, setServiceResource] = useState(getStoredServiceResource);
  const catalog = useServiceResourceCatalog();

  useEffect(() => {
    if (catalog.serviceResourcesLoading || catalog.serviceResourcesError) {
      return;
    }

    if (!serviceResource || !catalog.serviceResources.some((item) => (
      item.name === serviceResource && item.isActive
    ))) {
      setServiceResource('');
      setStoredServiceResource('');
    }
  }, [catalog.serviceResources, catalog.serviceResourcesError, catalog.serviceResourcesLoading, serviceResource]);

  useEffect(() => {
    const sync = () => {
      setServiceResource(getStoredServiceResource());
    };
    window.addEventListener(SERVICE_RESOURCE_CHANGE_EVENT, sync);
    window.addEventListener('storage', sync);
    return () => {
      window.removeEventListener(SERVICE_RESOURCE_CHANGE_EVENT, sync);
      window.removeEventListener('storage', sync);
    };
  }, []);

  return {
    serviceResource,
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
