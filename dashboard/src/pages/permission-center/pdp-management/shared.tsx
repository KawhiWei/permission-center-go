import { useEffect, useMemo, useState } from 'react';
import { Button, Drawer, Input, MessagePlugin, Select, Space, Tag } from 'tdesign-react';

import {
  getAuthorizationPolicyBindings,
  replaceAuthorizationPolicyBindings,
  type AuthorizationPolicy,
  type AuthorizationPolicyBinding,
  type EnforcementMode,
  type PolicyEffect,
  type ResourceType,
  type SubjectType,
} from '../../../api/pdp';
import { listRoles, type Role } from '../../../api/permission';
import { getRequestErrorMessage } from '../shared';

export type ResourceForm = {
  code: string;
  name: string;
  description: string;
  resourceType: ResourceType;
  matcher: string;
  enabled: boolean;
};

export type ActionForm = {
  code: string;
  name: string;
  description: string;
  enabled: boolean;
};

export type EndpointForm = {
  serviceCode: string;
  method: string;
  pathTemplate: string;
  resourceId: string;
  actionId: string;
  enforcementMode: EnforcementMode;
  enabled: boolean;
};

export type PolicyForm = {
  code: string;
  name: string;
  description: string;
  effect: PolicyEffect;
  priority: string;
  resourceCodes: string;
  actionCodes: string;
  enabled: boolean;
};

export type SubjectBindingDrawerProps = {
  serviceResource: string;
  policy: AuthorizationPolicy | null;
  visible: boolean;
  onClose: () => void;
};

export const EMPTY_RESOURCE_FORM: ResourceForm = {
  code: '',
  name: '',
  description: '',
  resourceType: 'entity',
  matcher: '',
  enabled: true,
};

export const EMPTY_ACTION_FORM: ActionForm = {
  code: '',
  name: '',
  description: '',
  enabled: true,
};

export const EMPTY_ENDPOINT_FORM: EndpointForm = {
  serviceCode: '',
  method: 'GET',
  pathTemplate: '',
  resourceId: '',
  actionId: '',
  enforcementMode: 'enforce',
  enabled: true,
};

export const EMPTY_POLICY_FORM: PolicyForm = {
  code: '',
  name: '',
  description: '',
  effect: 'allow',
  priority: '0',
  resourceCodes: '*',
  actionCodes: '*',
  enabled: true,
};

export const resourceTypeOptions = [
  { label: 'API 接口', value: 'api' },
  { label: '业务资源', value: 'entity' },
];

export const methodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'].map((method) => ({
  label: method,
  value: method,
}));

export const enforcementOptions = [
  { label: '强制拦截', value: 'enforce' },
  { label: '仅审计', value: 'audit' },
  { label: '停用', value: 'disabled' },
];

export const effectOptions = [
  { label: '允许', value: 'allow' },
  { label: '拒绝', value: 'deny' },
];

export const subjectTypeOptions = [
  { label: '角色', value: 'role' },
  { label: '用户 subject', value: 'subject' },
];

export const splitSelectors = (value: string): string[] => value
  .split(/[\s,，]+/)
  .map((item) => item.trim())
  .filter(Boolean);

export const joinSelectors = (values: string[]): string => values.join(', ');

export const resourceTypeLabel = (value: ResourceType): string => resourceTypeOptions.find((item) => item.value === value)?.label || value;
export const effectLabel = (value: PolicyEffect): string => value === 'deny' ? '拒绝' : '允许';
export const enforcementLabel = (value: EnforcementMode): string => enforcementOptions.find((item) => item.value === value)?.label || value;

type MessageType = 'success' | 'error' | 'warning';

// TDesign's legacy message plugin can throw before the React 19 adapter is loaded.
// Keep a failed notification from preventing the completed operation from closing
// its dialog and refreshing the current page.
export const notify = (type: MessageType, content: string) => {
  try {
    MessagePlugin[type](content);
  } catch {
    // The operation result remains authoritative when the optional toast host is unavailable.
  }
};

export const StatusTag = ({ enabled }: { enabled: boolean }) => (
  <Tag theme={enabled ? 'success' : 'default'} variant="light-outline">
    {enabled ? '启用' : '停用'}
  </Tag>
);

export const LoadingRow = ({ colSpan, loading, empty }: { colSpan: number; loading: boolean; empty: string }) => (
  <tr>
    <td colSpan={colSpan} className="permission-empty">{loading ? '正在加载...' : empty}</td>
  </tr>
);

