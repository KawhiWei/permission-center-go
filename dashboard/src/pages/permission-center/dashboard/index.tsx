import { useCallback, useEffect, useState } from 'react';
import { Button, Card, MessagePlugin, Space, Tag } from 'tdesign-react';

import { getMenuTree, listRoles, type Menu, type Role } from '../../../api/permission';
import { ApplicationField, getRequestErrorMessage, PageHeader, useApplicationScope } from '../shared';
import '../style.less';

type MenuCounts = {
  menus: number;
  buttons: number;
};

const countMenuTree = (nodes: Menu[]): MenuCounts => {
  return nodes.reduce<MenuCounts>((total, node) => {
    const own = node.type === 'button'
      ? { menus: 0, buttons: 1 }
      : { menus: 1, buttons: 0 };
    const children = countMenuTree(node.children);
    return {
      menus: total.menus + own.menus + children.menus,
      buttons: total.buttons + own.buttons + children.buttons,
    };
  }, { menus: 0, buttons: 0 });
};

const DashboardPage = () => {
  const {
    application,
    draftApplication,
    setDraftApplication,
    applyApplication,
    applications,
    applicationsLoading,
    applicationsError,
    reloadApplications,
  } = useApplicationScope();
  const [roles, setRoles] = useState<Role[]>([]);
  const [menuCounts, setMenuCounts] = useState<MenuCounts>({ menus: 0, buttons: 0 });
  const [loading, setLoading] = useState(false);
  const [loaded, setLoaded] = useState(false);

  const loadSummary = useCallback(async (scope: string) => {
    setLoading(true);
    setLoaded(false);
    try {
      const [nextRoles, nextTree] = await Promise.all([listRoles(scope), getMenuTree(scope)]);
      setRoles(nextRoles);
      setMenuCounts(countMenuTree(nextTree));
      setLoaded(true);
    } catch (error) {
      setRoles([]);
      setMenuCounts({ menus: 0, buttons: 0 });
      MessagePlugin.error(getRequestErrorMessage(error, '加载权限中心摘要失败'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSummary(application);
  }, [application, loadSummary]);

  const enabledRoleCount = roles.filter((role) => role.enabled).length;
  const totalPermissionCount = menuCounts.menus + menuCounts.buttons;

  return (
    <div className="permission-page permission-dashboard-page">
      <PageHeader
        title="仪表盘"
        description="查看当前应用的角色、菜单和按钮权限规模"
        actions={<Tag theme="primary" variant="light-outline">RBAC</Tag>}
      />

      <div className="permission-toolbar">
        <ApplicationField
          draftApplication={draftApplication}
          onDraftChange={setDraftApplication}
          onApply={applyApplication}
          loading={loading}
          applications={applications}
          applicationsLoading={applicationsLoading}
          applicationsError={applicationsError}
          onRefreshApplications={() => void reloadApplications()}
        />
        <span className="permission-toolbar-meta">当前应用：{application}</span>
      </div>

      <div className="permission-metric-grid">
        <div className="permission-metric">
          <span className="permission-metric-label">应用标识</span>
          <strong className="permission-metric-value" title={application}>{application}</strong>
          <span className="permission-metric-hint">权限数据按应用隔离</span>
        </div>
        <div className="permission-metric">
          <span className="permission-metric-label">启用角色</span>
          <strong className="permission-metric-value">{loading ? '...' : enabledRoleCount}</strong>
          <span className="permission-metric-hint">{roles.length} 个角色记录</span>
        </div>
        <div className="permission-metric">
          <span className="permission-metric-label">菜单节点</span>
          <strong className="permission-metric-value">{loading ? '...' : menuCounts.menus}</strong>
          <span className="permission-metric-hint">用于构建导航树</span>
        </div>
        <div className="permission-metric">
          <span className="permission-metric-label">按钮权限</span>
          <strong className="permission-metric-value">{loading ? '...' : menuCounts.buttons}</strong>
          <span className="permission-metric-hint">用于控制操作入口</span>
        </div>
      </div>

      <Card className="permission-card permission-dashboard-status" bordered>
        <div className="permission-card-title">
          <strong>权限概览</strong>
          <span>{loaded ? '数据已更新' : loading ? '正在加载' : '等待加载'}</span>
        </div>
        <div className="permission-dashboard-status-grid">
          <div>
            <span>可用角色</span>
            <strong>{enabledRoleCount}</strong>
          </div>
          <div>
            <span>权限节点总数</span>
            <strong>{loading ? '...' : totalPermissionCount}</strong>
          </div>
          <div>
            <span>数据来源</span>
            <strong>Permission Center API</strong>
          </div>
        </div>
      </Card>

      <Card className="permission-card" bordered>
        <div className="permission-card-title">
          <strong>操作流程</strong>
          <span>按以下顺序完成应用授权</span>
        </div>
        <div className="permission-dashboard-steps">
          <div className="permission-dashboard-step">
            <span>01</span>
            <div>
              <strong>建立角色</strong>
              <p>定义角色编码、名称与职责范围。</p>
            </div>
          </div>
          <div className="permission-dashboard-step">
            <span>02</span>
            <div>
              <strong>维护菜单与按钮</strong>
              <p>搭建导航树，并为按钮记录 API 路径。</p>
            </div>
          </div>
          <div className="permission-dashboard-step">
            <span>03</span>
            <div>
              <strong>授予角色权限</strong>
              <p>将菜单和按钮节点授权给角色。</p>
            </div>
          </div>
          <div className="permission-dashboard-step">
            <span>04</span>
            <div>
              <strong>绑定用户角色</strong>
              <p>使用统一登录用户的 subject 完成角色绑定。</p>
            </div>
          </div>
        </div>
        <Space className="permission-dashboard-note">
          <Tag theme="warning" variant="light">提示</Tag>
          <span>统一登录只负责身份认证，角色和权限数据由本权限中心维护。</span>
          <Button variant="text" theme="primary" type="button" onClick={() => void loadSummary(application)}>刷新摘要</Button>
        </Space>
      </Card>
    </div>
  );
};

export default DashboardPage;
