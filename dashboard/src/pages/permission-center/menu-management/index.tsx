import { useCallback, useEffect, useMemo, useState, type ReactElement } from 'react';
import { Button, Card, Dialog, Form, Input, MessagePlugin, Select, Space, Tag } from 'tdesign-react';

import { createMenu, getMenuTree, type CreateMenuRequest, type Menu, type MenuType } from '../../../api/permission';
import { getRequestErrorMessage, PageHeader, useServiceResourceScope } from '../shared';
import '../style.less';

type MenuForm = {
  type: MenuType;
  parentId: string;
  code: string;
  name: string;
  description: string;
  path: string;
  component: string;
  apiPath: string;
  httpMethod: string;
  icon: string;
  sort: string;
};

const EMPTY_MENU_FORM: MenuForm = {
  type: 'menu',
  parentId: '',
  code: '',
  name: '',
  description: '',
  path: '',
  component: '',
  apiPath: '',
  httpMethod: 'GET',
  icon: '',
  sort: '0',
};

const menuTypeOptions = [
  { label: '菜单', value: 'menu' },
  { label: '按钮', value: 'button' },
];

const httpMethodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((method) => ({ label: method, value: method }));

const renderMenuNode = (node: Menu): ReactElement => (
  <div key={node.id} className="permission-tree-node">
    <div className="permission-tree-row">
      <div className="permission-tree-label">
        <Tag theme={node.type === 'menu' ? 'primary' : 'warning'} variant="light-outline">
          {node.type === 'menu' ? '菜单' : '按钮'}
        </Tag>
        <strong title={node.name}>{node.name || '-'}</strong>
      </div>
      <div className="permission-tree-meta permission-table-code" title={node.code}>{node.code || '-'}</div>
      <div className="permission-tree-meta" title={node.type === 'button' ? node.apiPath : node.path}>
        {node.type === 'button' ? `${node.httpMethod || 'ANY'} ${node.apiPath || '-'}` : node.path || '-'}
      </div>
      <div className="permission-tree-meta" title={node.type === 'menu' ? node.component : node.description}>
        {node.type === 'menu' ? node.component || '-' : node.description || '-'}
      </div>
    </div>
    {node.children.length > 0 ? <div className="permission-tree-children">{node.children.map(renderMenuNode)}</div> : null}
  </div>
);

const flattenMenus = (nodes: Menu[]): Menu[] => nodes.flatMap((node) => [node, ...flattenMenus(node.children)]);

