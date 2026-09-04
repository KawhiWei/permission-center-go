import { expect, test, type Page, type Route } from '@playwright/test';

const SERVICE_RESOURCE_STORAGE_KEY = 'permission-center-service-resource';
const LAYOUT_TABS_STORAGE_KEY = 'permission-center-layout-tabs';
const FIRST_SERVICE_RESOURCE = {
  id: 'service-resource-content',
  name: 'content-platform',
  display_name: '内容平台',
  audience: 'permission.center.api',
  description: '端到端测试服务资源一',
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
};
const SECOND_SERVICE_RESOURCE = {
  id: 'service-resource-analytics',
  name: 'analytics-platform',
  display_name: '数据平台',
  audience: 'permission.center.analytics',
  description: '端到端测试服务资源二',
  is_active: true,
  created_at: '2026-01-02T00:00:00Z',
};

type RoleRecord = {
  id: string;
  service_resource: string;
  code: string;
  name: string;
  description: string;
  enabled: boolean;
};

type ResourceRecord = {
  id: string;
  service_resource: string;
  code: string;
  resource_type: string;
  name: string;
  description: string;
  matcher: string;
  enabled: boolean;
};

type ActionRecord = {
  id: string;
  service_resource: string;
  code: string;
  name: string;
  description: string;
  enabled: boolean;
};

type PolicyRecord = {
  id: string;
  service_resource: string;
  code: string;
  name: string;
  description: string;
  effect: 'allow' | 'deny';
  priority: number;
  resource_codes: string[];
  action_codes: string[];
  enabled: boolean;
};

type PermissionMockState = {
  authenticated: boolean;
  rolesByServiceResource: Record<string, RoleRecord[]>;
  userRoleIDs: string[];
  roleRequests: string[];
  userRoleRequests: string[];
  userRoleReads: number;
  serviceResourceRequests: number;
  resourcesByServiceResource: Record<string, ResourceRecord[]>;
  actionsByServiceResource: Record<string, ActionRecord[]>;
  endpointsByServiceResource: Record<string, Record<string, unknown>[]>;
  policiesByServiceResource: Record<string, PolicyRecord[]>;
  bindingsByPolicy: Record<string, Record<string, unknown>[]>;
  decisionRequests: Record<string, unknown>[];
};

const makeRole = (
  serviceResource: string,
  id: string,
  code: string,
  name: string,
  description = '',
): RoleRecord => ({
  id,
  service_resource: serviceResource,
  code,
  name,
  description,
  enabled: true,
});

const makeResource = (
  serviceResource: string,
  id: string,
  code: string,
  name: string,
  resourceType = 'entity',
): ResourceRecord => ({
  id,
  service_resource: serviceResource,
  code,
  resource_type: resourceType,
  name,
  description: '',
  matcher: '',
  enabled: true,
});

const makeAction = (
  serviceResource: string,
  id: string,
  code: string,
  name: string,
): ActionRecord => ({
  id,
  service_resource: serviceResource,
  code,
  name,
  description: '',
  enabled: true,
});

const jsonResult = (result: unknown) => ({
  success: true,
  errorCode: null,
  errorMessage: null,
  result,
});

const fulfillJSON = async (route: Route, body: unknown, status = 200) => {
  await route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
};

const getServiceResourceRecords = () => [FIRST_SERVICE_RESOURCE, SECOND_SERVICE_RESOURCE];

