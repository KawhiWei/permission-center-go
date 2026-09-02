import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Button, Form, Select, Space } from 'tdesign-react';
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
  const [applicationsLoading, setApplicationsLoading] = useState(false);
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
  const [draftApplication, setDraftApplication] = useState(application);
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
    setDraftApplication('');
    setStoredApplication('');
  }, [application, catalog.applications, catalog.applicationsError, catalog.applicationsLoading]);

  useEffect(() => {
    const sync = () => {
      const next = getStoredApplication();
      setApplication(next);
      setDraftApplication(next);
    };
    window.addEventListener(APPLICATION_CHANGE_EVENT, sync);
    window.addEventListener('storage', sync);
    return () => {
      window.removeEventListener(APPLICATION_CHANGE_EVENT, sync);
      window.removeEventListener('storage', sync);
    };
  }, []);

  const applyApplication = useCallback(() => {
    const next = setStoredApplication(draftApplication);
    setApplication(next);
    setDraftApplication(next);
    return next;
  }, [draftApplication]);

  return {
    application,
    draftApplication,
    setDraftApplication,
    applyApplication,
    ...catalog,
  };
};

export const ApplicationField = ({
  draftApplication,
  onDraftChange,
  onApply,
  loading = false,
  applications = [],
  applicationsLoading = false,
  applicationsError = null,
  onRefreshApplications,
}: {
  draftApplication: string;
  onDraftChange: (value: string) => void;
  onApply: () => void;
  loading?: boolean;
  applications?: Application[];
  applicationsLoading?: boolean;
  applicationsError?: string | null;
  onRefreshApplications?: () => void;
}) => {
  const activeApplications = applications.filter((item) => item.enabled && !item.isDeleted);
  const hasSelectedApplication = activeApplications.some((item) => item.application === draftApplication);

  return (
    <div className="permission-application-field">
      <Form layout="inline" className="permission-toolbar-form" onSubmit={(context) => {
        context.e?.preventDefault();
        onApply();
      }}>
        <Form.FormItem label="应用">
          <Select
            value={hasSelectedApplication ? draftApplication : ''}
            options={activeApplications.map((item) => ({
              label: item.name ? `${item.name} (${item.application})` : item.application,
              value: item.application,
            }))}
            loading={applicationsLoading}
            disabled={applicationsLoading || activeApplications.length === 0}
            placeholder={applicationsLoading ? '正在加载应用...' : '请选择应用'}
            filterable
            style={{ width: 280 }}
            onChange={(value) => onDraftChange(String(value || ''))}
          />
        </Form.FormItem>
        <Form.FormItem>
          <Button
            theme="primary"
            type="button"
            loading={loading}
            disabled={!hasSelectedApplication || applicationsLoading || applicationsError !== null}
            onClick={onApply}
          >应用</Button>
        </Form.FormItem>
      </Form>
      {applicationsError ? (
        <div className="permission-application-field-error">
          <span>应用列表加载失败：{applicationsError}</span>
          {onRefreshApplications ? (
            <Button variant="text" theme="primary" type="button" onClick={onRefreshApplications}>重试</Button>
          ) : null}
        </div>
      ) : !applicationsLoading && activeApplications.length === 0 ? (
        <span className="permission-application-field-empty">暂无可用应用，请先在应用管理中创建。</span>
      ) : null}
    </div>
  );
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
