import request from './request';

export const APPLICATION_STORAGE_KEY = 'permission-center-application';
export const APPLICATION_CHANGE_EVENT = 'permission-center-application-change';
export const DEFAULT_APPLICATION = '';

export type Application = {
  application: string;
  name: string;
  description: string;
  enabled: boolean;
  createdById?: string;
  createdByName?: string;
  createdAt?: string;
  updatedById?: string;
  updatedByName?: string;
  updatedAt?: string;
  isDeleted?: boolean;
};

export type CreateApplicationRequest = {
  application: string;
  name: string;
  description?: string;
};

export type UpdateApplicationRequest = {
  name: string;
  description?: string;
  enabled: boolean;
};

export type Role = {
  id: string;
  application: string;
  code: string;
  name: string;
  description: string;
  enabled: boolean;
  createdById?: string;
  createdByName?: string;
  createdAt?: string;
  updatedById?: string;
  updatedByName?: string;
  updatedAt?: string;
};

export type MenuType = 'menu' | 'button';

export type Menu = {
  id: string;
  application: string;
  parentId: string | null;
  code: string;
  name: string;
  description: string;
  type: MenuType;
  path: string;
  component: string;
  apiPath: string;
  httpMethod: string;
  icon: string;
  sort: number;
  enabled: boolean;
  createdById?: string;
  createdByName?: string;
  createdAt?: string;
  updatedById?: string;
  updatedByName?: string;
  updatedAt?: string;
  children: Menu[];
};

export type CreateRoleRequest = {
  application: string;
  code: string;
  name: string;
  description?: string;
};

export type CreateMenuRequest = {
  application: string;
  parent_id: string | null;
  code: string;
  name: string;
  description?: string;
  type: MenuType;
  path?: string;
  component?: string;
  api_path?: string;
  http_method?: string;
  icon?: string;
  sort?: number;
};

type RawRecord = Record<string, unknown>;

const rawString = (record: RawRecord, ...keys: string[]): string => {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'string') {
      return value;
    }
  }
  return '';
};

const rawBoolean = (record: RawRecord, ...keys: string[]): boolean => {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'boolean') {
      return value;
    }
  }
  return false;
};

const rawNumber = (record: RawRecord, ...keys: string[]): number => {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value;
    }
  }
  return 0;
};

const rawDate = (record: RawRecord, ...keys: string[]): string | undefined => {
  const value = rawString(record, ...keys);
  return value || undefined;
};

const normalizeRole = (value: unknown): Role => {
  const record = (value && typeof value === 'object' ? value : {}) as RawRecord;
  return {
    id: rawString(record, 'id', 'ID'),
    application: rawString(record, 'application', 'Application'),
    code: rawString(record, 'code', 'Code'),
    name: rawString(record, 'name', 'Name'),
    description: rawString(record, 'description', 'Description'),
    enabled: rawBoolean(record, 'enabled', 'Enabled'),
    createdById: rawString(record, 'created_by_id', 'CreatedByID') || undefined,
    createdByName: rawString(record, 'created_by_name', 'CreatedByName') || undefined,
    createdAt: rawDate(record, 'created_at', 'CreatedAt'),
    updatedById: rawString(record, 'updated_by_id', 'UpdatedByID') || undefined,
    updatedByName: rawString(record, 'updated_by_name', 'UpdatedByName') || undefined,
    updatedAt: rawDate(record, 'updated_at', 'UpdatedAt'),
  };
};

const normalizeApplication = (value: unknown): Application => {
  const record = (value && typeof value === 'object' ? value : {}) as RawRecord;
  return {
    application: rawString(record, 'application', 'Application'),
    name: rawString(record, 'name', 'Name'),
    description: rawString(record, 'description', 'Description'),
    enabled: rawBoolean(record, 'enabled', 'Enabled'),
    createdById: rawString(record, 'created_by_id', 'CreatedByID') || undefined,
    createdByName: rawString(record, 'created_by_name', 'CreatedByName') || undefined,
    createdAt: rawDate(record, 'created_at', 'CreatedAt'),
    updatedById: rawString(record, 'updated_by_id', 'UpdatedByID') || undefined,
    updatedByName: rawString(record, 'updated_by_name', 'UpdatedByName') || undefined,
    updatedAt: rawDate(record, 'updated_at', 'UpdatedAt'),
    isDeleted: rawBoolean(record, 'is_deleted', 'IsDeleted'),
  };
};

const normalizeMenu = (value: unknown): Menu => {
  const record = (value && typeof value === 'object' ? value : {}) as RawRecord;
  const type = rawString(record, 'type', 'Type') === 'button' ? 'button' : 'menu';
  const childrenValue = record.children ?? record.Children;
  const children = Array.isArray(childrenValue) ? childrenValue.map(normalizeMenu) : [];
  const parentID = rawString(record, 'parent_id', 'ParentID');

  return {
    id: rawString(record, 'id', 'ID'),
    application: rawString(record, 'application', 'Application'),
    parentId: parentID || null,
    code: rawString(record, 'code', 'Code'),
    name: rawString(record, 'name', 'Name'),
    description: rawString(record, 'description', 'Description'),
    type,
    path: rawString(record, 'path', 'Path', 'route', 'Route'),
    component: rawString(record, 'component', 'Component'),
    apiPath: rawString(record, 'api_path', 'APIPath', 'action', 'Action'),
    httpMethod: rawString(record, 'http_method', 'HTTPMethod'),
    icon: rawString(record, 'icon', 'Icon'),
    sort: rawNumber(record, 'sort', 'Sort', 'sort_order', 'SortOrder'),
    enabled: rawBoolean(record, 'enabled', 'Enabled'),
    createdById: rawString(record, 'created_by_id', 'CreatedByID') || undefined,
    createdByName: rawString(record, 'created_by_name', 'CreatedByName') || undefined,
    createdAt: rawDate(record, 'created_at', 'CreatedAt'),
    updatedById: rawString(record, 'updated_by_id', 'UpdatedByID') || undefined,
    updatedByName: rawString(record, 'updated_by_name', 'UpdatedByName') || undefined,
    updatedAt: rawDate(record, 'updated_at', 'UpdatedAt'),
    children,
  };
};

