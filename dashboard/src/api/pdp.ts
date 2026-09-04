import request from './request';

// API entry points, business entities and rate-limit rules are evaluated by
// the PDP. Menu visibility remains in the existing RBAC model.
export type ResourceType = 'api' | 'entity';
export type PolicyEffect = 'allow' | 'deny';
export type SubjectType = 'role' | 'subject';
export type EnforcementMode = 'enforce' | 'audit' | 'disabled';

export type AuthorizationResource = {
  id: string;
  serviceResource: string;
  code: string;
  resourceType: ResourceType;
  name: string;
  description: string;
  enabled: boolean;
  matcher?: string;
  createdById?: string;
  createdByName?: string;
  createdAt?: string;
  updatedById?: string;
  updatedByName?: string;
  updatedAt?: string;
};

export type AuthorizationAction = {
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

export type AuthorizationAPIEndpoint = {
  id: string;
  serviceResource: string;
  serviceCode: string;
  method: string;
  pathTemplate: string;
  resourceId: string;
  actionId: string;
  enforcementMode: EnforcementMode;
  enabled: boolean;
  createdById?: string;
  createdByName?: string;
  createdAt?: string;
  updatedById?: string;
  updatedByName?: string;
  updatedAt?: string;
};

export type AuthorizationPolicy = {
  id: string;
  serviceResource: string;
  code: string;
  name: string;
  description: string;
  effect: PolicyEffect;
  priority: number;
  resourceCodes: string[];
  actionCodes: string[];
  enabled: boolean;
  createdById?: string;
  createdByName?: string;
  createdAt?: string;
  updatedById?: string;
  updatedByName?: string;
  updatedAt?: string;
};

export type AuthorizationPolicyBinding = {
  policyId?: string;
  subjectType: SubjectType;
  subjectValue: string;
  enabled: boolean;
  createdById?: string;
  createdByName?: string;
  createdAt?: string;
  updatedById?: string;
  updatedByName?: string;
  updatedAt?: string;
};

export type CreateAuthorizationResourceRequest = {
  service_resource: string;
  code: string;
  resource_type: ResourceType;
  name: string;
  description?: string;
  matcher?: string;
  enabled?: boolean;
};

export type UpdateAuthorizationResourceRequest = Omit<CreateAuthorizationResourceRequest, 'service_resource'>;

export type CreateAuthorizationActionRequest = {
  service_resource: string;
  code: string;
  name: string;
  description?: string;
  enabled?: boolean;
};

export type UpdateAuthorizationActionRequest = Omit<CreateAuthorizationActionRequest, 'service_resource' | 'code'>;

export type CreateAuthorizationAPIEndpointRequest = {
  service_resource: string;
  service_code: string;
  method: string;
  path_template: string;
  resource_id: string;
  action_id: string;
  enforcement_mode: EnforcementMode;
  enabled?: boolean;
};

export type UpdateAuthorizationAPIEndpointRequest = Omit<CreateAuthorizationAPIEndpointRequest, 'service_resource'>;

export type ImportSwaggerAPIEndpointsRequest = {
  service_resource: string;
  swagger_url: string;
};

export type ImportSwaggerAPIEndpointsResult = {
  total: number;
  created: number;
  skipped: number;
};

export type CreateAuthorizationPolicyRequest = {
  service_resource: string;
  code: string;
  name: string;
  description?: string;
  effect: PolicyEffect;
  priority: number;
  resource_codes: string[];
  action_codes: string[];
  enabled?: boolean;
};

export type UpdateAuthorizationPolicyRequest = Omit<CreateAuthorizationPolicyRequest, 'service_resource' | 'code'>;

export type ReplaceAuthorizationPolicyBindingsRequest = {
  bindings: Array<{
    subject_type: SubjectType;
    subject_value: string;
    enabled: boolean;
  }>;
};

export type DecisionRequest = {
  service_resource: string;
  subject_id: string;
  resource_code: string;
  resource_type: ResourceType;
  resource_id?: string;
  action: string;
  service_code?: string;
  method?: string;
  path_template?: string;
};

export type Decision = {
  allow: boolean;
  reasonCode: string;
  matchedPolicyIds: string[];
};

type RawRecord = Record<string, unknown>;

const asRecord = (value: unknown): RawRecord => (
  value && typeof value === 'object' ? value as RawRecord : {}
);

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
    if (typeof value === 'string' && value.trim() && Number.isFinite(Number(value))) {
      return Number(value);
    }
  }
  return 0;
};

