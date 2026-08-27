import { useCallback, useEffect, useMemo, useState } from 'react';
import { AddIcon, RefreshIcon, TreeRoundDotVerticalIcon } from 'tdesign-icons-react';
import {
  Button,
  Checkbox,
  Dialog,
  Drawer,
  Form,
  Input,
  InputNumber,
  Radio,
  Space,
  Table,
  Tag,
  Textarea,
  type TableProps,
} from 'tdesign-react';
import './permission-console.less';
import Login from './pages/login';
import { getConfig, getCurrentUser, logout } from './api/login';

type ResourceType = 'menu' | 'button';
type Workspace = 'permissions' | 'user-roles';

type BaseFields = {
  created_by_id: string;
  created_by_name: string;
  created_at: string;
  updated_by_id: string;
  updated_by_name: string;
  updated_at: string;
  is_deleted: boolean;
};

type Resource = BaseFields & {
  id: string;
  application: string;
  parent_id: string | null;
  code: string;
  name: string;
  description?: string;
  type: ResourceType;
  path?: string;
  component?: string;
  api_path?: string;
  http_method?: string;
  icon?: string;
  sort: number;
  enabled: boolean;
  children?: Resource[];
};

type Role = BaseFields & {
  id: string;
  application: string;
  code: string;
  name: string;
  description?: string;
  enabled: boolean;
};

type ResourceForm = {
  code: string;
  name: string;
  description: string;
  type: ResourceType;
  parent_id: string;
  path: string;
  component: string;
  api_path: string;
  http_method: string;
  icon: string;
  sort: number;
};

type JsonObject = Record<string, unknown>;

const createEmptyResource = (): ResourceForm => ({
  code: '',
  name: '',
  description: '',
  type: 'menu',
  parent_id: '',
  path: '',
  component: '',
  api_path: '',
  http_method: 'GET',
  icon: '',
  sort: 0,
});