const installAPIMocks = async (page: Page): Promise<PermissionMockState> => {
  const state: PermissionMockState = {
    authenticated: true,
    rolesByServiceResource: {
      [FIRST_SERVICE_RESOURCE.name]: [
        makeRole(FIRST_SERVICE_RESOURCE.name, 'role-editor', 'content-editor', '内容编辑'),
      ],
      [SECOND_SERVICE_RESOURCE.name]: [
        makeRole(SECOND_SERVICE_RESOURCE.name, 'role-analyst', 'data-analyst', '数据分析'),
      ],
    },
    userRoleIDs: [],
    roleRequests: [],
    userRoleRequests: [],
    userRoleReads: 0,
    serviceResourceRequests: 0,
    resourcesByServiceResource: {
      [FIRST_SERVICE_RESOURCE.name]: [
        makeResource(FIRST_SERVICE_RESOURCE.name, 'resource-post', 'post', '帖子'),
      ],
      [SECOND_SERVICE_RESOURCE.name]: [],
    },
    actionsByServiceResource: {
      [FIRST_SERVICE_RESOURCE.name]: [
        makeAction(FIRST_SERVICE_RESOURCE.name, 'action-update', 'update', '编辑'),
      ],
      [SECOND_SERVICE_RESOURCE.name]: [],
    },
    endpointsByServiceResource: {
      [FIRST_SERVICE_RESOURCE.name]: [],
      [SECOND_SERVICE_RESOURCE.name]: [],
    },
    policiesByServiceResource: {
      [FIRST_SERVICE_RESOURCE.name]: [],
      [SECOND_SERVICE_RESOURCE.name]: [],
    },
    bindingsByPolicy: {},
    decisionRequests: [],
  };

  await page.route('**/api/auth/me', async (route) => {
    if (!state.authenticated) {
      await fulfillJSON(route, {
        success: false,
        errorCode: 'UNAUTHENTICATED',
        errorMessage: '未登录',
        result: null,
      }, 401);
      return;
    }
    await fulfillJSON(route, jsonResult({
      isAuthenticated: true,
      user: { sub: 'e2e-user', name: 'Playwright 用户', email: 'e2e@example.test' },
    }));
  });

  await page.route('**/api/auth/logout', async (route) => {
    state.authenticated = false;
    await fulfillJSON(route, jsonResult({ logoutUrl: '' }));
  });

  await page.route('**/api/v1/service-resources**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() === 'GET' && url.pathname === '/api/v1/service-resources') {
      state.serviceResourceRequests += 1;
      await fulfillJSON(route, jsonResult({ items: getServiceResourceRecords() }));
      return;
    }

    await fulfillJSON(route, jsonResult(null), 404);
  });

  await page.route('**/api/v1/roles**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const serviceResource = url.searchParams.get('service_resource') || '';

    if (request.method() === 'GET' && url.pathname === '/api/v1/roles') {
      state.roleRequests.push(serviceResource);
      await fulfillJSON(route, jsonResult({ items: state.rolesByServiceResource[serviceResource] || [] }));
      return;
    }

    if (request.method() === 'POST' && url.pathname === '/api/v1/roles') {
      const payload = request.postDataJSON() as {
        service_resource?: string;
        code?: string;
        name?: string;
        description?: string;
      };
      const serviceResourceRoles = state.rolesByServiceResource[payload.service_resource || ''] || [];
      const role = makeRole(
        payload.service_resource || '',
        `role-${serviceResourceRoles.length + 1}`,
        payload.code || '',
        payload.name || '',
        payload.description || '',
      );
      state.rolesByServiceResource[role.service_resource] = [...serviceResourceRoles, role];
      await fulfillJSON(route, jsonResult(role));
      return;
    }

    await fulfillJSON(route, jsonResult(null), 404);
  });

  await page.route('**/api/v1/users/*/roles**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    state.userRoleRequests.push(`${request.method()} ${url.pathname}${url.search}`);
    const userPath = url.pathname.match(/^\/api\/v1\/users\/([^/]+)\/roles$/);
    if (!userPath) {
      await fulfillJSON(route, jsonResult(null), 404);
      return;
    }

    if (request.method() === 'GET') {
      state.userRoleReads += 1;
      await fulfillJSON(route, jsonResult({ role_ids: state.userRoleIDs }));
      return;
    }

    if (request.method() === 'PUT') {
      const payload = request.postDataJSON() as { role_ids?: unknown };
      state.userRoleIDs = Array.isArray(payload.role_ids)
        ? payload.role_ids.filter((roleID): roleID is string => typeof roleID === 'string')
        : [];
      await fulfillJSON(route, jsonResult(null));
      return;
    }

    await fulfillJSON(route, jsonResult(null), 404);
  });

  await page.route('**/api/v1/authorization/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const serviceResource = url.searchParams.get('service_resource') || FIRST_SERVICE_RESOURCE.name;
    const collectionMatch = path.match(/^\/api\/v1\/authorization\/(resources|actions|api-endpoints|policies)$/);
    const itemMatch = path.match(/^\/api\/v1\/authorization\/(resources|actions|api-endpoints|policies)\/([^/]+)$/);
    const bindingMatch = path.match(/^\/api\/v1\/authorization\/policies\/([^/]+)\/bindings$/);

    if (collectionMatch && request.method() === 'GET') {
      const collection = collectionMatch[1];
      const items = collection === 'resources'
        ? state.resourcesByServiceResource[serviceResource] || []
        : collection === 'actions'
          ? state.actionsByServiceResource[serviceResource] || []
          : collection === 'api-endpoints'
            ? state.endpointsByServiceResource[serviceResource] || []
            : state.policiesByServiceResource[serviceResource] || [];
      await fulfillJSON(route, jsonResult({ items }));
      return;
    }

    if (collectionMatch && request.method() === 'POST') {
      const collection = collectionMatch[1];
      const payload = request.postDataJSON() as Record<string, unknown>;
      const id = `${collection}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
      if (collection === 'resources') {
        const resource: ResourceRecord = {
          id,
          service_resource: String(payload.service_resource || serviceResource),
          code: String(payload.code || ''),
          resource_type: String(payload.resource_type || 'entity'),
          name: String(payload.name || ''),
          description: String(payload.description || ''),
          matcher: String(payload.matcher || ''),
          enabled: payload.enabled !== false,
        };
        state.resourcesByServiceResource[resource.service_resource] = [...(state.resourcesByServiceResource[resource.service_resource] || []), resource];
        await fulfillJSON(route, jsonResult(resource), 201);
        return;
      }
      if (collection === 'actions') {
        const action: ActionRecord = {
          id,
          service_resource: String(payload.service_resource || serviceResource),
          code: String(payload.code || ''),
          name: String(payload.name || ''),
          description: String(payload.description || ''),
          enabled: payload.enabled !== false,
        };
        state.actionsByServiceResource[action.service_resource] = [...(state.actionsByServiceResource[action.service_resource] || []), action];
        await fulfillJSON(route, jsonResult(action), 201);
        return;
      }
      if (collection === 'api-endpoints') {
        const endpoint = { id, service_resource: String(payload.service_resource || serviceResource), ...payload };
        state.endpointsByServiceResource[String(endpoint.service_resource)] = [...(state.endpointsByServiceResource[String(endpoint.service_resource)] || []), endpoint];
        await fulfillJSON(route, jsonResult(endpoint), 201);
        return;
      }
      const policy: PolicyRecord = {
        id,
        service_resource: String(payload.service_resource || serviceResource),
        code: String(payload.code || ''),
        name: String(payload.name || ''),
        description: String(payload.description || ''),
        effect: payload.effect === 'deny' ? 'deny' : 'allow',
        priority: Number(payload.priority || 0),
        resource_codes: Array.isArray(payload.resource_codes) ? payload.resource_codes.filter((value): value is string => typeof value === 'string') : [],
        action_codes: Array.isArray(payload.action_codes) ? payload.action_codes.filter((value): value is string => typeof value === 'string') : [],
        enabled: payload.enabled !== false,
      };
      state.policiesByServiceResource[policy.service_resource] = [...(state.policiesByServiceResource[policy.service_resource] || []), policy];
      await fulfillJSON(route, jsonResult(policy), 201);
      return;
    }

    if (itemMatch && request.method() === 'PUT') {
      const collection = itemMatch[1];
      const id = itemMatch[2];
      const payload = request.postDataJSON() as Record<string, unknown>;
      const collectionState = collection === 'resources'
        ? state.resourcesByServiceResource[serviceResource] || []
        : collection === 'actions'
          ? state.actionsByServiceResource[serviceResource] || []
          : collection === 'api-endpoints'
            ? state.endpointsByServiceResource[serviceResource] || []
            : state.policiesByServiceResource[serviceResource] || [];
      const nextItems = collectionState.map((item) => item.id === id ? { ...item, ...payload } : item);
      if (collection === 'resources') state.resourcesByServiceResource[serviceResource] = nextItems as ResourceRecord[];
      if (collection === 'actions') state.actionsByServiceResource[serviceResource] = nextItems as ActionRecord[];
      if (collection === 'api-endpoints') state.endpointsByServiceResource[serviceResource] = nextItems;
      if (collection === 'policies') state.policiesByServiceResource[serviceResource] = nextItems as PolicyRecord[];
      await fulfillJSON(route, jsonResult(nextItems.find((item) => item.id === id) || null));
      return;
    }

    if (itemMatch && request.method() === 'DELETE') {
      const collection = itemMatch[1];
      const id = itemMatch[2];
      if (collection === 'resources') state.resourcesByServiceResource[serviceResource] = (state.resourcesByServiceResource[serviceResource] || []).filter((item) => item.id !== id);
      if (collection === 'actions') state.actionsByServiceResource[serviceResource] = (state.actionsByServiceResource[serviceResource] || []).filter((item) => item.id !== id);
      if (collection === 'api-endpoints') state.endpointsByServiceResource[serviceResource] = (state.endpointsByServiceResource[serviceResource] || []).filter((item) => item.id !== id);
      if (collection === 'policies') state.policiesByServiceResource[serviceResource] = (state.policiesByServiceResource[serviceResource] || []).filter((item) => item.id !== id);
      await fulfillJSON(route, jsonResult(null));
      return;
    }

    if (bindingMatch) {
      const policyID = bindingMatch[1];
      if (request.method() === 'GET') {
        await fulfillJSON(route, jsonResult({ items: state.bindingsByPolicy[policyID] || [] }));
        return;
      }
      if (request.method() === 'PUT') {
        const payload = request.postDataJSON() as { bindings?: unknown };
        state.bindingsByPolicy[policyID] = Array.isArray(payload.bindings) ? payload.bindings.filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === 'object') : [];
        await fulfillJSON(route, jsonResult(null));
        return;
      }
    }

    await fulfillJSON(route, jsonResult(null), 404);
  });

  await page.route('**/api/v1/pdp/decisions', async (route) => {
    const request = route.request();
    if (request.method() !== 'POST') {
      await fulfillJSON(route, jsonResult(null), 404);
      return;
    }
    const payload = request.postDataJSON() as Record<string, unknown>;
    state.decisionRequests.push(payload);
    const allowed = payload.subject_id === 'allow-user' && payload.action === 'update';
    await fulfillJSON(route, jsonResult({
      allow: allowed,
      reason_code: allowed ? 'ALLOW' : 'NO_MATCHING_POLICY',
      matched_policy_ids: allowed ? ['policy-editor'] : [],
    }));
  });

  // Dashboard and role authorization use this endpoint when a test navigates there.
  await page.route('**/api/v1/menus/tree**', async (route) => {
    await fulfillJSON(route, jsonResult({ items: [] }));
  });

  return state;
};

const seedServiceResource = async (page: Page, serviceResource = FIRST_SERVICE_RESOURCE.name) => {
  await page.addInitScript(({ key, value }) => {
    window.localStorage.setItem(key, value);
  }, { key: SERVICE_RESOURCE_STORAGE_KEY, value: serviceResource });
};

test.describe('权限中心关键操作流程', () => {
  test('未选择服务资源时直接访问角色管理会重定向到服务资源选择页', async ({ page }) => {
    await installAPIMocks(page);
    await page.goto('/roles');

    await expect(page).toHaveURL(/\/select-service-resource\?redirect=%2Froles$/);
    await expect(page.getByRole('heading', { name: '选择服务资源' })).toBeVisible();
    await expect(page.getByText('可用服务资源', { exact: true })).toBeVisible();
  });

  test('选择的服务资源会在刷新后保留，并显示在 Header 中', async ({ page }) => {
    await installAPIMocks(page);
    await seedServiceResource(page);
    await page.goto('/dashboard');

    const currentServiceResource = page.locator('.layout-current-service-resource-value');
    await expect(currentServiceResource).toHaveText(FIRST_SERVICE_RESOURCE.name);

    await page.reload();

    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(currentServiceResource).toHaveText(FIRST_SERVICE_RESOURCE.name);
  });

  test('服务资源选择页可以刷新列表并保持只读目录', async ({ page }) => {
    const state = await installAPIMocks(page);
    await page.goto('/select-service-resource');

    await expect(page.getByRole('heading', { name: '选择服务资源' })).toBeVisible();
    await expect(page.getByText(FIRST_SERVICE_RESOURCE.display_name, { exact: true })).toBeVisible();
    await page.getByRole('button', { name: '刷新列表', exact: true }).click();
    await expect.poll(() => state.serviceResourceRequests).toBeGreaterThan(1);
    await expect(page.getByRole('button', { name: '新建服务资源', exact: true })).toHaveCount(0);
  });

  test('退出登录会清空历史页签，重新登录后只保留仪表盘', async ({ page }) => {
    const state = await installAPIMocks(page);
    await seedServiceResource(page);
    await page.goto('/roles');
    await page.goto('/authorization/resources');

    await expect.poll(async () => page.evaluate((key) => {
      const tabs = JSON.parse(window.localStorage.getItem(key) || '[]') as unknown[];
      return tabs.length;
    }, LAYOUT_TABS_STORAGE_KEY)).toBeGreaterThan(1);

    await page.locator('.layout-avatar-trigger').click();
    await page.getByText('退出登录', { exact: true }).click();
    await expect(page).toHaveURL(/\/login/);
    await expect.poll(() => page.evaluate(
      (key) => window.localStorage.getItem(key),
      LAYOUT_TABS_STORAGE_KEY,
    )).toBeNull();

    state.authenticated = true;
    await page.goto('/dashboard');
    const layoutTabs = page.locator('.layout-content-tabs .t-tabs__nav-item');
    await expect(layoutTabs).toHaveCount(1);
    await expect(layoutTabs.first()).toContainText('仪表盘');
    await expect(page.evaluate(
      (key) => JSON.parse(window.localStorage.getItem(key) || '[]'),
      LAYOUT_TABS_STORAGE_KEY,
    )).resolves.toEqual([{ value: '/dashboard', label: '仪表盘', removable: false }]);
  });

  test('Header 切换服务资源后返回原页面，并按新服务资源刷新角色列表', async ({ page }) => {
    const state = await installAPIMocks(page);
    await seedServiceResource(page);
    await page.goto('/roles');

    await expect(page.getByText('内容编辑', { exact: true })).toBeVisible();
    await page.getByRole('button', { name: '切换服务资源', exact: true }).click();
    await expect(page).toHaveURL(/\/select-service-resource\?redirect=%2Froles$/);

    await page.locator('.permission-service-resource-option').filter({ hasText: SECOND_SERVICE_RESOURCE.display_name }).click();
    await page.getByRole('button', { name: '进入权限中心', exact: true }).click();

    await expect(page).toHaveURL(/\/roles$/);
    await expect(page.locator('.layout-current-service-resource-value')).toHaveText(SECOND_SERVICE_RESOURCE.name);
    await expect(page.getByText('数据分析', { exact: true })).toBeVisible();
    await expect(page.getByText('内容编辑', { exact: true })).not.toBeVisible();
    expect(state.roleRequests).toContain(SECOND_SERVICE_RESOURCE.name);
  });

  test('创建角色成功后关闭弹窗、重置表单并刷新列表', async ({ page }) => {
    await installAPIMocks(page);
    await seedServiceResource(page);
    await page.goto('/roles');

    await page.getByRole('button', { name: '新建角色', exact: true }).click();
    const dialog = page.locator('.t-dialog').filter({ hasText: '新建角色' });
    await expect(dialog).toBeVisible();
    await dialog.getByPlaceholder('例如 content-editor').fill('new-role');
    await dialog.getByPlaceholder('例如 内容编辑').fill('新建角色');
    await dialog.getByPlaceholder('可选').fill('创建后应刷新');
    await dialog.getByRole('button', { name: '创建', exact: true }).click();

    await expect(dialog).toBeHidden();
    await expect(page.getByText('新建角色', { exact: true })).toBeVisible();

    await page.getByRole('button', { name: '新建角色', exact: true }).click();
    const freshDialog = page.locator('.t-dialog').filter({ hasText: '新建角色' });
    await expect(freshDialog.getByPlaceholder('例如 content-editor')).toHaveValue('');
    await expect(freshDialog.getByPlaceholder('例如 内容编辑')).toHaveValue('');
    await expect(freshDialog.getByPlaceholder('可选')).toHaveValue('');
  });

  test('用户角色绑定保存后会重新读取服务端角色数据', async ({ page }) => {
    const state = await installAPIMocks(page);
    await seedServiceResource(page);
    await page.goto('/user-roles');

    await page.getByPlaceholder('输入 NexusAuth subject').fill('e2e-user');
    await page.getByRole('button', { name: '查询角色', exact: true }).click();
    await expect(page.getByText('内容编辑', { exact: true })).toBeVisible();

    const roleOption = page.locator('.permission-role-option').filter({ hasText: '内容编辑' });
    await roleOption.locator('.t-checkbox__input').click();
    const replaceRequest = page.waitForRequest((request) => (
      request.method() === 'PUT'
      && request.url().includes('/api/v1/users/e2e-user/roles?service_resource=content-platform')
    ));
    await page.getByRole('button', { name: '保存绑定', exact: true }).click();
    await replaceRequest;

    await expect.poll(
      () => state.userRoleReads,
      { message: `用户角色请求：${state.userRoleRequests.join(', ')}` },
    ).toBeGreaterThan(1);
    expect(state.userRoleIDs).toEqual(['role-editor']);
    await expect(roleOption.locator('.t-checkbox.t-is-checked')).toBeVisible();
  });

  test('PDP 五类功能使用独立菜单和独立路由', async ({ page }) => {
    await installAPIMocks(page);
    await seedServiceResource(page);

    const pages = [
      ['/authorization/resources', '资源管理'],
      ['/authorization/actions', '动作管理'],
      ['/authorization/api-endpoints', 'API 端点管理'],
      ['/authorization/policies', '策略管理'],
      ['/authorization/simulator', '策略模拟'],
    ] as const;

    for (const [path, heading] of pages) {
      await page.goto(path);
      await expect(page).toHaveURL(new RegExp(`${path}$`));
      await expect(page.getByRole('heading', { name: heading })).toBeVisible();
      await expect(page.locator('.t-menu').getByText(heading, { exact: true })).toBeVisible();
      await expect(page.locator('.permission-pdp-page').getByRole('tab')).toHaveCount(0);
    }

    await page.goto('/pdp');
    await expect(page).toHaveURL(/\/authorization\/resources$/);
  });

  test('PDP 创建资源成功后关闭弹窗、重置表单并刷新列表', async ({ page }) => {
    await installAPIMocks(page);
    await seedServiceResource(page);
    await page.goto('/authorization/resources');

    await expect(page).toHaveURL(/\/authorization\/resources$/);
    await expect(page.getByRole('heading', { name: '资源管理' })).toBeVisible();
    await expect(page.getByRole('complementary').getByText('资源管理', { exact: true })).toBeVisible();
    await page.goto('/pdp');
    await expect(page).toHaveURL(/\/authorization\/resources$/);
    await expect(page.getByText('帖子', { exact: true })).toBeVisible();
    await page.getByRole('button', { name: '新建资源', exact: true }).click();

    const dialog = page.locator('.t-dialog').filter({ hasText: '新建资源' });
    await expect(dialog).toBeVisible();
    await dialog.getByPlaceholder('例如 post').fill('article');
    await dialog.getByPlaceholder('例如 帖子').fill('文章');
    await dialog.getByPlaceholder('例如 /v1/posts/{postId}').fill('/v1/articles/{articleId}');
    await dialog.getByRole('button', { name: '创建', exact: true }).click();

    await expect(dialog).toBeHidden();
    await expect(page.getByText('文章', { exact: true })).toBeVisible();

    await page.getByRole('button', { name: '新建资源', exact: true }).click();
    const freshDialog = page.locator('.t-dialog').filter({ hasText: '新建资源' });
    await expect(freshDialog.getByPlaceholder('例如 post')).toHaveValue('');
    await expect(freshDialog.getByPlaceholder('例如 帖子')).toHaveValue('');
    await expect(freshDialog.getByPlaceholder('例如 /v1/posts/{postId}')).toHaveValue('');
  });

  test('PDP 策略模拟会展示 allow 和 deny 结果', async ({ page }) => {
    const state = await installAPIMocks(page);
    await seedServiceResource(page);
    await page.goto('/authorization/simulator');

    await expect(page).toHaveURL(/\/authorization\/simulator$/);
    await expect(page.getByRole('heading', { name: '策略模拟' })).toBeVisible();
    await page.getByPlaceholder('例如 nexus-user-001').fill('allow-user');
    await page.getByRole('textbox', { name: '例如 post', exact: true }).fill('post');
    await page.getByPlaceholder('例如 update').fill('update');
    await page.getByRole('button', { name: '执行模拟', exact: true }).click();

    await expect(page.getByText('允许', { exact: true })).toBeVisible();
    await expect(page.getByText('ALLOW', { exact: true })).toBeVisible();

    await page.getByPlaceholder('例如 nexus-user-001').fill('deny-user');
    await page.getByRole('button', { name: '执行模拟', exact: true }).click();
    await expect(page.getByText('拒绝', { exact: true })).toBeVisible();
    await expect(page.getByText('NO_MATCHING_POLICY', { exact: true })).toBeVisible();
    expect(state.decisionRequests).toHaveLength(2);
  });
});