const rawDate = (record: RawRecord, ...keys: string[]): string | undefined => {
  const value = rawString(record, ...keys);
  return value || undefined;
};

const rawStringArray = (record: RawRecord, ...keys: string[]): string[] => {
  for (const key of keys) {
    const value = record[key];
    if (Array.isArray(value)) {
      return value
        .map((item) => (typeof item === 'string' ? item : ''))
        .filter(Boolean);
    }
  }
  return [];
};

const readItems = (value: unknown): unknown[] => {
  if (Array.isArray(value)) {
    return value;
  }
  const record = asRecord(value);
  const items = record.items ?? record.Items ?? record.data ?? record.Data;
  return Array.isArray(items) ? items : [];
};

const readItem = (value: unknown): unknown => {
  const record = asRecord(value);
  return record.item ?? record.Item ?? record.data ?? record.Data ?? value;
};

const normalizeBase = (record: RawRecord) => ({
  createdById: rawString(record, 'created_by_id', 'createdById', 'CreatedByID') || undefined,
  createdByName: rawString(record, 'created_by_name', 'createdByName', 'CreatedByName') || undefined,
  createdAt: rawDate(record, 'created_at', 'createdAt', 'CreatedAt'),
  updatedById: rawString(record, 'updated_by_id', 'updatedById', 'UpdatedByID') || undefined,
  updatedByName: rawString(record, 'updated_by_name', 'updatedByName', 'UpdatedByName') || undefined,
  updatedAt: rawDate(record, 'updated_at', 'updatedAt', 'UpdatedAt'),
});

const normalizeResource = (value: unknown): AuthorizationResource => {
  const record = asRecord(value);
  const resourceType = rawString(record, 'resource_type', 'resourceType', 'type', 'Type');
  return {
    id: rawString(record, 'id', 'ID'),
    serviceResource: rawString(record, 'service_resource', 'serviceResource', 'ServiceResource'),
    code: rawString(record, 'code', 'Code'),
    resourceType: resourceType === 'entity' ? resourceType : 'api',
    name: rawString(record, 'name', 'Name'),
    description: rawString(record, 'description', 'Description'),
    enabled: rawBoolean(record, 'enabled', 'Enabled'),
    matcher: rawString(record, 'matcher', 'Matcher') || undefined,
    ...normalizeBase(record),
  };
};

const normalizeAction = (value: unknown): AuthorizationAction => {
  const record = asRecord(value);
  return {
    id: rawString(record, 'id', 'ID'),
    serviceResource: rawString(record, 'service_resource', 'serviceResource', 'ServiceResource'),
    code: rawString(record, 'code', 'Code'),
    name: rawString(record, 'name', 'Name'),
    description: rawString(record, 'description', 'Description'),
    enabled: rawBoolean(record, 'enabled', 'Enabled'),
    ...normalizeBase(record),
  };
};

const normalizeEndpoint = (value: unknown): AuthorizationAPIEndpoint => {
  const record = asRecord(value);
  const enforcementMode = rawString(record, 'enforcement_mode', 'enforcementMode', 'EnforcementMode');
  return {
    id: rawString(record, 'id', 'ID'),
    serviceResource: rawString(record, 'service_resource', 'serviceResource', 'ServiceResource'),
    serviceCode: rawString(record, 'service_code', 'serviceCode', 'ServiceCode'),
    method: rawString(record, 'method', 'Method').toUpperCase(),
    pathTemplate: rawString(record, 'path_template', 'pathTemplate', 'PathTemplate'),
    resourceId: rawString(record, 'resource_id', 'resourceId', 'ResourceID'),
    actionId: rawString(record, 'action_id', 'actionId', 'ActionID'),
    enforcementMode: enforcementMode === 'audit' || enforcementMode === 'disabled' ? enforcementMode : 'enforce',
    enabled: rawBoolean(record, 'enabled', 'Enabled'),
    ...normalizeBase(record),
  };
};

