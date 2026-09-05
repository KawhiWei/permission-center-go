import { useEffect, useMemo, useState } from 'react';
import { Button, Input, Loading, MessagePlugin, Select, Tag } from 'tdesign-react';
import {
  CheckCircleIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  CloudIcon,
  SearchIcon,
} from 'tdesign-icons-react';
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
  useServiceResources,
} from '../permission-center/shared';
import '../permission-center/style.less';

const pageSizeOptions = [6, 12, 24].map((value) => ({ label: `每页 ${value} 条`, value }));

const hasAudience = (serviceResource: ServiceResource) => Boolean(serviceResource.audience.trim());

const SelectServiceResourcePage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const {
    serviceResources,
    serviceResourcesLoading,
    serviceResourcesError,
    reloadServiceResources,
  } = useServiceResources();
  const [selectedServiceResource, setSelectedServiceResource] = useState(getStoredServiceResource);
  const [submitting, setSubmitting] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const [audienceFilter, setAudienceFilter] = useState('all');
  const [pageSize, setPageSize] = useState(6);
  const [currentPage, setCurrentPage] = useState(1);

  const activeServiceResources = useMemo(
    () => serviceResources.filter((item) => item.isActive),
    [serviceResources],
  );

  const filteredServiceResources = useMemo(() => {
    const query = searchValue.trim().toLocaleLowerCase();
    return activeServiceResources.filter((item) => {
      const matchesQuery = !query || [item.key, item.name, item.displayName, item.audience, item.description]
        .some((value) => value.toLocaleLowerCase().includes(query));
      const matchesAudience = audienceFilter === 'all'
        || (audienceFilter === 'with-audience' && hasAudience(item))
        || (audienceFilter === 'without-audience' && !hasAudience(item));
      return matchesQuery && matchesAudience;
    });
  }, [activeServiceResources, audienceFilter, searchValue]);

  const totalPages = Math.max(1, Math.ceil(filteredServiceResources.length / pageSize));
  const currentItems = useMemo(() => {
    const start = (currentPage - 1) * pageSize;
    return filteredServiceResources.slice(start, start + pageSize);
  }, [currentPage, filteredServiceResources, pageSize]);

  useEffect(() => {
    if (serviceResourcesLoading || serviceResourcesError) return;
    if (!activeServiceResources.some((item) => item.key === selectedServiceResource)) {
      setSelectedServiceResource('');
      if (getStoredServiceResource()) setStoredServiceResource('');
    }
  }, [activeServiceResources, selectedServiceResource, serviceResourcesError, serviceResourcesLoading]);

  useEffect(() => {
    setCurrentPage((page) => Math.min(page, totalPages));
  }, [totalPages]);

  const resetPagination = () => setCurrentPage(1);

  const handleContinue = () => {
    const serviceResource = selectedServiceResource.trim();
    if (!serviceResource || !activeServiceResources.some((item) => item.key === serviceResource)) {
      MessagePlugin.warning('请选择一个启用的服务资源');
      return;
    }

    setSubmitting(true);
    setStoredServiceResource(serviceResource);
    const requestedRedirect = getSafeRedirectPath(new URLSearchParams(location.search).get('redirect'));
    navigate(getPostServiceResourceSelectionPath(requestedRedirect), { replace: true });
  };

  const handlePageSizeChange = (value: unknown) => {
    const nextPageSize = Number(value);
    if (!pageSizeOptions.some((option) => option.value === nextPageSize)) return;
    setPageSize(nextPageSize);
    resetPagination();
  };

  const startIndex = filteredServiceResources.length ? (currentPage - 1) * pageSize + 1 : 0;
  const endIndex = Math.min(currentPage * pageSize, filteredServiceResources.length);

  return (
    <main className="service-resource-selection-page">
      <section className="service-resource-selection-shell" aria-labelledby="service-resource-title">
        <div className="service-resource-selection-intro">
          <div className="service-resource-login-status"><CheckCircleIcon size="16px" /><span>登录成功</span></div>
          <h1 id="service-resource-title">选择允许访问的服务资源</h1>
          <p>身份验证已完成。选择本次会话需要访问的资源，可随时在控制台中切换。</p>
        </div>

        <div className="service-resource-session-state">
          <span className="service-resource-session-avatar">已</span>
          <span><strong>已验证会话</strong><small>请选择一个启用的服务资源</small></span>
        </div>

        <div className="service-resource-filter-bar">
          <Input
            className="service-resource-search"
            value={searchValue}
            placeholder="搜索服务名称或资源标识"
            prefixIcon={<SearchIcon />}
            onChange={(value) => { setSearchValue(value); resetPagination(); }}
          />
          <Select
            className="service-resource-filter"
            value={audienceFilter}
            options={[
              { label: '全部接入标识', value: 'all' },
              { label: '已配置 Audience', value: 'with-audience' },
              { label: '未配置 Audience', value: 'without-audience' },
            ]}
            onChange={(value) => { setAudienceFilter(String(value)); resetPagination(); }}
          />
          <span className="service-resource-filter-count">{filteredServiceResources.length} 个可用资源</span>
        </div>

        {selectedServiceResource ? (
          <div className="service-resource-selected-notice"><CheckCircleIcon size="18px" /><span>已选择 1 个资源，访问范围将在继续后生效。</span></div>
        ) : null}

        {serviceResourcesLoading ? (
          <div className="service-resource-state-panel"><Loading size="small" text="正在加载服务资源列表" /></div>
        ) : serviceResourcesError ? (
          <div className="service-resource-state-panel service-resource-state-panel--error">
            <span>服务资源列表加载失败：{getRequestErrorMessage(serviceResourcesError, '请稍后重试')}</span>
            <Button variant="text" theme="primary" type="button" onClick={() => void reloadServiceResources()}>重试</Button>
          </div>
        ) : activeServiceResources.length === 0 ? (
          <div className="service-resource-state-panel"><strong>暂无可用服务资源</strong><span>请联系管理员配置服务资源后，再进入权限管理页面。</span></div>
        ) : filteredServiceResources.length === 0 ? (
          <div className="service-resource-state-panel"><strong>没有匹配的服务资源</strong><span>调整筛选条件或清空搜索后重试。</span></div>
        ) : (
          <div className="service-resource-card-grid">
            {currentItems.map((item) => {
              const isSelected = selectedServiceResource === item.key;
              return (
                <button
                  className={`service-resource-card${isSelected ? ' is-selected' : ''}`}
                  type="button"
                  key={item.id || item.key}
                  aria-pressed={isSelected}
                  onClick={() => setSelectedServiceResource(item.key)}
                >
                  <span className="service-resource-card-icon"><CloudIcon size="20px" /></span>
                  <span className="service-resource-card-copy">
                    <strong title={item.displayName || item.name || item.key}>{item.displayName || item.name || item.key}</strong>
                    <code title={item.key}>{item.key}</code>
                    <Tag className="service-resource-card-source" theme={item.source === 'local' ? 'primary' : 'default'} variant="light-outline">
                      {item.source === 'local' ? '本地' : 'NexusAuth'}
                    </Tag>
                  </span>
                  <span className="service-resource-card-check"><CheckCircleIcon size="17px" /></span>
                  <span className="service-resource-card-divider" />
                  <span className="service-resource-card-meta">
                    <span><small>接入标识</small><b title={item.audience || '未配置'}>{item.audience || '未配置'}</b></span>
                    <span><small>资源状态</small><b className="service-resource-card-status">启用</b></span>
                  </span>
                  <span className="service-resource-card-description" title={item.description || '暂未提供资源描述'}>{item.description || '暂未提供资源描述'}</span>
                </button>
              );
            })}
          </div>
        )}

        <footer className="service-resource-selection-footer">
          <span>显示 {startIndex} - {endIndex} 条，共 {filteredServiceResources.length} 条</span>
          <Select className="service-resource-page-size" value={pageSize} options={pageSizeOptions} onChange={handlePageSizeChange} />
          <div className="service-resource-pagination" aria-label="服务资源分页">
            <Button variant="outline" shape="square" disabled={currentPage === 1} icon={<ChevronLeftIcon />} onClick={() => setCurrentPage((page) => page - 1)} />
            {Array.from({ length: totalPages }, (_, index) => index + 1).slice(0, 3).map((page) => (
              <Button key={page} variant={page === currentPage ? 'base' : 'outline'} theme={page === currentPage ? 'primary' : 'default'} shape="square" onClick={() => setCurrentPage(page)}>{page}</Button>
            ))}
            {totalPages > 3 ? <span className="service-resource-page-ellipsis">...</span> : null}
            <Button variant="outline" shape="square" disabled={currentPage === totalPages} icon={<ChevronRightIcon />} onClick={() => setCurrentPage((page) => page + 1)} />
          </div>
          <Button className="service-resource-continue" theme="primary" type="button" loading={submitting} disabled={!selectedServiceResource} onClick={handleContinue}>继续并进入控制台</Button>
        </footer>
      </section>
    </main>
  );
};

export default SelectServiceResourcePage;
