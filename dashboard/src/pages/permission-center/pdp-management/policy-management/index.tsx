import { useCallback, useEffect, useState } from 'react';
import { Button, Card, Dialog, Form, Input, Select, Space, Tag, Textarea } from 'tdesign-react';

import {
  createAuthorizationPolicy,
  deleteAuthorizationPolicy,
  listAuthorizationPolicies,
  updateAuthorizationPolicy,
  type AuthorizationPolicy,
  type PolicyEffect,
} from '../../../../api/pdp';
import { getRequestErrorMessage, PageHeader, useServiceResourceScope } from '../../shared';
import {
  effectLabel,
  effectOptions,
  EMPTY_POLICY_FORM,
  joinSelectors,
  LoadingRow,
  notify,
  splitSelectors,
  StatusTag,
  SubjectBindingDrawer,
  type PolicyForm,
} from '../shared';
import '../../style.less';
import '../style.less';

const PolicyManagementPage = () => {
  const { serviceResource } = useServiceResourceScope();
  const [policies, setPolicies] = useState<AuthorizationPolicy[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogVisible, setDialogVisible] = useState(false);
  const [editingID, setEditingID] = useState('');
  const [form, setForm] = useState<PolicyForm>({ ...EMPTY_POLICY_FORM });
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<AuthorizationPolicy | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [bindingPolicy, setBindingPolicy] = useState<AuthorizationPolicy | null>(null);

  const loadPolicies = useCallback(async (scope: string) => {
    setLoading(true);
    try {
      setPolicies(await listAuthorizationPolicies(scope));
    } catch (error) {
      setPolicies([]);
      notify('error', getRequestErrorMessage(error, '加载策略列表失败'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadPolicies(serviceResource);
  }, [serviceResource, loadPolicies]);

  const resetForm = () => {
    setEditingID('');
    setForm({ ...EMPTY_POLICY_FORM });
  };

  const openCreateDialog = () => {
    resetForm();
    setDialogVisible(true);
  };

  const openEditDialog = (item: AuthorizationPolicy) => {
    setEditingID(item.id);
    setForm({
      code: item.code,
      name: item.name,
      description: item.description,
      effect: item.effect,
      priority: String(item.priority),
      resourceCodes: joinSelectors(item.resourceCodes),
      actionCodes: joinSelectors(item.actionCodes),
      enabled: item.enabled,
    });
    setDialogVisible(true);
  };

  const closeDialog = () => {
    if (!saving) {
      setDialogVisible(false);
      resetForm();
    }
  };

  const submitForm = async () => {
    const code = form.code.trim();
    const name = form.name.trim();
    const priority = Number(form.priority);
    const resourceCodes = splitSelectors(form.resourceCodes);
    const actionCodes = splitSelectors(form.actionCodes);
    if (!code || !name || !resourceCodes.length || !actionCodes.length) {
      notify('warning', '请填写策略编码、名称、资源选择器和动作选择器');
      return;
    }
    if (!Number.isInteger(priority)) {
      notify('warning', '策略优先级必须是整数');
      return;
    }
    setSaving(true);
    try {
      const payload = {
        name,
        description: form.description.trim(),
        effect: form.effect,
        priority,
        resource_codes: resourceCodes,
        action_codes: actionCodes,
        enabled: form.enabled,
      };
      if (editingID) {
        await updateAuthorizationPolicy(editingID, payload);
        notify('success', '策略已更新');
      } else {
        await createAuthorizationPolicy({ service_resource: serviceResource, code, ...payload });
        notify('success', '策略已创建');
      }
      setDialogVisible(false);
      resetForm();
      await loadPolicies(serviceResource);
    } catch (error) {
      notify('error', getRequestErrorMessage(error, editingID ? '更新策略失败' : '创建策略失败'));
    } finally {
      setSaving(false);
    }
  };

  const confirmDelete = async () => {
    if (!deleteTarget) {
      return;
    }
    const target = deleteTarget;
    setDeleting(true);
    try {
      await deleteAuthorizationPolicy(target.id);
      setDeleteTarget(null);
      notify('success', '策略已删除');
      await loadPolicies(serviceResource);
    } catch (error) {
      notify('error', getRequestErrorMessage(error, '删除策略失败'));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="permission-page permission-pdp-page">
      <PageHeader
        title="策略管理"
        description="策略按优先级计算，命中任意拒绝策略时优先拒绝，未命中默认拒绝。"
        actions={(
          <Space>
            <Button variant="outline" loading={loading} type="button" onClick={() => void loadPolicies(serviceResource)}>刷新</Button>
            <Button theme="primary" type="button" onClick={openCreateDialog}>新建策略</Button>
          </Space>
        )}
      />

      <div className="permission-toolbar">
        <span className="permission-toolbar-meta">当前服务资源共 {policies.length} 个策略</span>
      </div>

      <Card className="permission-card" bordered>
        <div className="permission-table-wrap">
          <table className="permission-table permission-pdp-table">
            <thead><tr><th>策略</th><th>效果</th><th>优先级</th><th>资源选择器</th><th>动作选择器</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {loading ? <LoadingRow colSpan={7} loading empty="当前服务资源暂无策略" /> : policies.length === 0 ? <LoadingRow colSpan={7} loading={false} empty="当前服务资源暂无策略" /> : policies.map((item) => (
                <tr key={item.id}>
                  <td><div className="permission-table-name">{item.name || '-'}</div><div className="permission-table-code">{item.code || item.id}</div></td>
                  <td><Tag theme={item.effect === 'allow' ? 'success' : 'danger'} variant="light-outline">{effectLabel(item.effect)}</Tag></td>
                  <td className="permission-table-code">{item.priority}</td>
                  <td className="permission-table-code">{joinSelectors(item.resourceCodes) || '-'}</td>
                  <td className="permission-table-code">{joinSelectors(item.actionCodes) || '-'}</td>
                  <td><StatusTag enabled={item.enabled} /></td>
                  <td><Space size="small"><Button variant="text" theme="primary" type="button" onClick={() => openEditDialog(item)}>编辑</Button><Button variant="text" theme="primary" type="button" onClick={() => setBindingPolicy(item)}>绑定主体</Button><Button variant="text" theme="danger" type="button" onClick={() => setDeleteTarget(item)}>删除</Button></Space></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <Dialog
        visible={dialogVisible}
        header={editingID ? '编辑策略' : '新建策略'}
        width={700}
        confirmBtn={{ content: editingID ? '保存' : '创建', theme: 'primary', loading: saving }}
        cancelBtn="取消"
        confirmLoading={saving}
        onConfirm={() => void submitForm()}
        onCancel={closeDialog}
        onClose={closeDialog}
        destroyOnClose
      >
        <Form labelAlign="top" className="permission-pdp-dialog-form permission-pdp-dialog-grid">
          <Form.FormItem label="策略编码" help="服务资源内稳定唯一。">
            <Input value={form.code} disabled={Boolean(editingID)} maxlength={100} placeholder="例如 post-editor" onChange={(value) => setForm((prev) => ({ ...prev, code: value }))} />
          </Form.FormItem>
          <Form.FormItem label="策略名称">
            <Input value={form.name} maxlength={100} placeholder="例如 帖子编辑者" onChange={(value) => setForm((prev) => ({ ...prev, name: value }))} />
          </Form.FormItem>
          <Form.FormItem label="效果">
            <Select value={form.effect} options={effectOptions} onChange={(value) => setForm((prev) => ({ ...prev, effect: String(value) as PolicyEffect }))} />
          </Form.FormItem>
          <Form.FormItem label="优先级" help="数值越大越先计算；拒绝策略最终优先。">
            <Input type="number" value={form.priority} placeholder="0" onChange={(value) => setForm((prev) => ({ ...prev, priority: value }))} />
          </Form.FormItem>
          <Form.FormItem label="资源选择器" help="支持精确编码和 *，多个值使用逗号分隔。">
            <Input value={form.resourceCodes} placeholder="例如 post, post-api 或 *" onChange={(value) => setForm((prev) => ({ ...prev, resourceCodes: value }))} />
          </Form.FormItem>
          <Form.FormItem label="动作选择器" help="支持精确编码和 *，多个值使用逗号分隔。">
            <Input value={form.actionCodes} placeholder="例如 read, update 或 *" onChange={(value) => setForm((prev) => ({ ...prev, actionCodes: value }))} />
          </Form.FormItem>
          <Form.FormItem label="描述" className="full-width">
            <Textarea value={form.description} maxlength={500} placeholder="可选" autosize={{ minRows: 3, maxRows: 6 }} onChange={(value) => setForm((prev) => ({ ...prev, description: value }))} />
          </Form.FormItem>
          <Form.FormItem label="状态">
            <Select value={form.enabled ? 'enabled' : 'disabled'} options={[{ label: '启用', value: 'enabled' }, { label: '停用', value: 'disabled' }]} onChange={(value) => setForm((prev) => ({ ...prev, enabled: value === 'enabled' }))} />
          </Form.FormItem>
        </Form>
      </Dialog>

      <Dialog
        visible={Boolean(deleteTarget)}
        header="删除策略"
        width={460}
        confirmBtn={{ content: '确认删除', theme: 'danger', loading: deleting }}
        cancelBtn="取消"
        confirmLoading={deleting}
        onConfirm={() => void confirmDelete()}
        onCancel={() => { if (!deleting) setDeleteTarget(null); }}
        onClose={() => { if (!deleting) setDeleteTarget(null); }}
        destroyOnClose
      >
        <p className="permission-delete-warning">确认删除策略 <strong>{deleteTarget?.name || deleteTarget?.code}</strong>？删除后该策略及其主体绑定将不再参与 PDP 决策。</p>
      </Dialog>

      <SubjectBindingDrawer
        serviceResource={serviceResource}
        policy={bindingPolicy}
        visible={Boolean(bindingPolicy)}
        onClose={() => setBindingPolicy(null)}
      />
    </div>
  );
};

export default PolicyManagementPage;