const request = async <T,>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(`/api${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  });
  const payload = response.status === 204 ? undefined : await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = payload && typeof payload === 'object' ? payload as { error?: string; message?: string } : {};
    throw new Error(error.error || error.message || `请求失败 (${response.status})`);
  }
  return payload as T;
};

const objectOf = (value: unknown): JsonObject => (
  value && typeof value === 'object' ? value as JsonObject : {}
);

const valueOf = (object: JsonObject, key: string, upperKey: string) => object[key] ?? object[upperKey];

const textOf = (value: unknown, fallback = '') => (
  typeof value === 'string' ? value : value == null ? fallback : String(value)
);

const booleanOf = (value: unknown, fallback = true) => typeof value === 'boolean' ? value : fallback;

const numberOf = (value: unknown, fallback = 0) => typeof value === 'number' && Number.isFinite(value) ? value : fallback;

const arrayFromEnvelope = (value: unknown, names: string[] = ['items', 'Items']): unknown[] => {
  if (Array.isArray(value)) {
    return value;
  }
  const object = objectOf(value);
  for (const name of names) {
    if (Array.isArray(object[name])) {
      return object[name] as unknown[];
    }
  }
  return [];
};

const normalizeRole = (value: unknown): Role => {
  const object = objectOf(value);
  return {
    id: textOf(valueOf(object, 'id', 'ID')),
    application: textOf(valueOf(object, 'application', 'Application')),
    code: textOf(valueOf(object, 'code', 'Code')),
    name: textOf(valueOf(object, 'name', 'Name')),
    description: textOf(valueOf(object, 'description', 'Description')),
    enabled: booleanOf(valueOf(object, 'enabled', 'Enabled')),
    created_by_id: textOf(valueOf(object, 'created_by_id', 'CreatedByID')),
    created_by_name: textOf(valueOf(object, 'created_by_name', 'CreatedByName')),
    created_at: textOf(valueOf(object, 'created_at', 'CreatedAt')),
    updated_by_id: textOf(valueOf(object, 'updated_by_id', 'UpdatedByID')),
    updated_by_name: textOf(valueOf(object, 'updated_by_name', 'UpdatedByName')),
    updated_at: textOf(valueOf(object, 'updated_at', 'UpdatedAt')),
    is_deleted: booleanOf(valueOf(object, 'is_deleted', 'IsDeleted'), false),
  };
};

const normalizeResource = (value: unknown): Resource => {
  const object = objectOf(value);
  const children = valueOf(object, 'children', 'Children');
  const type = textOf(valueOf(object, 'type', 'Type'));
  const parentID = valueOf(object, 'parent_id', 'ParentID') ?? valueOf(object, 'parentId', 'ParentId');
  return {
    id: textOf(valueOf(object, 'id', 'ID')),
    application: textOf(valueOf(object, 'application', 'Application')),
    parent_id: parentID == null || parentID === '' ? null : textOf(parentID),
    code: textOf(valueOf(object, 'code', 'Code')),
    name: textOf(valueOf(object, 'name', 'Name')),
    description: textOf(valueOf(object, 'description', 'Description')),
    type: type === 'button' ? 'button' : 'menu',
    path: textOf(valueOf(object, 'path', 'Path')),
    component: textOf(valueOf(object, 'component', 'Component')),
    api_path: textOf(valueOf(object, 'api_path', 'APIPath')),
    http_method: textOf(valueOf(object, 'http_method', 'HTTPMethod')),
    icon: textOf(valueOf(object, 'icon', 'Icon')),
    sort: numberOf(valueOf(object, 'sort', 'Sort')),
    enabled: booleanOf(valueOf(object, 'enabled', 'Enabled')),
    created_by_id: textOf(valueOf(object, 'created_by_id', 'CreatedByID')),
    created_by_name: textOf(valueOf(object, 'created_by_name', 'CreatedByName')),
    created_at: textOf(valueOf(object, 'created_at', 'CreatedAt')),
    updated_by_id: textOf(valueOf(object, 'updated_by_id', 'UpdatedByID')),
    updated_by_name: textOf(valueOf(object, 'updated_by_name', 'UpdatedByName')),
    updated_at: textOf(valueOf(object, 'updated_at', 'UpdatedAt')),
    is_deleted: booleanOf(valueOf(object, 'is_deleted', 'IsDeleted'), false),
    children: arrayFromEnvelope(children, ['children', 'Children']).map(normalizeResource),
  };
};

const flatten = (nodes: Resource[], depth = 0): Array<Resource & { depth: number }> => (
  nodes.flatMap((node) => [
    { ...node, depth },
    ...flatten(node.children || [], depth + 1),
  ])
);

const readRoleIDs = (value: unknown): string[] => {
  const object = objectOf(value);
  const direct = valueOf(object, 'role_ids', 'RoleIDs');
  const directIDs = Array.isArray(direct) ? direct : null;
  if (directIDs) {
    return directIDs.map((item) => {
      if (item && typeof item === 'object') {
        const role = objectOf(item);
        return textOf(valueOf(role, 'id', 'ID'));
      }
      return textOf(item);
    }).filter(Boolean);
  }

  const roleItems = arrayFromEnvelope(value, ['roles', 'Roles', 'items', 'Items']);
  return roleItems.map((item) => {
    if (item && typeof item === 'object') {
      const role = objectOf(item);
      return textOf(valueOf(role, 'id', 'ID'));
    }
    return textOf(item);
  }).filter(Boolean);
};

const App = () => {
  const [authReady, setAuthReady] = useState(false);
  const [authenticated, setAuthenticated] = useState(false);
  const [oidcEnabled, setOIDCEnabled] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [workspace, setWorkspace] = useState<Workspace>('permissions');
  const [application, setApplication] = useState('admin-console');
  const [activeApp, setActiveApp] = useState('admin-console');
  const [roles, setRoles] = useState<Role[]>([]);
  const [tree, setTree] = useState<Resource[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [roleVisible, setRoleVisible] = useState(false);
  const [resourceVisible, setResourceVisible] = useState(false);
  const [grantRole, setGrantRole] = useState<Role>();
  const [roleForm, setRoleForm] = useState({ code: '', name: '', description: '' });
  const [resourceForm, setResourceForm] = useState<ResourceForm>(createEmptyResource);
  const [grants, setGrants] = useState<string[]>([]);
  const [notice, setNotice] = useState('');
  const [subject, setSubject] = useState('');
  const [loadedSubject, setLoadedSubject] = useState('');
  const [userRoleIDs, setUserRoleIDs] = useState<string[]>([]);
  const [userRoleLoading, setUserRoleLoading] = useState(false);
  const [userRoleSaving, setUserRoleSaving] = useState(false);

  const resources = useMemo(() => flatten(tree), [tree]);
  const menus = useMemo(() => resources.filter((item) => item.type === 'menu'), [resources]);
  const selectedRoles = useMemo(
    () => roles.filter((role) => userRoleIDs.includes(role.id)),
    [roles, userRoleIDs],
  );

  const load = useCallback(async (app: string): Promise<boolean> => {
    const normalizedApp = app.trim();
    if (!normalizedApp) {
      setNotice('请输入应用标识');
      return false;
    }
    try {
      setLoading(true);
      setNotice('');
      const [roleResponse, treeResponse] = await Promise.all([
        request<unknown>(`/v1/roles?application=${encodeURIComponent(normalizedApp)}`),
        request<unknown>(`/v1/resources/tree?application=${encodeURIComponent(normalizedApp)}`),
      ]);
      setRoles(arrayFromEnvelope(roleResponse).map(normalizeRole));
      setTree(arrayFromEnvelope(treeResponse).map(normalizeResource));
      setApplication(normalizedApp);
      setActiveApp(normalizedApp);
      setGrantRole(undefined);
      setGrants([]);
      return true;
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '加载应用权限失败');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    Promise.all([getCurrentUser(), getConfig()])
      .then(([user, oidc]) => {
        setAuthenticated(user.isAuthenticated);
        setOIDCEnabled(oidc.enabled);
      })
      .catch(() => setAuthenticated(false))
      .finally(() => setAuthReady(true));
  }, []);

  useEffect(() => {
    if (authenticated) {
      void load('admin-console');
    }
  }, [authenticated, load]);

  const addRole = async () => {
    if (!roleForm.code.trim() || !roleForm.name.trim()) {
      setNotice('请填写角色编码和名称');
      return;
    }
    try {
      setSubmitting(true);
      await request('/v1/roles', {
        method: 'POST',
        body: JSON.stringify({
          application: activeApp,
          code: roleForm.code.trim(),
          name: roleForm.name.trim(),
          description: roleForm.description.trim(),
        }),
      });
      setNotice('角色已创建');
      setRoleVisible(false);
      setRoleForm({ code: '', name: '', description: '' });
      await load(activeApp);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '创建角色失败');
    } finally {
      setSubmitting(false);
    }
  };

  const addResource = async () => {
    if (!resourceForm.code.trim() || !resourceForm.name.trim()) {
      setNotice('请填写资源编码和名称');
      return;
    }
    if (resourceForm.type === 'button' && (!resourceForm.parent_id || !resourceForm.api_path.trim())) {
      setNotice('按钮必须选择父菜单并填写 API 路径');
      return;
    }
    try {
      setSubmitting(true);
      await request('/v1/resources', {
        method: 'POST',
        body: JSON.stringify({
          application: activeApp,
          parent_id: resourceForm.type === 'button' ? resourceForm.parent_id || null : null,
          code: resourceForm.code.trim(),
          name: resourceForm.name.trim(),
          description: resourceForm.description.trim(),
          type: resourceForm.type,
          path: resourceForm.type === 'menu' ? resourceForm.path.trim() : '',
          component: resourceForm.type === 'menu' ? resourceForm.component.trim() : '',
          api_path: resourceForm.type === 'button' ? resourceForm.api_path.trim() : '',
          http_method: resourceForm.type === 'button' ? resourceForm.http_method.trim().toUpperCase() : '',
          icon: resourceForm.icon.trim(),
          sort: resourceForm.sort,
        }),
      });
      setNotice('资源已创建');
      setResourceVisible(false);
      setResourceForm(createEmptyResource());
      await load(activeApp);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '创建资源失败');
    } finally {
      setSubmitting(false);
    }
  };

  const openGrant = async (role: Role) => {
    try {
      const response = await request<unknown>(`/v1/roles/${encodeURIComponent(role.id)}/resources`);
      setGrants(readRoleIDs(response));
      setGrantRole(role);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '加载授权失败');
    }
  };

  const saveGrant = async () => {
    if (!grantRole) {
      return;
    }
    try {
      setSubmitting(true);
      await request(`/v1/roles/${encodeURIComponent(grantRole.id)}/resources`, {
        method: 'PUT',
        body: JSON.stringify({ resource_ids: grants }),
      });
      setNotice('授权已保存');
      setGrantRole(undefined);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '保存授权失败');
    } finally {
      setSubmitting(false);
    }
  };

  const loadUserRoles = async () => {
    const normalizedSubject = subject.trim();
    const normalizedApp = application.trim();
    if (!normalizedApp) {
      setNotice('请输入应用标识');
      return;
    }
    if (!normalizedSubject) {
      setNotice('请输入 NexusAuth user_id / subject');
      return;
    }
    if (normalizedSubject.length > 80) {
      setNotice('user_id / subject 最长为 80 个字符');
      return;
    }

    try {
      setUserRoleLoading(true);
      setNotice('');
      const loaded = await load(normalizedApp);
      if (!loaded) {
        return;
      }
      const response = await request<unknown>(
        `/v1/users/${encodeURIComponent(normalizedSubject)}/roles?application=${encodeURIComponent(normalizedApp)}`,
      );
      setSubject(normalizedSubject);
      setLoadedSubject(normalizedSubject);
      setUserRoleIDs(readRoleIDs(response));
    } catch (error) {
      setLoadedSubject('');
      setUserRoleIDs([]);
      setNotice(error instanceof Error ? error.message : '加载用户角色失败');
    } finally {
      setUserRoleLoading(false);
    }
  };

  const saveUserRoles = async () => {
    const normalizedSubject = loadedSubject.trim();
    if (!normalizedSubject) {
      setNotice('请先查询用户角色');
      return;
    }
    try {
      setUserRoleSaving(true);
      await request(`/v1/users/${encodeURIComponent(normalizedSubject)}/roles?application=${encodeURIComponent(activeApp)}`, {
        method: 'PUT',
        body: JSON.stringify({ role_ids: userRoleIDs }),
      });
      setNotice('用户角色已保存');
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '保存用户角色失败');
    } finally {
      setUserRoleSaving(false);
    }
  };

  const signOut = async () => {
    try {
      setLoggingOut(true);
      const result = await logout();
      setAuthenticated(false);
      if (result.logoutUrl) {
        window.location.assign(result.logoutUrl);
      }
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '退出登录失败');
    } finally {
      setLoggingOut(false);
    }
  };

  const roleColumns: TableProps<Role>['columns'] = [
    {
      colKey: 'name',
      title: '角色名称',
      minWidth: 160,
      cell: ({ row }) => (
        <div>
          <strong>{row.name}</strong>
          <small>{row.description || '未填写说明'}</small>
        </div>
      ),
    },
    {
      colKey: 'code',
      title: '角色编码',
      minWidth: 160,
      cell: ({ row }) => <code>{row.code}</code>,
    },
    {
      colKey: 'enabled',
      title: '状态',
      width: 90,
      cell: ({ row }) => (
        <Tag theme={row.enabled ? 'success' : 'default'} variant="light">
          {row.enabled ? '启用' : '停用'}
        </Tag>
      ),
    },
    {
      colKey: 'created_by_name',
      title: '创建人',
      minWidth: 130,
      cell: ({ row }) => (
        <div>
          <strong>{row.created_by_name || 'system'}</strong>
          <small>{row.created_by_id || 'system'}</small>
        </div>
      ),
    },
    {
      colKey: 'created_at',
      title: '创建时间',
      minWidth: 170,
      cell: ({ row }) => row.created_at ? new Date(row.created_at).toLocaleString('zh-CN', { hour12: false }) : '-',
    },
    {
      colKey: 'updated_at',
      title: '最后修改',
      minWidth: 190,
      cell: ({ row }) => (
        <div>
          <strong>{row.updated_by_name || 'system'}</strong>
          <small>{row.updated_at ? new Date(row.updated_at).toLocaleString('zh-CN', { hour12: false }) : '-'}</small>
        </div>
      ),
    },
    {
      colKey: 'action',
      title: '操作',
      width: 110,
      cell: ({ row }) => (
        <Button theme="primary" variant="text" size="small" onClick={() => void openGrant(row)}>
          配置权限
        </Button>
      ),
    },
  ];

  const isUserRoles = workspace === 'user-roles';

  if (!authReady) {
    return null;
  }
  if (!authenticated) {
    return <Login />;
  }

  return (
    <main className="console">
      <header>
        <div style={{ display: 'flex', alignItems: 'center', gap: 24, minWidth: 0 }}>
          <div className="brand">
            <b>权限中心</b>
            <span>RBAC 管理工作台</span>
          </div>
          <nav aria-label="权限中心功能" style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
            <Button
              size="small"
              theme={isUserRoles ? 'default' : 'primary'}
              variant={isUserRoles ? 'outline' : 'base'}
              onClick={() => setWorkspace('permissions')}
            >
              角色与资源
            </Button>
            <Button
              size="small"
              theme={isUserRoles ? 'primary' : 'default'}
              variant={isUserRoles ? 'base' : 'outline'}
              onClick={() => setWorkspace('user-roles')}
            >
              用户角色
            </Button>
          </nav>
        </div>
        <div className="app-picker">
          <Input
            value={application}
            onChange={(value) => setApplication(value)}
            onEnter={() => void load(application)}
            placeholder="应用标识"
            maxlength={100}
            aria-label="应用标识"
          />
          <Button icon={<RefreshIcon />} loading={loading} onClick={() => void load(application)}>
            加载应用
          </Button>
          {oidcEnabled && (
            <Button variant="outline" loading={loggingOut} onClick={() => void signOut()}>
              退出登录
            </Button>
          )}
        </div>
      </header>

      {notice && (
        <div
          role="alert"
          style={{
            maxWidth: 1400,
            width: 'calc(100% - 60px)',
            margin: '12px auto 0',
            padding: '9px 12px',
            color: 'var(--app-text)',
            border: '1px solid var(--app-border)',
            borderRadius: 6,
            background: 'var(--app-surface)',
          }}
        >
          {notice}
          <Button variant="text" size="small" onClick={() => setNotice('')} style={{ float: 'right' }}>
            关闭
          </Button>
        </div>
      )}

      <section className="intro">
        <div>
          <span>{isUserRoles ? 'USER ROLES' : 'APPLICATION'}</span>
          <h1>{isUserRoles ? '用户角色' : '角色与资源权限'}</h1>
          <p>
            {isUserRoles
              ? '输入 NexusAuth user_id / subject，查询并维护该用户在当前应用下的角色。'
              : '应用标识仅用于隔离角色、菜单与按钮资源，不包含应用管理能力。'}
          </p>
        </div>
        <div className="stats">
          <div>
            <small>当前应用</small>
            <b>{activeApp}</b>
          </div>
          <div>
            <small>{isUserRoles ? '已选角色' : '角色'}</small>
            <b>{isUserRoles ? selectedRoles.length : roles.length}</b>
          </div>
          <div>
            <small>{isUserRoles ? '用户 subject' : '资源'}</small>
            <b>{isUserRoles ? loadedSubject || '未查询' : resources.length}</b>
          </div>
        </div>
      </section>

      {!isUserRoles ? (
        <section className="grid">
          <article>
            <div className="heading">
              <div>
                <span>ROLES</span>
                <h2>角色管理</h2>
              </div>
              <Button theme="primary" icon={<AddIcon />} onClick={() => setRoleVisible(true)}>
                新增角色
              </Button>
            </div>
            <Table rowKey="id" data={roles} columns={roleColumns} loading={loading} hover empty="当前应用还没有角色" />
          </article>
          <article>
            <div className="heading">
              <div>
                <span>RESOURCES</span>
                <h2>资源树</h2>
              </div>
              <Button
                icon={<AddIcon />}
                onClick={() => {
                  setResourceForm(createEmptyResource());
                  setResourceVisible(true);
                }}
              >
                新增资源
              </Button>
            </div>
            <div className="tree">
              {resources.length === 0 ? (
                <p>当前应用还没有菜单或按钮权限</p>
              ) : resources.map((item) => (
                <div className="tree-row" key={item.id} style={{ paddingLeft: 16 + item.depth * 22 }}>
                  <TreeRoundDotVerticalIcon />
                  <div>
                    <b>{item.name}</b>
                    <small>{item.code}</small>
                  </div>
                  <Tag theme={item.type === 'menu' ? 'primary' : 'default'} variant="light">
                    {item.type === 'menu' ? '菜单' : '按钮'}
                  </Tag>
                </div>
              ))}
            </div>
          </article>
        </section>
      ) : (
        <section className="grid" style={{ display: 'block' }}>
          <article>
            <div className="heading">
              <div>
                <span>USER ASSIGNMENT</span>
                <h2>用户角色配置</h2>
              </div>
              <Tag theme={loadedSubject ? 'success' : 'default'} variant="light">
                {loadedSubject ? '已加载' : '待查询'}
              </Tag>
            </div>
            <div style={{ padding: '16px 18px 4px' }}>
              <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'flex-end', gap: 12 }}>
                <div style={{ flex: '1 1 320px', maxWidth: 520 }}>
                  <label style={{ display: 'block', marginBottom: 6, color: 'var(--app-text-secondary)', fontSize: 12 }}>
                    NexusAuth user_id / subject
                  </label>
                  <Input
                    value={subject}
                    maxlength={80}
                    onChange={(value) => setSubject(value)}
                    onEnter={() => void loadUserRoles()}
                    placeholder="例如：user_01 或 sub_xxx"
                    aria-label="NexusAuth user_id / subject"
                  />
                </div>
                <Button theme="primary" loading={userRoleLoading} onClick={() => void loadUserRoles()}>
                  查询角色
                </Button>
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, margin: '16px 0 8px', color: 'var(--app-text-muted)', fontSize: 12 }}>
                <span>1. 输入用户 subject</span>
                <span>2. 查询当前应用角色</span>
                <span>3. 勾选角色并保存</span>
              </div>
            </div>
            <div style={{ padding: '8px 18px 20px' }}>
              {!loadedSubject ? (
                <p style={{ margin: '28px 0', color: 'var(--app-text-muted)', textAlign: 'center' }}>
                  输入 subject 后查询，开始配置该用户的角色
                </p>
              ) : roles.length === 0 ? (
                <p style={{ margin: '28px 0', color: 'var(--app-text-muted)', textAlign: 'center' }}>
                  当前应用暂无可分配角色
                </p>
              ) : (
                <>
                  <Checkbox.Group value={userRoleIDs} onChange={(value) => setUserRoleIDs(value.map(String))}>
                    {roles.map((role) => (
                      <Checkbox key={role.id} value={role.id} disabled={!role.enabled} className="grant">
                        <b>{role.name}</b>
                        <small>{role.code}{role.enabled ? '' : ' · 已停用'}</small>
                      </Checkbox>
                    ))}
                  </Checkbox.Group>
                  <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 8 }}>
                    <Button theme="primary" loading={userRoleSaving} onClick={() => void saveUserRoles()}>
                      保存用户角色
                    </Button>
                  </div>
                </>
              )}
            </div>
          </article>
        </section>
      )}

      <Dialog
        header="新增角色"
        visible={roleVisible}
        onClose={() => setRoleVisible(false)}
        confirmBtn={{ content: '创建角色', loading: submitting, onClick: () => void addRole() }}
      >
        <Form labelWidth={82}>
          <Form.FormItem label="应用标识">
            <Input disabled value={activeApp} />
          </Form.FormItem>
          <Form.FormItem label="角色编码">
            <Input value={roleForm.code} onChange={(value) => setRoleForm({ ...roleForm, code: value })} />
          </Form.FormItem>
          <Form.FormItem label="角色名称">
            <Input value={roleForm.name} onChange={(value) => setRoleForm({ ...roleForm, name: value })} />
          </Form.FormItem>
          <Form.FormItem label="说明">
            <Textarea value={roleForm.description} onChange={(value) => setRoleForm({ ...roleForm, description: value })} />
          </Form.FormItem>
        </Form>
      </Dialog>

      <Dialog
        header="新增资源"
        visible={resourceVisible}
        width={560}
        onClose={() => setResourceVisible(false)}
        confirmBtn={{ content: '创建资源', loading: submitting, onClick: () => void addResource() }}
      >
        <Form labelWidth={82}>
          <Form.FormItem label="资源类型">
            <Radio.Group
              value={resourceForm.type}
              onChange={(value) => setResourceForm({ ...resourceForm, type: value as ResourceType, parent_id: value === 'menu' ? '' : resourceForm.parent_id })}
            >
              <Radio value="menu">菜单</Radio>
              <Radio value="button">按钮</Radio>
            </Radio.Group>
          </Form.FormItem>
          <Form.FormItem label="资源编码">
            <Input value={resourceForm.code} onChange={(value) => setResourceForm({ ...resourceForm, code: value })} />
          </Form.FormItem>
          <Form.FormItem label="资源名称">
            <Input value={resourceForm.name} onChange={(value) => setResourceForm({ ...resourceForm, name: value })} />
          </Form.FormItem>
          {resourceForm.type === 'button' && (
            <Form.FormItem label="父级菜单">
              <select value={resourceForm.parent_id} onChange={(event) => setResourceForm({ ...resourceForm, parent_id: event.target.value })}>
                <option value="">请选择菜单</option>
                {menus.map((menu) => (
                  <option key={menu.id} value={menu.id}>
                    {'-- '.repeat(menu.depth)}{menu.name}
                  </option>
                ))}
              </select>
            </Form.FormItem>
          )}
          <Form.FormItem label={resourceForm.type === 'menu' ? '前端路径' : 'API 路径'}>
            <Input
              value={resourceForm.type === 'menu' ? resourceForm.path : resourceForm.api_path}
              onChange={(value) => setResourceForm(resourceForm.type === 'menu' ? { ...resourceForm, path: value } : { ...resourceForm, api_path: value })}
              placeholder={resourceForm.type === 'menu' ? '/users' : '/v1/users'}
            />
          </Form.FormItem>
          {resourceForm.type === 'menu' && (
            <Form.FormItem label="组件路径">
              <Input value={resourceForm.component} onChange={(value) => setResourceForm({ ...resourceForm, component: value })} placeholder="/pages/users" />
            </Form.FormItem>
          )}
          {resourceForm.type === 'button' && (
            <Form.FormItem label="请求方法">
              <Input value={resourceForm.http_method} onChange={(value) => setResourceForm({ ...resourceForm, http_method: value.toUpperCase() })} />
            </Form.FormItem>
          )}
          <Form.FormItem label="图标编码">
            <Input value={resourceForm.icon} onChange={(value) => setResourceForm({ ...resourceForm, icon: value })} />
          </Form.FormItem>
          <Form.FormItem label="排序">
            <InputNumber value={resourceForm.sort} onChange={(value) => setResourceForm({ ...resourceForm, sort: Number(value || 0) })} />
          </Form.FormItem>
          <Form.FormItem label="说明">
            <Textarea value={resourceForm.description} onChange={(value) => setResourceForm({ ...resourceForm, description: value })} />
          </Form.FormItem>
        </Form>
      </Dialog>

      <Drawer
        header={grantRole ? `配置权限 · ${grantRole.name}` : '配置权限'}
        visible={Boolean(grantRole)}
        size="420px"
        footer={(
          <Space>
            <Button variant="outline" onClick={() => setGrantRole(undefined)}>取消</Button>
            <Button theme="primary" loading={submitting} onClick={() => void saveGrant()}>保存授权</Button>
          </Space>
        )}
        onClose={() => setGrantRole(undefined)}
      >
        <p>选择角色在 <b>{activeApp}</b> 下可访问的菜单和按钮。保存将全量替换当前授权。</p>
        <Checkbox.Group value={grants} onChange={(value) => setGrants(value.map(String))}>
          {resources.map((item) => (
            <Checkbox key={item.id} value={item.id} className="grant" style={{ marginLeft: item.depth * 18 }}>
              <b>{item.name}</b>
              <small>{item.type === 'menu' ? '菜单' : '按钮'} · {item.code}</small>
            </Checkbox>
          ))}
        </Checkbox.Group>
      </Drawer>
    </main>
  );
};

export default App;
