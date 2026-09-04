import { useCallback, useEffect, useState } from 'react';
import { Button, Card, Dialog, Drawer, Form, Input, MessagePlugin, Space, Tag, Checkbox } from 'tdesign-react';

import {
  createRole,
  getMenuTree,
  getRoleMenuIDs,
  listRoles,
  replaceRoleMenus,
  type Menu,
  type Role,
} from '../../../api/permission';
import { formatDateTime, getRequestErrorMessage, PageHeader, useApplicationScope } from '../shared';
import '../style.less';

type RoleForm = {
  code: string;
  name: string;
  description: string;
};

type FlatMenu = {
  menu: Menu;
  depth: number;
};

const EMPTY_ROLE_FORM: RoleForm = { code: '', name: '', description: '' };

const flattenMenus = (nodes: Menu[], depth = 0): FlatMenu[] => nodes.flatMap((menu) => [
  { menu, depth },
  ...flattenMenus(menu.children, depth + 1),
]);

const RoleManagementPage = () => {
  const { application } = useApplicationScope();
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogVisible, setDialogVisible] = useState(false);
  const [roleForm, setRoleForm] = useState<RoleForm>(EMPTY_ROLE_FORM);
  const [creating, setCreating] = useState(false);
  const [grantVisible, setGrantVisible] = useState(false);
  const [grantRole, setGrantRole] = useState<Role | null>(null);
  const [grantMenus, setGrantMenus] = useState<FlatMenu[]>([]);
  const [grantMenuIDs, setGrantMenuIDs] = useState<string[]>([]);
  const [grantLoading, setGrantLoading] = useState(false);
  const [savingGrant, setSavingGrant] = useState(false);

  const loadRoles = useCallback(async (scope: string) => {
    setLoading(true);
    try {
      setRoles(await listRoles(scope));
    } catch (error) {
      setRoles([]);
      MessagePlugin.error(getRequestErrorMessage(error, '加载角色失败'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadRoles(application);
  }, [application, loadRoles]);

  const openCreateDialog = () => {
    setRoleForm({ ...EMPTY_ROLE_FORM });
    setDialogVisible(true);
  };

  const closeCreateDialog = () => {
    if (creating) {
      return;
    }
    setDialogVisible(false);
    setRoleForm({ ...EMPTY_ROLE_FORM });
  };

  const submitCreateRole = async () => {
    const code = roleForm.code.trim();
    const name = roleForm.name.trim();
    if (!code || !name) {
      MessagePlugin.warning('请填写角色编码和角色名称');
      return;
    }

    setCreating(true);
    try {
      await createRole({ application, code, name, description: roleForm.description.trim() });
      setDialogVisible(false);
      setRoleForm({ ...EMPTY_ROLE_FORM });
      MessagePlugin.success('角色已创建');
      await loadRoles(application);
    } catch (error) {
      MessagePlugin.error(getRequestErrorMessage(error, '创建角色失败'));
    } finally {
      setCreating(false);
    }
  };

  const openGrantDrawer = async (role: Role) => {
    setGrantRole(role);
    setGrantVisible(true);
    setGrantLoading(true);
    try {
      const [tree, selected] = await Promise.all([getMenuTree(role.application), getRoleMenuIDs(role.id)]);
      setGrantMenus(flattenMenus(tree));
      setGrantMenuIDs(selected);
    } catch (error) {
      setGrantMenus([]);
      setGrantMenuIDs([]);
      MessagePlugin.error(getRequestErrorMessage(error, '加载角色授权信息失败'));
    } finally {
      setGrantLoading(false);
    }
  };

  const closeGrantDrawer = () => {
    if (savingGrant) {
      return;
    }
    setGrantVisible(false);
    setGrantRole(null);
    setGrantMenus([]);
    setGrantMenuIDs([]);
  };

  const saveGrant = async () => {
    if (!grantRole) {
      return;
    }
    setSavingGrant(true);
    try {
      await replaceRoleMenus(grantRole.id, grantMenuIDs);
      MessagePlugin.success('角色菜单权限已保存');
      setGrantVisible(false);
      setGrantRole(null);
      setGrantMenus([]);
      setGrantMenuIDs([]);
    } catch (error) {
      MessagePlugin.error(getRequestErrorMessage(error, '保存角色授权失败'));
    } finally {
      setSavingGrant(false);
    }
  };

  return (
    <div className="permission-page permission-role-page">
      <PageHeader
        title="角色管理"
        description="在应用范围内建立角色，并为角色分配菜单与按钮权限"
        actions={<Button theme="primary" type="button" onClick={openCreateDialog}>新建角色</Button>}
      />

      <div className="permission-toolbar">
        <span className="permission-toolbar-meta">共 {roles.length} 个启用角色</span>
      </div>

      <Card className="permission-card" bordered>
        <div className="permission-table-wrap">
          <table className="permission-table">
            <thead>
              <tr>
                <th>角色名称</th>
                <th>角色编码</th>
                <th>描述</th>
                <th>创建时间</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr><td colSpan={6} className="permission-empty">正在加载角色...</td></tr>
              ) : roles.length === 0 ? (
                <tr><td colSpan={6} className="permission-empty">当前应用暂无角色</td></tr>
              ) : roles.map((role) => (
                <tr key={role.id}>
                  <td>
                    <div className="permission-table-name">{role.name || '-'}</div>
                    <div className="permission-table-code">{role.id || '-'}</div>
                  </td>
                  <td className="permission-table-code">{role.code || '-'}</td>
                  <td className="permission-table-muted">{role.description || '-'}</td>
                  <td className="permission-table-muted">{formatDateTime(role.createdAt)}</td>
                  <td><Tag theme={role.enabled ? 'success' : 'default'} variant="light-outline">{role.enabled ? '启用' : '停用'}</Tag></td>
                  <td>
                    <Button variant="text" theme="primary" type="button" onClick={() => void openGrantDrawer(role)}>菜单授权</Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <Dialog
        visible={dialogVisible}
        header="新建角色"
        width={520}
        confirmBtn={{ content: '创建', theme: 'primary', loading: creating }}
        cancelBtn="取消"
        confirmLoading={creating}
        onConfirm={() => void submitCreateRole()}
        onCancel={closeCreateDialog}
        onClose={closeCreateDialog}
        destroyOnClose
      >
        <Form className="permission-drawer-form" labelAlign="top">
          <Form.FormItem label="角色编码" help="建议使用稳定、便于接口传递的英文编码。">
            <Input
              value={roleForm.code}
              placeholder="例如 content-editor"
              maxlength={100}
              onChange={(value) => setRoleForm((prev) => ({ ...prev, code: value }))}
            />
          </Form.FormItem>
          <Form.FormItem label="角色名称">
            <Input
              value={roleForm.name}
              placeholder="例如 内容编辑"
              maxlength={100}
              onChange={(value) => setRoleForm((prev) => ({ ...prev, name: value }))}
            />
          </Form.FormItem>
          <Form.FormItem label="描述">
            <Input
              value={roleForm.description}
              placeholder="可选"
              maxlength={200}
              onChange={(value) => setRoleForm((prev) => ({ ...prev, description: value }))}
            />
          </Form.FormItem>
        </Form>
      </Dialog>

      <Drawer
        visible={grantVisible}
        header={grantRole ? `菜单授权 · ${grantRole.name}` : '菜单授权'}
        size="520px"
        confirmBtn={{ content: '保存授权', theme: 'primary', loading: savingGrant || grantLoading }}
        cancelBtn="取消"
        onConfirm={() => void saveGrant()}
        onCancel={closeGrantDrawer}
        onClose={closeGrantDrawer}
        destroyOnClose
      >
        <Space direction="vertical" size={10} style={{ width: '100%' }}>
          <div className="permission-form-help">勾选角色可见的菜单和按钮，保存后会整体替换该角色在当前应用内的授权。</div>
          {grantLoading ? (
            <div className="permission-empty">正在加载菜单权限...</div>
          ) : grantMenus.length === 0 ? (
            <div className="permission-empty">当前应用暂无可授权的菜单或按钮</div>
          ) : (
            <Checkbox.Group
              value={grantMenuIDs}
              onChange={(value) => setGrantMenuIDs(value.map(String))}
            >
              <div className="permission-grant-list">
                {grantMenus.map(({ menu, depth }) => (
                  <div key={menu.id} className="permission-grant-item" style={{ paddingLeft: 8 + depth * 22 }}>
                    <Checkbox value={menu.id} />
                    <div className="permission-grant-label">
                      <Tag theme={menu.type === 'menu' ? 'primary' : 'warning'} variant="light-outline">
                        {menu.type === 'menu' ? '菜单' : '按钮'}
                      </Tag>
                      <strong title={menu.name}>{menu.name}</strong>
                      <span className="permission-grant-code" title={menu.code}>{menu.code}</span>
                    </div>
                  </div>
                ))}
              </div>
            </Checkbox.Group>
          )}
        </Space>
      </Drawer>
    </div>
  );
};

export default RoleManagementPage;
