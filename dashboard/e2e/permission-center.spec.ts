import { expect, test, type Page, type Route } from '@playwright/test';

const APPLICATION_STORAGE_KEY = 'permission-center-application';
const FIRST_APPLICATION = {
  application: 'content-platform',
  name: '内容平台',
  description: '端到端测试应用一',
  enabled: true,
  is_deleted: false,
};
const SECOND_APPLICATION = {
  application: 'analytics-platform',
  name: '数据平台',
  description: '端到端测试应用二',
  enabled: true,
  is_deleted: false,
};

type RoleRecord = {
  id: string;
  application: string;
  code: string;
  name: string;
  description: string;
  enabled: boolean;
};

type ResourceRecord = {
  id: string;
  application: string;
  code: string;
  resource_type: string;
  name: string;
  description: string;
  matcher: string;
  enabled: boolean;
};

type ActionRecord = {
  id: string;
  application: string;
  code: string;
  name: string;
  description: string;
  enabled: boolean;
};

type PolicyRecord = {
  id: string;
  application: string;
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
  rolesByApplication: Record<string, RoleRecord[]>;
  userRoleIDs: string[];
  roleRequests: string[];
  userRoleRequests: string[];
  userRoleReads: number;
  resourcesByApplication: Record<string, ResourceRecord[]>;
  actionsByApplication: Record<string, ActionRecord[]>;
  endpointsByApplication: Record<string, Record<string, unknown>[]>;
  policiesByApplication: Record<string, PolicyRecord[]>;
  bindingsByPolicy: Record<string, Record<string, unknown>[]>;
  decisionRequests: Record<string, unknown>[];
};

const makeRole = (
  application: string,
  id: string,
  code: string,
  name: string,
  description = '',
): RoleRecord => ({
  id,
  application,
  code,
  name,
  description,
  enabled: true,
});

const makeResource = (
  application: string,
  id: string,
  code: string,
  name: string,
  resourceType = 'entity',
): ResourceRecord => ({
  id,
  application,
  code,
  resource_type: resourceType,
  name,
  description: '',
  matcher: '',
  enabled: true,
});

