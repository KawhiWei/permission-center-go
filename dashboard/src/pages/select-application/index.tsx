import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Loading, MessagePlugin, Radio, Space, Tag } from 'tdesign-react';
import { useLocation, useNavigate } from 'react-router-dom';

import {
  getStoredApplication,
  setStoredApplication,
  type Application,
} from '../../api/permission';
import {
  getPostApplicationSelectionPath,
  getSafeRedirectPath,
} from '../../router/auth';
import {
  getRequestErrorMessage,
  useApplicationCatalog,
} from '../permission-center/shared';
import '../permission-center/style.less';

const isEnabledApplication = (application: Application) => application.enabled && !application.isDeleted;

const SelectApplicationPage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const {
    applications,
    applicationsLoading,
    applicationsError,
    reloadApplications,
  } = useApplicationCatalog();
  const [selectedApplication, setSelectedApplication] = useState(getStoredApplication);
  const [submitting, setSubmitting] = useState(false);

  const activeApplications = useMemo(
    () => applications.filter(isEnabledApplication),
    [applications],
  );

  useEffect(() => {
    if (applicationsLoading || applicationsError) {
      return;
    }

    if (!activeApplications.some((item) => item.application === selectedApplication)) {
      setSelectedApplication('');
      if (getStoredApplication()) {
        setStoredApplication('');
      }
    }
  }, [activeApplications, applicationsError, applicationsLoading, selectedApplication]);

  const handleContinue = () => {
    const application = selectedApplication.trim();
    if (!application || !activeApplications.some((item) => item.application === application)) {
      MessagePlugin.warning('请选择一个启用的应用');
      return;
    }

    setSubmitting(true);
    setStoredApplication(application);
    const requestedRedirect = getSafeRedirectPath(new URLSearchParams(location.search).get('redirect'));
    navigate(getPostApplicationSelectionPath(requestedRedirect), { replace: true });
  };

  const handleReload = () => {
    void reloadApplications();
  };

  return (
    <div className="permission-page permission-select-application-page">
      <div className="permission-select-application-heading">
        <div>
          <Tag theme="primary" variant="light-outline">权限上下文</Tag>
          <h1>选择应用</h1>
          <p>请选择要管理的应用。角色、菜单、按钮和用户绑定都会按应用隔离。</p>
        </div>
      </div>

      <Card className="permission-card permission-select-application-card" bordered>
        <div className="permission-card-title">
          <strong>可用应用</strong>
          <span>{activeApplications.length} 个启用应用</span>
        </div>

        {applicationsLoading ? (
          <div className="permission-select-application-loading">
            <Loading size="small" text="正在加载应用列表" />
          </div>
        ) : applicationsError ? (
          <div className="permission-load-error permission-select-application-error">
            <span>应用列表加载失败：{getRequestErrorMessage(applicationsError, '请稍后重试')}</span>
            <Button variant="text" theme="primary" type="button" onClick={handleReload}>重试</Button>
          </div>
        ) : activeApplications.length === 0 ? (
          <div className="permission-select-application-empty">
            <strong>暂无可用应用</strong>
            <span>请联系管理员初始化应用后，再进入权限管理页面。</span>
          </div>
        ) : (
          <>
            <Radio.Group
              value={selectedApplication}
              onChange={(value) => setSelectedApplication(String(value))}
            >
              <div className="permission-application-option-grid">
                {activeApplications.map((item) => (
                  <label
                    className={`permission-application-option${selectedApplication === item.application ? ' is-selected' : ''}`}
                    key={item.application}
                  >
                    <Radio value={item.application} />
                    <span className="permission-application-option-copy">
                      <strong title={item.name || item.application}>{item.name || item.application}</strong>
                      <code title={item.application}>{item.application}</code>
                      {item.description ? <span title={item.description}>{item.description}</span> : null}
                    </span>
                  </label>
                ))}
              </div>
            </Radio.Group>
            <div className="permission-select-application-actions">
              <Space>
                <Button variant="outline" type="button" onClick={handleReload}>刷新列表</Button>
                <Button
                  theme="primary"
                  type="button"
                  loading={submitting}
                  disabled={!selectedApplication}
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

export default SelectApplicationPage;
