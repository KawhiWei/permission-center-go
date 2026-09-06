import { useCallback, useEffect, useMemo, useState } from 'react';
import { flushSync } from 'react-dom';
import { Button, Card, Dialog, Drawer, Form, Input, MessagePlugin, Select, Space, Tag, Textarea } from 'tdesign-react';

import {
  createAuthorizationPolicy,
  decidePDP,
  deleteAuthorizationPolicy,
  listAPIEndpoints,
  listAuthorizationPolicies,
  listRoles,
  publishAuthorizationPolicy,
  updateAuthorizationPolicy,
  type APIEndpoint,
  type AuthorizationPolicy,
  type PDPDecision,
  type PolicyAuthorizationType,
  type PolicyComparisonOperator,
  type PolicyEffect,
  type PolicyRequest,
  type Role,
} from '../../../api/permission';
import { PageHeader, getRequestErrorMessage, useServiceResourceScope } from '../shared';
import '../style.less';

type PolicyForm = { code: string; name: string; description: string; effect: PolicyEffect; authorizationType: PolicyAuthorizationType; priority: string; roleIDs: string[]; endpointIDs: string[]; attributePath: string; operator: PolicyComparisonOperator; literal: string };
const EMPTY_FORM: PolicyForm = { code: '', name: '', description: '', effect: 'allow', authorizationType: 'api', priority: '0', roleIDs: [], endpointIDs: [], attributePath: '', operator: 'eq', literal: '' };
const effectOptions = [{ label: '允许', value: 'allow' }, { label: '拒绝', value: 'deny' }];
const authorizationTypeOptions = [{ label: 'API 接口级', value: 'api' }, { label: '业务数据级', value: 'data' }];
const operatorOptions = [{ label: '等于', value: 'eq' }, { label: '不等于', value: 'neq' }, { label: '存在', value: 'exists' }];
const statusTheme = (status: AuthorizationPolicy['status']) => status === 'published' ? 'success' : status === 'draft' ? 'warning' : 'default';
const endpointLabel = (endpoint: APIEndpoint) => `${endpoint.method} ${endpoint.pathTemplate}${endpoint.enabled ? '' : '（停用）'}`;
const normalizeMultiSelectValues = (value: unknown): string[] => {
  const values = Array.isArray(value) ? value : [value];
  return [...new Set(values.filter((item): item is string | number => typeof item === 'string' || typeof item === 'number').map(String))];
};