const readItems = (value: unknown): unknown[] => {
  if (Array.isArray(value)) {
    return value;
  }
  if (value && typeof value === 'object') {
    const items = (value as RawRecord).items;
    return Array.isArray(items) ? items : [];
  }
  return [];
};

const readItem = (value: unknown): unknown => {
  if (!value || typeof value !== 'object') {
    return value;
  }
  const record = value as RawRecord;
  return record.item ?? record.data ?? value;
};

const readIDs = (value: unknown, key: 'menu_ids' | 'role_ids'): string[] => {
  if (!value || typeof value !== 'object') {
    return [];
  }
  const values = (value as RawRecord)[key];
  return Array.isArray(values) ? values.filter((item): item is string => typeof item === 'string') : [];
};

export const getStoredApplication = (): string => {
  if (typeof window === 'undefined') {
    return DEFAULT_APPLICATION;
  }
  try {
    return window.localStorage.getItem(APPLICATION_STORAGE_KEY)?.trim() || DEFAULT_APPLICATION;
  } catch {
    return DEFAULT_APPLICATION;
  }
};

export const setStoredApplication = (application: string): string => {
  const next = application.trim();
  if (typeof window !== 'undefined') {
    try {
      if (next) {
        window.localStorage.setItem(APPLICATION_STORAGE_KEY, next);
      } else {
        window.localStorage.removeItem(APPLICATION_STORAGE_KEY);
      }
      window.dispatchEvent(new CustomEvent(APPLICATION_CHANGE_EVENT, { detail: next }));
    } catch {
      // Browsers can disable localStorage in private or restricted contexts.
    }
  }
  return next;
};

export const listApplications = async (): Promise<Application[]> => {
  const response = await request.get<unknown>('/v1/applications');
  return readItems(response).map(normalizeApplication);
};

export const getApplication = async (application: string): Promise<Application> => {
  const key = application.trim();
  const response = await request.get<unknown>(`/v1/applications/${encodeURIComponent(key)}`);
  return normalizeApplication(readItem(response));
};

export const createApplication = async (payload: CreateApplicationRequest): Promise<Application> => {
  const response = await request.post<unknown>('/v1/applications', payload);
  return normalizeApplication(readItem(response));
};

export const updateApplication = async (
  application: string,
  payload: UpdateApplicationRequest,
): Promise<Application> => {
  const key = application.trim();
  const response = await request.put<unknown>(`/v1/applications/${encodeURIComponent(key)}`, payload);
  return normalizeApplication(readItem(response));
};

export const deleteApplication = async (application: string): Promise<void> => {
  const key = application.trim();
  await request.delete<unknown>(`/v1/applications/${encodeURIComponent(key)}`);
};

export const listRoles = async (application: string): Promise<Role[]> => {
  const response = await request.get<unknown>(`/v1/roles?application=${encodeURIComponent(application.trim())}`);
  return readItems(response).map(normalizeRole);
};

export const createRole = async (payload: CreateRoleRequest): Promise<Role> => {
  const response = await request.post<unknown>('/v1/roles', payload);
  return normalizeRole(response);
};

export const getMenuTree = async (application: string): Promise<Menu[]> => {
  const response = await request.get<unknown>(`/v1/menus/tree?application=${encodeURIComponent(application.trim())}`);
  return readItems(response).map(normalizeMenu);
};

export const createMenu = async (payload: CreateMenuRequest): Promise<Menu> => {
  const response = await request.post<unknown>('/v1/menus', payload);
  return normalizeMenu(response);
};

export const getRoleMenuIDs = async (roleID: string): Promise<string[]> => {
  const response = await request.get<unknown>(`/v1/roles/${encodeURIComponent(roleID)}/menus`);
  return readIDs(response, 'menu_ids');
};

export const replaceRoleMenus = async (roleID: string, menuIDs: string[]): Promise<void> => {
  await request.put<unknown>(`/v1/roles/${encodeURIComponent(roleID)}/menus`, { menu_ids: menuIDs });
};

export const getUserRoleIDs = async (userID: string, application: string): Promise<string[]> => {
  const response = await request.get<unknown>(
    `/v1/users/${encodeURIComponent(userID)}/roles?application=${encodeURIComponent(application.trim())}`,
  );
  return readIDs(response, 'role_ids');
};

export const replaceUserRoles = async (userID: string, application: string, roleIDs: string[]): Promise<void> => {
  await request.put<unknown>(
    `/v1/users/${encodeURIComponent(userID)}/roles?application=${encodeURIComponent(application.trim())}`,
    { role_ids: roleIDs },
  );
};