const normalizePolicy = (value: unknown): AuthorizationPolicy => {
  const record = asRecord(value);
  const effect = rawString(record, 'effect', 'Effect');
  return {
    id: rawString(record, 'id', 'ID'),
    serviceResource: rawString(record, 'service_resource', 'serviceResource', 'ServiceResource'),
    code: rawString(record, 'code', 'Code'),
    name: rawString(record, 'name', 'Name'),
    description: rawString(record, 'description', 'Description'),
    effect: effect === 'deny' ? 'deny' : 'allow',
    priority: rawNumber(record, 'priority', 'Priority'),
    resourceCodes: rawStringArray(record, 'resource_codes', 'resourceCodes', 'ResourceCodes'),
    actionCodes: rawStringArray(record, 'action_codes', 'actionCodes', 'ActionCodes'),
    enabled: rawBoolean(record, 'enabled', 'Enabled'),
    ...normalizeBase(record),
  };
};

const normalizeBinding = (value: unknown): AuthorizationPolicyBinding => {
  const record = asRecord(value);
  const subjectType = rawString(record, 'subject_type', 'subjectType', 'SubjectType');
  return {
    policyId: rawString(record, 'policy_id', 'policyId', 'PolicyID') || undefined,
    subjectType: subjectType === 'subject' ? 'subject' : 'role',
    subjectValue: rawString(record, 'subject_value', 'subjectValue', 'SubjectValue'),
    enabled: rawBoolean(record, 'enabled', 'Enabled'),
    ...normalizeBase(record),
  };
};

const normalizeDecision = (value: unknown): Decision => {
  const record = asRecord(readItem(value));
  const matched = record.matched_policy_ids ?? record.matchedPolicyIds ?? record.MatchedPolicyIDs;
  return {
    allow: rawBoolean(record, 'allow', 'Allow'),
    reasonCode: rawString(record, 'reason_code', 'reasonCode', 'ReasonCode'),
    matchedPolicyIds: Array.isArray(matched)
      ? matched.filter((item): item is string => typeof item === 'string')
      : [],
  };
};

const scopeQuery = (serviceResource: string) => `?service_resource=${encodeURIComponent(serviceResource.trim())}`;

export const listAuthorizationResources = async (serviceResource: string): Promise<AuthorizationResource[]> => {
  const response = await request.get<unknown>(`/v1/authorization/resources${scopeQuery(serviceResource)}`);
  return readItems(response).map(normalizeResource);
};

export const createAuthorizationResource = async (payload: CreateAuthorizationResourceRequest): Promise<AuthorizationResource> => {
  const response = await request.post<unknown>('/v1/authorization/resources', payload);
  return normalizeResource(readItem(response));
};

export const updateAuthorizationResource = async (id: string, payload: UpdateAuthorizationResourceRequest): Promise<AuthorizationResource> => {
  const response = await request.put<unknown>(`/v1/authorization/resources/${encodeURIComponent(id)}`, payload);
  return normalizeResource(readItem(response));
};

export const deleteAuthorizationResource = async (id: string): Promise<void> => {
  await request.delete<unknown>(`/v1/authorization/resources/${encodeURIComponent(id)}`);
};

export const listAuthorizationActions = async (serviceResource: string): Promise<AuthorizationAction[]> => {
  const response = await request.get<unknown>(`/v1/authorization/actions${scopeQuery(serviceResource)}`);
  return readItems(response).map(normalizeAction);
};

export const createAuthorizationAction = async (payload: CreateAuthorizationActionRequest): Promise<AuthorizationAction> => {
  const response = await request.post<unknown>('/v1/authorization/actions', payload);
  return normalizeAction(readItem(response));
};

