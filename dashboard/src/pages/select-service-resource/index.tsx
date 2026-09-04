import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Loading, MessagePlugin, Radio, Space, Tag } from 'tdesign-react';
import { useLocation, useNavigate } from 'react-router-dom';

import {
  getStoredServiceResource,
  setStoredServiceResource,
  type ServiceResource,
} from '../../api/permission';
import {
  getPostServiceResourceSelectionPath,
  getSafeRedirectPath,
} from '../../router/auth';
import {
  getRequestErrorMessage,
  useServiceResourceCatalog,
} from '../permission-center/shared';
import '../permission-center/style.less';

const isActiveServiceResource = (serviceResource: ServiceResource) => serviceResource.isActive;

const SelectServiceResourcePage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const {
    serviceResources,
    serviceResourcesLoading,
    serviceResourcesError,
    reloadServiceResources,
  } = useServiceResourceCatalog();
  const [selectedServiceResource, setSelectedServiceResource] = useState(getStoredServiceResource);
  const [submitting, setSubmitting] = useState(false);

  const activeServiceResources = useMemo(
    () => serviceResources.filter(isActiveServiceResource),
    [serviceResources],
  );

  useEffect(() => {
    if (serviceResourcesLoading || serviceResourcesError) {
      return;
    }

    if (!activeServiceResources.some((item) => item.name === selectedServiceResource)) {
      setSelectedServiceResource('');
      if (getStoredServiceResource()) {
        setStoredServiceResource('');
      }
    }
  }, [activeServiceResources, selectedServiceResource, serviceResourcesError, serviceResourcesLoading]);

  const handleContinue = () => {
    const serviceResource = selectedServiceResource.trim();
    if (!serviceResource || !activeServiceResources.some((item) => item.name === serviceResource)) {
      MessagePlugin.warning('请选择一个启用的服务资源');
      return;
    }

    setSubmitting(true);
    setStoredServiceResource(serviceResource);
    const requestedRedirect = getSafeRedirectPath(new URLSearchParams(location.search).get('redirect'));
    navigate(getPostServiceResourceSelectionPath(requestedRedirect), { replace: true });
  };

  const handleReload = () => {
    void reloadServiceResources();
  };

  return (
    <div className="permission-page permission-select-service-resource-page">
      <div className="permission-select-service-resource-heading">
        <div>
          <Tag theme="primary" variant="light-outline">权限上下文</Tag>
          <h1>选择服务资源</h1>
          <p>请选择要管理的服务资源。角色、菜单、按钮和 PDP 配置都会按服务资源隔离。</p>
        </div>
      </div>

      <Card className="permission-card permission-select-service-resource-card" bordered>
        <div className="permission-card-title">
          <strong>可用服务资源</strong>
          <span>{activeServiceResources.length} 个启用服务资源</span>
        </div>

        {serviceResourcesLoading ? (
          <div className="permission-select-service-resource-loading">
            <Loading size="small" text="正在加载服务资源列表" />
          </div>
        ) : serviceResourcesError ? (
          <div className="permission-load-error permission-select-service-resource-error">
            <span>服务资源列表加载失败：{getRequestErrorMessage(serviceResourcesError, '请稍后重试')}</span>
            <Button variant="text" theme="primary" type="button" onClick={handleReload}>重试</Button>
          </div>
        ) : activeServiceResources.length === 0 ? (
          <div className="permission-select-service-resource-empty">
            <strong>暂无可用服务资源</strong>
            <span>请联系管理员配置服务资源后，再进入权限管理页面。</span>
          </div>
        ) : (
          <>
            <Radio.Group
              value={selectedServiceResource}
              onChange={(value) => setSelectedServiceResource(String(value))}
            >
              <div className="permission-service-resource-option-grid">
                {activeServiceResources.map((item) => (
                  <label
                    className={`permission-service-resource-option${selectedServiceResource === item.name ? ' is-selected' : ''}`}
                    key={item.name}
                  >
                    <Radio value={item.name} />
                    <span className="permission-service-resource-option-copy">
                      <strong title={item.displayName || item.name}>{item.displayName || item.name}</strong>
                      <code title={item.name}>{item.name}</code>
                      {item.audience ? <span title={item.audience}>Audience: {item.audience}</span> : null}
                      {item.description ? <span title={item.description}>{item.description}</span> : null}
                    </span>
                  </label>
                ))}
              </div>
            </Radio.Group>
            <div className="permission-select-service-resource-actions">
              <Space>
                <Button variant="outline" type="button" onClick={handleReload}>刷新列表</Button>
                <Button
                  theme="primary"
                  type="button"
                  loading={submitting}
                  disabled={!selectedServiceResource}
                  onClick={handleContinue}
                >
                  进入权限中心
                </Button>
              </Space>
            </div>
          </>
        )}
      </Card>
    </div>
  );
};

export default SelectServiceResourcePage;