const makeAction = (
  application: string,
  id: string,
  code: string,
  name: string,
): ActionRecord => ({
  id,
  application,
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

const getApplicationRecords = () => [FIRST_APPLICATION, SECOND_APPLICATION];

const installAPIMocks = async (page: Page): Promise<PermissionMockState> => {
  const state: PermissionMockState = {
    rolesByApplication: {
      [FIRST_APPLICATION.application]: [
        makeRole(FIRST_APPLICATION.application, 'role-editor', 'content-editor', '内容编辑'),
      ],
      [SECOND_APPLICATION.application]: [
        makeRole(SECOND_APPLICATION.application, 'role-analyst', 'data-analyst', '数据分析'),
      ],
    },
    userRoleIDs: [],
    roleRequests: [],
    userRoleRequests: [],
    userRoleReads: 0,
    resourcesByApplication: {
      [FIRST_APPLICATION.application]: [
        makeResource(FIRST_APPLICATION.application, 'resource-post', 'post', '帖子'),
      ],
      [SECOND_APPLICATION.application]: [],
    },
    actionsByApplication: {
      [FIRST_APPLICATION.application]: [
        makeAction(FIRST_APPLICATION.application, 'action-update', 'update', '编辑'),
      ],
      [SECOND_APPLICATION.application]: [],
    },
    endpointsByApplication: {
      [FIRST_APPLICATION.application]: [],
      [SECOND_APPLICATION.application]: [],
    },
    policiesByApplication: {
      [FIRST_APPLICATION.application]: [],
      [SECOND_APPLICATION.application]: [],
    },
    bindingsByPolicy: {},
    decisionRequests: [],
  };

  await page.route('**/api/auth/me', async (route) => {
    await fulfillJSON(route, jsonResult({
      isAuthenticated: true,
      user: { sub: 'e2e-user', name: 'Playwright 用户', email: 'e2e@example.test' },
    }));
  });

  await page.route('**/api/v1/applications**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() === 'GET' && url.pathname === '/api/v1/applications') {
      await fulfillJSON(route, jsonResult({ items: getApplicationRecords() }));
      return;
    }

    await fulfillJSON(route, jsonResult(null), 404);
  });

  await page.route('**/api/v1/roles**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const application = url.searchParams.get('application') || '';

    if (request.method() === 'GET' && url.pathname === '/api/v1/roles') {
      state.roleRequests.push(application);
      await fulfillJSON(route, jsonResult({ items: state.rolesByApplication[application] || [] }));
      return;
    }

    if (request.method() === 'POST' && url.pathname === '/api/v1/roles') {
      const payload = request.postDataJSON() as {
        application?: string;
        code?: string;
        name?: string;
        description?: string;
      };
      const applicationRoles = state.rolesByApplication[payload.application || ''] || [];
      const role = makeRole(
        payload.application || '',
        `role-${applicationRoles.length + 1}`,
        payload.code || '',
        payload.name || '',
        payload.description || '',
      );
      state.rolesByApplication[role.application] = [...applicationRoles, role];
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
    const application = url.searchParams.get('application') || FIRST_APPLICATION.application;
    const collectionMatch = path.match(/^\/api\/v1\/authorization\/(resources|actions|api-endpoints|policies)$/);
    const itemMatch = path.match(/^\/api\/v1\/authorization\/(resources|actions|api-endpoints|policies)\/([^/]+)$/);
    const bindingMatch = path.match(/^\/api\/v1\/authorization\/policies\/([^/]+)\/bindings$/);

    if (collectionMatch && request.method() === 'GET') {
      const collection = collectionMatch[1];
      const items = collection === 'resources'
        ? state.resourcesByApplication[application] || []
        : collection === 'actions'
          ? state.actionsByApplication[application] || []
          : collection === 'api-endpoints'
            ? state.endpointsByApplication[application] || []
            : state.policiesByApplication[application] || [];
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
          application: String(payload.application || application),
          code: String(payload.code || ''),
          resource_type: String(payload.resource_type || 'entity'),
          name: String(payload.name || ''),
          description: String(payload.description || ''),
          matcher: String(payload.matcher || ''),
          enabled: payload.enabled !== false,
        };
        state.resourcesByApplication[resource.application] = [...(state.resourcesByApplication[resource.application] || []), resource];
        await fulfillJSON(route, jsonResult(resource), 201);
        return;
      }
      if (collection === 'actions') {
        const action: ActionRecord = {
          id,
          application: String(payload.application || application),
          code: String(payload.code || ''),
          name: String(payload.name || ''),
          description: String(payload.description || ''),
          enabled: payload.enabled !== false,
        };
        state.actionsByApplication[action.application] = [...(state.actionsByApplication[action.application] || []), action];
        await fulfillJSON(route, jsonResult(action), 201);
        return;
      }
      if (collection === 'api-endpoints') {
        const endpoint = { id, application: String(payload.application || application), ...payload };
        state.endpointsByApplication[String(endpoint.application)] = [...(state.endpointsByApplication[String(endpoint.application)] || []), endpoint];
        await fulfillJSON(route, jsonResult(endpoint), 201);
        return;
      }
      const policy: PolicyRecord = {
        id,
        application: String(payload.application || application),
        code: String(payload.code || ''),
        name: String(payload.name || ''),
        description: String(payload.description || ''),
        effect: payload.effect === 'deny' ? 'deny' : 'allow',
        priority: Number(payload.priority || 0),
        resource_codes: Array.isArray(payload.resource_codes) ? payload.resource_codes.filter((value): value is string => typeof value === 'string') : [],
        action_codes: Array.isArray(payload.action_codes) ? payload.action_codes.filter((value): value is string => typeof value === 'string') : [],
        enabled: payload.enabled !== false,
      };
      state.policiesByApplication[policy.application] = [...(state.policiesByApplication[policy.application] || []), policy];
      await fulfillJSON(route, jsonResult(policy), 201);
      return;
    }

    if (itemMatch && request.method() === 'PUT') {
      const collection = itemMatch[1];
      const id = itemMatch[2];
      const payload = request.postDataJSON() as Record<string, unknown>;
      const collectionState = collection === 'resources'
        ? state.resourcesByApplication[application] || []
        : collection === 'actions'
          ? state.actionsByApplication[application] || []
          : collection === 'api-endpoints'
            ? state.endpointsByApplication[application] || []
            : state.policiesByApplication[application] || [];
      const nextItems = collectionState.map((item) => item.id === id ? { ...item, ...payload } : item);
      if (collection === 'resources') state.resourcesByApplication[application] = nextItems as ResourceRecord[];
      if (collection === 'actions') state.actionsByApplication[application] = nextItems as ActionRecord[];
      if (collection === 'api-endpoints') state.endpointsByApplication[application] = nextItems;
      if (collection === 'policies') state.policiesByApplication[application] = nextItems as PolicyRecord[];
      await fulfillJSON(route, jsonResult(nextItems.find((item) => item.id === id) || null));
      return;
    }

    if (itemMatch && request.method() === 'DELETE') {
      const collection = itemMatch[1];
      const id = itemMatch[2];
      if (collection === 'resources') state.resourcesByApplication[application] = (state.resourcesByApplication[application] || []).filter((item) => item.id !== id);
      if (collection === 'actions') state.actionsByApplication[application] = (state.actionsByApplication[application] || []).filter((item) => item.id !== id);
      if (collection === 'api-endpoints') state.endpointsByApplication[application] = (state.endpointsByApplication[application] || []).filter((item) => item.id !== id);
      if (collection === 'policies') state.policiesByApplication[application] = (state.policiesByApplication[application] || []).filter((item) => item.id !== id);
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

const seedApplication = async (page: Page, application = FIRST_APPLICATION.application) => {
  await page.addInitScript(({ key, value }) => {
    window.localStorage.setItem(key, value);
  }, { key: APPLICATION_STORAGE_KEY, value: application });
};

test.describe('权限中心关键操作流程', () => {
  test('未选择应用时直接访问角色管理会重定向到应用选择页', async ({ page }) => {
    await installAPIMocks(page);
    await page.goto('/roles');

    await expect(page).toHaveURL(/\/select-application\?redirect=%2Froles$/);
    await expect(page.getByRole('heading', { name: '选择应用' })).toBeVisible();
    await expect(page.getByText('可用应用', { exact: true })).toBeVisible();
  });

  test('选择的应用会在刷新后保留，并显示在 Header 中', async ({ page }) => {
    await installAPIMocks(page);
    await seedApplication(page);
    await page.goto('/dashboard');

    const currentApplication = page.locator('.layout-current-application-value');
    await expect(currentApplication).toHaveText(FIRST_APPLICATION.application);

    await page.reload();

    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(currentApplication).toHaveText(FIRST_APPLICATION.application);
  });

  test('Header 切换应用后返回原页面，并按新应用刷新角色列表', async ({ page }) => {
    const state = await installAPIMocks(page);
    await seedApplication(page);
    await page.goto('/roles');

    await expect(page.getByText('内容编辑', { exact: true })).toBeVisible();
    await page.getByRole('button', { name: '切换应用', exact: true }).click();
    await expect(page).toHaveURL(/\/select-application\?redirect=%2Froles$/);

    await page.locator('.permission-application-option').filter({ hasText: SECOND_APPLICATION.name }).click();
    await page.getByRole('button', { name: '进入权限中心', exact: true }).click();

    await expect(page).toHaveURL(/\/roles$/);
    await expect(page.locator('.layout-current-application-value')).toHaveText(SECOND_APPLICATION.application);
    await expect(page.getByText('数据分析', { exact: true })).toBeVisible();
    await expect(page.getByText('内容编辑', { exact: true })).not.toBeVisible();
    expect(state.roleRequests).toContain(SECOND_APPLICATION.application);
  });

  test('创建角色成功后关闭弹窗、重置表单并刷新列表', async ({ page }) => {
    await installAPIMocks(page);
    await seedApplication(page);
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
    await seedApplication(page);
    await page.goto('/user-roles');

    await page.getByPlaceholder('输入 NexusAuth subject').fill('e2e-user');
    await page.getByRole('button', { name: '查询角色', exact: true }).click();
    await expect(page.getByText('内容编辑', { exact: true })).toBeVisible();

    const roleOption = page.locator('.permission-role-option').filter({ hasText: '内容编辑' });
    await roleOption.locator('.t-checkbox__input').click();
    const replaceRequest = page.waitForRequest((request) => (
      request.method() === 'PUT'
      && request.url().includes('/api/v1/users/e2e-user/roles?application=content-platform')
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
    await seedApplication(page);

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
    await seedApplication(page);
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
    await seedApplication(page);
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