export const updateAuthorizationAction = async (id: string, payload: UpdateAuthorizationActionRequest): Promise<AuthorizationAction> => {
  const response = await request.put<unknown>(`/v1/authorization/actions/${encodeURIComponent(id)}`, payload);
  return normalizeAction(readItem(response));
};

export const deleteAuthorizationAction = async (id: string): Promise<void> => {
  await request.delete<unknown>(`/v1/authorization/actions/${encodeURIComponent(id)}`);
};

export const listAuthorizationAPIEndpoints = async (serviceResource: string): Promise<AuthorizationAPIEndpoint[]> => {
  const response = await request.get<unknown>(`/v1/authorization/api-endpoints${scopeQuery(serviceResource)}`);
  return readItems(response).map(normalizeEndpoint);
};

export const createAuthorizationAPIEndpoint = async (payload: CreateAuthorizationAPIEndpointRequest): Promise<AuthorizationAPIEndpoint> => {
  const response = await request.post<unknown>('/v1/authorization/api-endpoints', payload);
  return normalizeEndpoint(readItem(response));
};

export const updateAuthorizationAPIEndpoint = async (id: string, payload: UpdateAuthorizationAPIEndpointRequest): Promise<AuthorizationAPIEndpoint> => {
  const response = await request.put<unknown>(`/v1/authorization/api-endpoints/${encodeURIComponent(id)}`, payload);
  return normalizeEndpoint(readItem(response));
};

export const deleteAuthorizationAPIEndpoint = async (id: string): Promise<void> => {
  await request.delete<unknown>(`/v1/authorization/api-endpoints/${encodeURIComponent(id)}`);
};

export const importSwaggerAPIEndpoints = async (payload: ImportSwaggerAPIEndpointsRequest): Promise<ImportSwaggerAPIEndpointsResult> => {
  const response = await request.post<unknown>('/v1/authorization/api-endpoints/import-swagger', payload);
  const item = asRecord(readItem(response));
  return {
    total: rawNumber(item, 'total', 'Total'),
    created: rawNumber(item, 'created', 'Created'),
    skipped: rawNumber(item, 'skipped', 'Skipped'),
  };
};

export const listAuthorizationPolicies = async (serviceResource: string): Promise<AuthorizationPolicy[]> => {
  const response = await request.get<unknown>(`/v1/authorization/policies${scopeQuery(serviceResource)}`);
  return readItems(response).map(normalizePolicy);
};

export const createAuthorizationPolicy = async (payload: CreateAuthorizationPolicyRequest): Promise<AuthorizationPolicy> => {
  const response = await request.post<unknown>('/v1/authorization/policies', payload);
  return normalizePolicy(readItem(response));
};

export const updateAuthorizationPolicy = async (id: string, payload: UpdateAuthorizationPolicyRequest): Promise<AuthorizationPolicy> => {
  const response = await request.put<unknown>(`/v1/authorization/policies/${encodeURIComponent(id)}`, payload);
  return normalizePolicy(readItem(response));
};

export const deleteAuthorizationPolicy = async (id: string): Promise<void> => {
  await request.delete<unknown>(`/v1/authorization/policies/${encodeURIComponent(id)}`);
};

export const getAuthorizationPolicyBindings = async (policyID: string): Promise<AuthorizationPolicyBinding[]> => {
  const response = await request.get<unknown>(`/v1/authorization/policies/${encodeURIComponent(policyID)}/bindings`);
  return readItems(response).map(normalizeBinding);
};

export const replaceAuthorizationPolicyBindings = async (
  policyID: string,
  bindings: AuthorizationPolicyBinding[],
): Promise<void> => {
  await request.put<unknown>(`/v1/authorization/policies/${encodeURIComponent(policyID)}/bindings`, {
    bindings: bindings.map((binding) => ({
      subject_type: binding.subjectType,
      subject_value: binding.subjectValue,
      enabled: binding.enabled,
    })),
  } satisfies ReplaceAuthorizationPolicyBindingsRequest);
};

export const decideAuthorization = async (payload: DecisionRequest): Promise<Decision> => {
  const response = await request.post<unknown>('/v1/pdp/decisions', payload);
  return normalizeDecision(response);
};