export const SubjectBindingDrawer = ({ serviceResource, policy, visible, onClose }: SubjectBindingDrawerProps) => {
  const [bindingLoading, setBindingLoading] = useState(false);
  const [bindingSaving, setBindingSaving] = useState(false);
  const [bindingRows, setBindingRows] = useState<AuthorizationPolicyBinding[]>([]);
  const [bindingType, setBindingType] = useState<SubjectType>('role');
  const [bindingValue, setBindingValue] = useState('');
  const [roles, setRoles] = useState<Role[]>([]);

  const roleOptions = useMemo(
    () => roles.filter((role) => role.enabled).map((role) => ({ label: `${role.name || role.code} · ${role.id}`, value: role.id })),
    [roles],
  );

  const reset = () => {
    setBindingRows([]);
    setBindingValue('');
    setBindingType('role');
    setRoles([]);
  };

  const loadBindings = async () => {
    if (!policy) {
      return;
    }
    setBindingLoading(true);
    try {
      const [nextBindings, nextRoles] = await Promise.all([
        getAuthorizationPolicyBindings(policy.id),
        listRoles(serviceResource),
      ]);
      setBindingRows(nextBindings);
      setRoles(nextRoles);
    } catch (error) {
      setBindingRows([]);
      notify('error', getRequestErrorMessage(error, '加载策略绑定失败'));
    } finally {
      setBindingLoading(false);
    }
  };

  useEffect(() => {
    if (visible) {
      void loadBindings();
    }
    // A policy change means the drawer represents a new binding set.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [policy?.id, visible]);

  const handleVisibleChange = (nextVisible: boolean) => {
    if (nextVisible) return;
    if (!bindingSaving) {
      reset();
      onClose();
    }
  };

  const addBinding = () => {
    const value = bindingValue.trim();
    if (!value) {
      notify('warning', '请输入角色 ID 或用户 subject');
      return;
    }
    if (bindingRows.some((item) => item.subjectType === bindingType && item.subjectValue === value)) {
      notify('warning', '该主体已经绑定');
      return;
    }
    setBindingRows((prev) => [...prev, { subjectType: bindingType, subjectValue: value, enabled: true }]);
    setBindingValue('');
  };

  const saveBindings = async () => {
    if (!policy) {
      return;
    }
    setBindingSaving(true);
    try {
      await replaceAuthorizationPolicyBindings(policy.id, bindingRows);
      notify('success', '策略绑定已保存');
      reset();
      onClose();
    } catch (error) {
      notify('error', getRequestErrorMessage(error, '保存策略绑定失败'));
    } finally {
      setBindingSaving(false);
    }
  };

  return (
    <Drawer
      visible={visible}
      header={policy ? `策略绑定 · ${policy.name}` : '策略绑定'}
      size="560px"
      confirmBtn={{ content: '保存绑定', theme: 'primary', loading: bindingSaving || bindingLoading }}
      cancelBtn="取消"
      onConfirm={() => void saveBindings()}
      onCancel={() => handleVisibleChange(false)}
      onClose={() => handleVisibleChange(false)}
      destroyOnClose
    >
      <Space direction="vertical" size={14} style={{ width: '100%' }}>
        <div className="permission-form-help">角色绑定会随当前服务资源隔离；subject 绑定用于精确授权。保存会整体替换此策略的主体集合。</div>
        <div className="permission-pdp-binding-add">
          <Select value={bindingType} options={subjectTypeOptions} onChange={(value) => setBindingType(String(value) as SubjectType)} />
          {bindingType === 'role' && roleOptions.length > 0 ? (
            <Select value={bindingValue} options={roleOptions} filterable placeholder="选择角色或直接输入 ID" onChange={(value) => setBindingValue(String(value))} />
          ) : (
            <Input value={bindingValue} placeholder={bindingType === 'role' ? '输入角色 ID' : '输入 NexusAuth subject'} onChange={setBindingValue} onEnter={addBinding} />
          )}
          <Button variant="outline" type="button" onClick={addBinding}>添加</Button>
        </div>
        {bindingLoading ? <div className="permission-empty">正在加载绑定...</div> : bindingRows.length === 0 ? <div className="permission-empty">还没有绑定主体</div> : (
          <div className="permission-pdp-binding-list">
            {bindingRows.map((binding, index) => (
              <div className="permission-pdp-binding-row" key={`${binding.subjectType}-${binding.subjectValue}-${index}`}>
                <Tag theme={binding.subjectType === 'role' ? 'primary' : 'warning'} variant="light-outline">{binding.subjectType === 'role' ? '角色' : 'subject'}</Tag>
                <span className="permission-table-code" title={binding.subjectValue}>{binding.subjectValue}</span>
                <Button variant="text" theme="danger" type="button" onClick={() => setBindingRows((prev) => prev.filter((_, rowIndex) => rowIndex !== index))}>移除</Button>
              </div>
            ))}
          </div>
        )}
      </Space>
    </Drawer>
  );
};