const PolicyManagementPage = () => {
  const { serviceResource } = useServiceResourceScope();
  const [policies, setPolicies] = useState<AuthorizationPolicy[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [endpoints, setEndpoints] = useState<APIEndpoint[]>([]);
  const [loading, setLoading] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [editing, setEditing] = useState<AuthorizationPolicy | null>(null);
  const [form, setForm] = useState<PolicyForm>({ ...EMPTY_FORM });
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<AuthorizationPolicy | null>(null);
  const [simulator, setSimulator] = useState<{ subjectID: string; tenantID: string; endpointID: string; authorizationType: PolicyAuthorizationType }>({ subjectID: '', tenantID: '', endpointID: '', authorizationType: 'api' });
  const [decision, setDecision] = useState<PDPDecision | null>(null);
  const [deciding, setDeciding] = useState(false);

  const reload = useCallback(async () => {
    if (!serviceResource) return;
    setLoading(true);
    setPolicies([]); setRoles([]); setEndpoints([]);
    const [policyResult, roleResult, endpointResult] = await Promise.allSettled([listAuthorizationPolicies(serviceResource), listRoles(serviceResource), listAPIEndpoints(serviceResource)]);
    if (policyResult.status === 'fulfilled') setPolicies(policyResult.value); else MessagePlugin.error(getRequestErrorMessage(policyResult.reason, '加载 PDP 策略失败'));
    if (roleResult.status === 'fulfilled') setRoles(roleResult.value); else MessagePlugin.error(getRequestErrorMessage(roleResult.reason, '加载角色失败'));
    if (endpointResult.status === 'fulfilled') setEndpoints(endpointResult.value); else MessagePlugin.error(getRequestErrorMessage(endpointResult.reason, '加载 API 端点失败'));
    setLoading(false);
  }, [serviceResource]);
  useEffect(() => { void reload(); }, [reload]);
  const roleOptions = useMemo(() => roles.filter((role) => role.enabled).map((role) => ({ label: `${role.name} (${role.code})`, value: role.id })), [roles]);
  const endpointOptions = useMemo(() => endpoints.map((endpoint) => ({ label: endpointLabel(endpoint), value: endpoint.id })), [endpoints]);
  const enabledEndpointOptions = useMemo(() => endpoints.filter((endpoint) => endpoint.enabled).map((endpoint) => ({ label: endpointLabel(endpoint), value: endpoint.id })), [endpoints]);

  const openCreate = () => { setEditing(null); setForm({ ...EMPTY_FORM }); setDrawerVisible(true); };
  const openEdit = (policy: AuthorizationPolicy) => {
    const comparison = policy.condition?.comparison;
    flushSync(() => {
      setEditing(policy);
      setForm({ code: policy.code, name: policy.name, description: policy.description, effect: policy.effect, authorizationType: policy.authorizationType, priority: String(policy.priority), roleIDs: [...policy.roleIds], endpointIDs: [...policy.endpointIds], attributePath: comparison?.left.path || '', operator: comparison?.op || 'eq', literal: typeof comparison?.right?.value === 'string' ? comparison.right.value : '' });
    });
    setDrawerVisible(true);
  };
  const changeRoleSelection = (value: unknown) => setForm((current) => ({ ...current, roleIDs: normalizeMultiSelectValues(value) }));
  const changeEndpointSelection = (value: unknown) => setForm((current) => ({ ...current, endpointIDs: normalizeMultiSelectValues(value) }));
  const buildPayload = (): PolicyRequest | null => {
    if (!form.code.trim() || !form.name.trim() || form.roleIDs.length === 0 || form.endpointIDs.length === 0) { MessagePlugin.warning('请填写编码、名称，并选择角色和 API 端点'); return null; }
    const priority = Number(form.priority); const condition = form.attributePath.trim() ? { comparison: { left: { source: 'subject' as const, path: form.attributePath.trim(), type: 'string' as const }, op: form.operator, ...(form.operator === 'exists' ? {} : { right: { source: 'literal' as const, type: 'string' as const, value: form.literal } }) } } : null;
    return { ...(editing ? {} : { service_resource: serviceResource }), code: form.code.trim(), name: form.name.trim(), description: form.description.trim(), effect: form.effect, authorization_type: form.authorizationType, priority: Number.isFinite(priority) ? priority : 0, condition, role_ids: form.roleIDs, endpoint_ids: form.endpointIDs };
  };
  const save = async () => { const payload = buildPayload(); if (!payload) return; setSaving(true); try { const saved = editing ? await updateAuthorizationPolicy(editing.id, payload) : await createAuthorizationPolicy(payload); setPolicies((current) => editing ? current.map((policy) => policy.id === saved.id ? saved : policy) : [saved, ...current]); setDrawerVisible(false); MessagePlugin.success(editing ? '策略已保存为草稿' : '策略草稿已创建'); } catch (error) { MessagePlugin.error(getRequestErrorMessage(error, '保存策略失败')); } finally { setSaving(false); } };
  const publish = async (policy: AuthorizationPolicy) => { try { const saved = await publishAuthorizationPolicy(policy.id); setPolicies((current) => current.map((item) => item.id === saved.id ? saved : item)); MessagePlugin.success('策略已发布'); } catch (error) { MessagePlugin.error(getRequestErrorMessage(error, '发布策略失败')); } };
  const remove = async () => { if (!deleteTarget) return; try { await deleteAuthorizationPolicy(deleteTarget.id); setPolicies((current) => current.filter((policy) => policy.id !== deleteTarget.id)); setDeleteTarget(null); MessagePlugin.success('策略已归档'); } catch (error) { MessagePlugin.error(getRequestErrorMessage(error, '归档策略失败')); } };
  const simulate = async () => { if (!simulator.subjectID.trim() || !simulator.tenantID.trim() || !simulator.endpointID) { MessagePlugin.warning('请填写 Subject、租户并选择 API 端点'); return; } const endpoint = endpoints.find((item) => item.id === simulator.endpointID); if (!endpoint) { MessagePlugin.warning('请选择有效 API 端点'); return; } setDeciding(true); try { setDecision(await decidePDP({ authorization_type: simulator.authorizationType, service_resource: serviceResource, tenant_id: simulator.tenantID.trim(), subject: { id: simulator.subjectID.trim() }, action: { kind: 'http', method: endpoint.method }, resource: { kind: 'api_endpoint', endpoint_id: endpoint.id } })); } catch (error) { MessagePlugin.error(getRequestErrorMessage(error, '决策模拟失败')); } finally { setDeciding(false); } };

  return <div className="permission-page permission-policy-page">
    <PageHeader title="PDP 授权策略" description="角色与 API 端点通过已发布策略建立授权关系，菜单权限不参与决策" actions={<Button theme="primary" onClick={openCreate}>新建策略</Button>} />
    <div className="permission-metric-grid">
      <div className="permission-metric"><span className="permission-metric-label">已发布策略</span><strong className="permission-metric-value">{policies.filter((item) => item.status === 'published').length}</strong><span className="permission-metric-hint">运行时快照生效</span></div>
      <div className="permission-metric"><span className="permission-metric-label">草稿策略</span><strong className="permission-metric-value">{policies.filter((item) => item.status === 'draft').length}</strong><span className="permission-metric-hint">发布前可编辑</span></div>
      <div className="permission-metric"><span className="permission-metric-label">已登记 API</span><strong className="permission-metric-value">{endpoints.length}</strong><span className="permission-metric-hint">策略 target</span></div>
      <div className="permission-metric"><span className="permission-metric-label">可用角色</span><strong className="permission-metric-value">{roles.filter((role) => role.enabled).length}</strong><span className="permission-metric-hint">策略 subject</span></div>
    </div>
    <Card className="permission-card"><div className="permission-card-title"><strong>策略列表</strong><span>{loading ? '正在同步...' : `${policies.length} 条策略`}</span></div><div className="permission-table-wrap"><table className="permission-table"><thead><tr><th>策略</th><th>鉴权类型</th><th>效果</th><th>角色</th><th>API 端点</th><th>状态</th><th>版本</th><th>操作</th></tr></thead><tbody>{policies.length === 0 ? <tr><td colSpan={8} className="permission-empty">当前服务资源暂无 PDP 策略</td></tr> : policies.map((policy) => <tr key={policy.id}><td><div className="permission-table-name">{policy.name}</div><div className="permission-table-code">{policy.code}</div></td><td><Tag variant="light-outline">{policy.authorizationType === 'data' ? '数据级' : 'API 级'}</Tag></td><td><Tag theme={policy.effect === 'allow' ? 'success' : 'danger'} variant="light-outline">{policy.effect === 'allow' ? '允许' : '拒绝'}</Tag></td><td>{policy.roleIds.length}</td><td>{policy.endpointIds.length}</td><td><Tag theme={statusTheme(policy.status)} variant="light-outline">{policy.status}</Tag></td><td>v{policy.currentVersion || '-'}</td><td><Space size="small"><Button variant="text" onClick={() => openEdit(policy)} disabled={policy.status === 'published'}>编辑</Button><Button variant="text" theme="primary" onClick={() => void publish(policy)} disabled={policy.status === 'published'}>发布</Button><Button variant="text" theme="danger" onClick={() => setDeleteTarget(policy)}>归档</Button></Space></td></tr>)}</tbody></table></div></Card>
    <Card className="permission-card"><div className="permission-card-title"><strong>决策模拟</strong><span>后端从 subject 现有角色加载策略，不接受前端 role_ids</span></div><div className="permission-policy-simulator"><Select value={simulator.authorizationType} options={authorizationTypeOptions} onChange={(value) => setSimulator((current) => ({ ...current, authorizationType: value === 'data' ? 'data' : 'api' }))} /><Input value={simulator.subjectID} placeholder="NexusAuth subject" onChange={(value) => setSimulator((current) => ({ ...current, subjectID: value }))} /><Input value={simulator.tenantID} placeholder="tenant_id" onChange={(value) => setSimulator((current) => ({ ...current, tenantID: value }))} /><Select value={simulator.endpointID} options={enabledEndpointOptions} placeholder="选择已启用 API 端点" onChange={(value) => setSimulator((current) => ({ ...current, endpointID: String(value) }))} /><Button theme="primary" loading={deciding} onClick={() => void simulate()}>执行决策</Button></div>{decision ? <div className={`permission-decision permission-decision-${decision.decision}`}><strong>{decision.decision === 'allow' ? 'ALLOW' : 'DENY'}</strong><span>{decision.reasonCode || 'default_deny'} · 命中 {decision.matchedPolicyIds.length} 条策略 · 快照 v{decision.snapshotVersion || '-'}</span></div> : null}</Card>
    {drawerVisible ? (
      <Drawer
        key={editing?.id || 'create'}
        visible
        size="620px"
        header={editing ? '编辑策略草稿' : '新建策略草稿'}
        confirmBtn={{ content: editing ? '保存' : '创建', theme: 'primary', loading: saving }}
        cancelBtn="取消"
        onConfirm={() => void save()}
        onClose={() => setDrawerVisible(false)}
        onCancel={() => setDrawerVisible(false)}
      >
        <Form className="permission-drawer-form" labelAlign="top" initialData={form}>
          <Form.FormItem label="策略编码" name="code">
            <Input value={form.code} placeholder="例如 post-read" onChange={(value) => setForm((current) => ({ ...current, code: value }))} />
          </Form.FormItem>
          <Form.FormItem label="策略名称" name="name">
            <Input value={form.name} placeholder="例如 阅读帖子" onChange={(value) => setForm((current) => ({ ...current, name: value }))} />
          </Form.FormItem>
          <Form.FormItem label="效果" name="effect">
            <Select value={form.effect} options={effectOptions} onChange={(value) => setForm((current) => ({ ...current, effect: value === 'deny' ? 'deny' : 'allow' }))} />
          </Form.FormItem>
          <Form.FormItem label="鉴权类型" name="authorizationType">
            <Select value={form.authorizationType} options={authorizationTypeOptions} onChange={(value) => setForm((current) => ({ ...current, authorizationType: value === 'data' ? 'data' : 'api' }))} />
          </Form.FormItem>
          <Form.FormItem label="优先级" name="priority">
            <Input type="number" value={form.priority} onChange={(value) => setForm((current) => ({ ...current, priority: value }))} />
          </Form.FormItem>
          <Form.FormItem label="角色" name="roleIDs" help="至少选择一个已启用角色">
            <Select multiple valueType="value" value={form.roleIDs} options={roleOptions} filterable onChange={changeRoleSelection} />
          </Form.FormItem>
          <Form.FormItem label="API 端点" name="endpointIDs" help="至少选择一个已启用端点">
            <Select multiple valueType="value" value={form.endpointIDs} options={endpointOptions} filterable onChange={changeEndpointSelection} />
          </Form.FormItem>
          <Form.FormItem label="Subject 属性路径" name="attributePath">
            <Input value={form.attributePath} placeholder="可选，例如 department" onChange={(value) => setForm((current) => ({ ...current, attributePath: value }))} />
          </Form.FormItem>
          <Form.FormItem label="条件操作" name="operator">
            <Select value={form.operator} options={operatorOptions} onChange={(value) => setForm((current) => ({ ...current, operator: value as PolicyComparisonOperator }))} />
          </Form.FormItem>
          {form.operator !== 'exists' ? (
            <Form.FormItem label="条件值" name="literal">
              <Input value={form.literal} placeholder="字符串值" onChange={(value) => setForm((current) => ({ ...current, literal: value }))} />
            </Form.FormItem>
          ) : null}
          <Form.FormItem label="描述" name="description" className="full-width">
            <Textarea value={form.description} maxlength={500} onChange={(value) => setForm((current) => ({ ...current, description: value }))} />
          </Form.FormItem>
        </Form>
      </Drawer>
    ) : null}
    <Dialog visible={Boolean(deleteTarget)} header="归档策略" confirmBtn={{ content: '确认归档', theme: 'danger' }} onConfirm={() => void remove()} onClose={() => setDeleteTarget(null)} onCancel={() => setDeleteTarget(null)}>归档后策略不会参与 PDP 决策，历史决策日志保持可查。</Dialog>
  </div>;
};
export default PolicyManagementPage;
