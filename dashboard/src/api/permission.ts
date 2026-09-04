import request from './request';

export const SERVICE_RESOURCE_STORAGE_KEY = 'permission-center-service-resource';
export const SERVICE_RESOURCE_CHANGE_EVENT = 'permission-center-service-resource-change';
export const DEFAULT_SERVICE_RESOURCE = '';

export type ServiceResource = {
  id: string;
  name: string;
  displayName: string;
  audience: string;
  description: string;
  isActive: boolean;
  createdAt?: string;
};

export type Role = {
  id: string;
  serviceResource: string;
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
  serviceResource: string;
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
  service_resource: string;
  code: string;
  name: string;
  description?: string;
};

export type CreateMenuRequest = {
  service_resource: string;
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

const normalizeServiceResource = (value: unknown): ServiceResource => {
  const record = (value && typeof value === 'object' ? value : {}) as RawRecord;
  return {
    id: rawString(record, 'id', 'ID'),
    name: rawString(record, 'name', 'Name'),
    displayName: rawString(record, 'display_name', 'displayName', 'DisplayName'),
    audience: rawString(record, 'audience', 'Audience'),
    description: rawString(record, 'description', 'Description'),
    isActive: rawBoolean(record, 'is_active', 'isActive', 'IsActive'),
    createdAt: rawDate(record, 'created_at', 'createdAt', 'CreatedAt'),
  };
};

const normalizeRole = (value: unknown): Role => {
  const record = (value && typeof value === 'object' ? value : {}) as RawRecord;
  return {
    id: rawString(record, 'id', 'ID'),
    serviceResource: rawString(record, 'service_resource', 'serviceResource', 'ServiceResource'),
    code: rawString(record, 'code', 'Code'),
    name: rawString(record, 'name', 'Name'),
    description: rawString(record, 'description', 'Description'),
    enabled: rawBoolean(record, 'enabled', 'Enabled'),
    createdById: rawString(record, 'created_by_id', 'createdById', 'CreatedByID') || undefined,
    createdByName: rawString(record, 'created_by_name', 'createdByName', 'CreatedByName') || undefined,
    createdAt: rawDate(record, 'created_at', 'createdAt', 'CreatedAt'),
    updatedById: rawString(record, 'updated_by_id', 'updatedById', 'UpdatedByID') || undefined,
    updatedByName: rawString(record, 'updated_by_name', 'updatedByName', 'UpdatedByName') || undefined,
    updatedAt: rawDate(record, 'updated_at', 'updatedAt', 'UpdatedAt'),
  };
};

const normalizeMenu = (value: unknown): Menu => {
  const record = (value && typeof value === 'object' ? value : {}) as RawRecord;
  const type = rawString(record, 'type', 'Type') === 'button' ? 'button' : 'menu';
  const childrenValue = record.children ?? record.Children;
  const children = Array.isArray(childrenValue) ? childrenValue.map(normalizeMenu) : [];
  const parentID = rawString(record, 'parent_id', 'parentId', 'ParentID');

  return {
    id: rawString(record, 'id', 'ID'),
    serviceResource: rawString(record, 'service_resource', 'serviceResource', 'ServiceResource'),
    parentId: parentID || null,
    code: rawString(record, 'code', 'Code'),
    name: rawString(record, 'name', 'Name'),
    description: rawString(record, 'description', 'Description'),
    type,
    path: rawString(record, 'path', 'Path', 'route', 'Route'),
    component: rawString(record, 'component', 'Component'),
    apiPath: rawString(record, 'api_path', 'apiPath', 'APIPath', 'action', 'Action'),
    httpMethod: rawString(record, 'http_method', 'httpMethod', 'HTTPMethod'),
    icon: rawString(record, 'icon', 'Icon'),
    sort: rawNumber(record, 'sort', 'Sort', 'sort_order', 'sortOrder', 'SortOrder'),
    enabled: rawBoolean(record, 'enabled', 'Enabled'),
    createdById: rawString(record, 'created_by_id', 'createdById', 'CreatedByID') || undefined,
    createdByName: rawString(record, 'created_by_name', 'createdByName', 'CreatedByName') || undefined,
    createdAt: rawDate(record, 'created_at', 'createdAt', 'CreatedAt'),
    updatedById: rawString(record, 'updated_by_id', 'updatedById', 'UpdatedByID') || undefined,
    updatedByName: rawString(record, 'updated_by_name', 'updatedByName', 'UpdatedByName') || undefined,
    updatedAt: rawDate(record, 'updated_at', 'updatedAt', 'UpdatedAt'),
    children,
  };
};

const readItems = (value: unknown): unknown[] => {
  if (Array.isArray(value)) {
    return value;
  }
  if (value && typeof value === 'object') {
    const record = value as RawRecord;
    const items = record.items ?? record.Items ?? record.data ?? record.Data;
    return Array.isArray(items) ? items : [];
  }
  return [];
};

const readItem = (value: unknown): unknown => {
  if (!value || typeof value !== 'object') {
    return value;
  }
  const record = value as RawRecord;
  return record.item ?? record.Item ?? record.data ?? record.Data ?? value;
};

const readIDs = (value: unknown, key: 'menu_ids' | 'role_ids'): string[] => {
  if (!value || typeof value !== 'object') {
    return [];
  }
  const values = (value as RawRecord)[key];
  return Array.isArray(values) ? values.filter((item): item is string => typeof item === 'string') : [];
};

export const getStoredServiceResource = (): string => {
  if (typeof window === 'undefined') {
    return DEFAULT_SERVICE_RESOURCE;
  }
  try {
    return window.localStorage.getItem(SERVICE_RESOURCE_STORAGE_KEY)?.trim() || DEFAULT_SERVICE_RESOURCE;
  } catch {
    return DEFAULT_SERVICE_RESOURCE;
  }
};

export const setStoredServiceResource = (serviceResource: string): string => {
  const next = serviceResource.trim();
  if (typeof window !== 'undefined') {
    try {
      if (next) {
        window.localStorage.setItem(SERVICE_RESOURCE_STORAGE_KEY, next);
      } else {
        window.localStorage.removeItem(SERVICE_RESOURCE_STORAGE_KEY);
      }
      window.dispatchEvent(new CustomEvent(SERVICE_RESOURCE_CHANGE_EVENT, { detail: next }));
    } catch {
      // Browsers can disable localStorage in private or restricted contexts.
    }
  }
  return next;
};

export const listServiceResources = async (): Promise<ServiceResource[]> => {
  const response = await request.get<unknown>('/v1/service-resources');
  return readItems(response).map(normalizeServiceResource);
};

export const getServiceResource = async (name: string): Promise<ServiceResource> => {
  const key = name.trim();
  const response = await request.get<unknown>(`/v1/service-resources/${encodeURIComponent(key)}`);
  return normalizeServiceResource(readItem(response));
};

export const listRoles = async (serviceResource: string): Promise<Role[]> => {
  const response = await request.get<unknown>(`/v1/roles?service_resource=${encodeURIComponent(serviceResource.trim())}`);
  return readItems(response).map(normalizeRole);
};

export const createRole = async (payload: CreateRoleRequest): Promise<Role> => {
  const response = await request.post<unknown>('/v1/roles', payload);
  return normalizeRole(readItem(response));
};

export const getMenuTree = async (serviceResource: string): Promise<Menu[]> => {
  const response = await request.get<unknown>(`/v1/menus/tree?service_resource=${encodeURIComponent(serviceResource.trim())}`);
  return readItems(response).map(normalizeMenu);
};

export const createMenu = async (payload: CreateMenuRequest): Promise<Menu> => {
  const response = await request.post<unknown>('/v1/menus', payload);
  return normalizeMenu(readItem(response));
};

export const getRoleMenuIDs = async (roleID: string): Promise<string[]> => {
  const response = await request.get<unknown>(`/v1/roles/${encodeURIComponent(roleID)}/menus`);
  return readIDs(response, 'menu_ids');
};

export const replaceRoleMenus = async (roleID: string, menuIDs: string[]): Promise<void> => {
  await request.put<unknown>(`/v1/roles/${encodeURIComponent(roleID)}/menus`, { menu_ids: menuIDs });
};

export const getUserRoleIDs = async (userID: string, serviceResource: string): Promise<string[]> => {
  const response = await request.get<unknown>(
    `/v1/users/${encodeURIComponent(userID)}/roles?service_resource=${encodeURIComponent(serviceResource.trim())}`,
  );
  return readIDs(response, 'role_ids');
};

export const replaceUserRoles = async (userID: string, serviceResource: string, roleIDs: string[]): Promise<void> => {
  await request.put<unknown>(
    `/v1/users/${encodeURIComponent(userID)}/roles?service_resource=${encodeURIComponent(serviceResource.trim())}`,
    { role_ids: roleIDs },
  );
};