const MenuManagementPage = () => {
  const { serviceResource } = useServiceResourceScope();
  const [tree, setTree] = useState<Menu[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogVisible, setDialogVisible] = useState(false);
  const [menuForm, setMenuForm] = useState<MenuForm>(EMPTY_MENU_FORM);
  const [creating, setCreating] = useState(false);

  const loadTree = useCallback(async (scope: string) => {
    setLoading(true);
    try {
      setTree(await getMenuTree(scope));
    } catch (error) {
      setTree([]);
      MessagePlugin.error(getRequestErrorMessage(error, '加载菜单树失败'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadTree(serviceResource);
  }, [serviceResource, loadTree]);

  const menuParents = useMemo(
    () => flattenMenus(tree)
      .filter((menu) => menu.type === 'menu')
      .map((menu) => ({ label: menu.name || menu.code, value: menu.id })),
    [tree],
  );

  const openCreateDialog = (type: MenuType) => {
    setMenuForm({ ...EMPTY_MENU_FORM, type });
    setDialogVisible(true);
  };

  const closeCreateDialog = () => {
    if (creating) {
      return;
    }
    setDialogVisible(false);
    setMenuForm({ ...EMPTY_MENU_FORM });
  };

  const submitCreateMenu = async () => {
    const code = menuForm.code.trim();
    const name = menuForm.name.trim();
    const apiPath = menuForm.apiPath.trim();
    if (!code || !name) {
      MessagePlugin.warning('请填写编码和名称');
      return;
    }
    if (menuForm.type === 'button' && !menuForm.parentId) {
      MessagePlugin.warning('按钮必须选择父级菜单');
      return;
    }
    if (menuForm.type === 'button' && !apiPath) {
      MessagePlugin.warning('按钮必须填写 API 路径');
      return;
    }

    const parsedSort = Number(menuForm.sort);
    const payload: CreateMenuRequest = {
      service_resource: serviceResource,
      parent_id: menuForm.parentId || null,
      code,
      name,
      description: menuForm.description.trim(),
      type: menuForm.type,
      path: menuForm.type === 'menu' ? menuForm.path.trim() : '',
      component: menuForm.type === 'menu' ? menuForm.component.trim() : '',
      api_path: menuForm.type === 'button' ? apiPath : '',
      http_method: menuForm.type === 'button' ? menuForm.httpMethod : '',
      icon: menuForm.type === 'menu' ? menuForm.icon.trim() : '',
      sort: Number.isFinite(parsedSort) ? parsedSort : 0,
    };

    setCreating(true);
    try {
      await createMenu(payload);
      setDialogVisible(false);
      setMenuForm({ ...EMPTY_MENU_FORM });
      MessagePlugin.success(`${menuForm.type === 'menu' ? '菜单' : '按钮'}已创建`);
      await loadTree(serviceResource);
    } catch (error) {
      MessagePlugin.error(getRequestErrorMessage(error, '创建权限节点失败'));
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="permission-page permission-menu-page">
      <PageHeader
        title="菜单与按钮管理"
        description="维护服务资源导航树和操作按钮，菜单与按钮使用类型严格区分"
        actions={(
          <Space>
            <Button variant="outline" theme="primary" type="button" onClick={() => openCreateDialog('menu')}>新建菜单</Button>
            <Button theme="primary" type="button" onClick={() => openCreateDialog('button')}>新建按钮</Button>
          </Space>
        )}
      />

      <div className="permission-toolbar">
        <span className="permission-toolbar-meta">{flattenMenus(tree).length} 个权限节点</span>
      </div>

      <Card className="permission-card" bordered>
        <div className="permission-tree">
          <div className="permission-tree-row permission-tree-header">
            <span>名称</span>
            <span>编码</span>
            <span>路由 / API</span>
            <span>组件 / 描述</span>
          </div>
          {loading ? (
            <div className="permission-empty">正在加载菜单树...</div>
          ) : tree.length === 0 ? (
          <div className="permission-empty">当前服务资源暂无菜单或按钮</div>
          ) : tree.map(renderMenuNode)}
        </div>
      </Card>

      <Dialog
        visible={dialogVisible}
        header={`新建${menuForm.type === 'menu' ? '菜单' : '按钮'}`}
        width={720}
        confirmBtn={{ content: '创建', theme: 'primary', loading: creating }}
        cancelBtn="取消"
        confirmLoading={creating}
        onConfirm={() => void submitCreateMenu()}
        onCancel={closeCreateDialog}
        onClose={closeCreateDialog}
        destroyOnClose
      >
        <Form className="permission-inline-form" labelAlign="top">
          <Form.FormItem label="类型">
            <Select
              value={menuForm.type}
              options={menuTypeOptions}
              onChange={(value) => setMenuForm((prev) => ({ ...prev, type: value === 'button' ? 'button' : 'menu' }))}
            />
          </Form.FormItem>
          <Form.FormItem label="父级菜单" help="菜单可作为根节点或挂在已有菜单下，按钮必须挂在菜单下。">
            <Select
              value={menuForm.parentId}
              options={[{ label: '根节点', value: '' }, ...menuParents]}
              placeholder="请选择父级菜单"
              filterable
              onChange={(value) => setMenuForm((prev) => ({ ...prev, parentId: String(value || '') }))}
            />
          </Form.FormItem>
          <Form.FormItem label="编码">
            <Input value={menuForm.code} maxlength={100} placeholder="例如 article-management" onChange={(value) => setMenuForm((prev) => ({ ...prev, code: value }))} />
          </Form.FormItem>
          <Form.FormItem label="名称">
            <Input value={menuForm.name} maxlength={100} placeholder="例如 文章管理" onChange={(value) => setMenuForm((prev) => ({ ...prev, name: value }))} />
          </Form.FormItem>
          <Form.FormItem label="排序">
            <Input type="number" value={menuForm.sort} placeholder="0" onChange={(value) => setMenuForm((prev) => ({ ...prev, sort: value }))} />
          </Form.FormItem>
          <Form.FormItem label="描述">
            <Input value={menuForm.description} maxlength={200} placeholder="可选" onChange={(value) => setMenuForm((prev) => ({ ...prev, description: value }))} />
          </Form.FormItem>

          {menuForm.type === 'menu' ? (
            <>
              <Form.FormItem label="路由路径">
                <Input value={menuForm.path} placeholder="例如 /articles" onChange={(value) => setMenuForm((prev) => ({ ...prev, path: value }))} />
              </Form.FormItem>
              <Form.FormItem label="组件路径">
                <Input value={menuForm.component} placeholder="例如 /pages/articles" onChange={(value) => setMenuForm((prev) => ({ ...prev, component: value }))} />
              </Form.FormItem>
              <Form.FormItem label="图标标识">
                <Input value={menuForm.icon} placeholder="可选" onChange={(value) => setMenuForm((prev) => ({ ...prev, icon: value }))} />
              </Form.FormItem>
            </>
          ) : (
            <>
              <Form.FormItem label="API 路径" className="full-width" help="按钮权限在后端鉴权时可映射到该接口路径。">
                <Input value={menuForm.apiPath} placeholder="例如 /v1/articles" onChange={(value) => setMenuForm((prev) => ({ ...prev, apiPath: value }))} />
              </Form.FormItem>
              <Form.FormItem label="HTTP 方法">
                <Select value={menuForm.httpMethod} options={httpMethodOptions} onChange={(value) => setMenuForm((prev) => ({ ...prev, httpMethod: String(value) }))} />
              </Form.FormItem>
            </>
          )}
        </Form>
      </Dialog>
    </div>
  );
};

export default MenuManagementPage;
