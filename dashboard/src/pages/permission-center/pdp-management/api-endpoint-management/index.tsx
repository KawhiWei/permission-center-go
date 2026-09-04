import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Dialog, Form, Input, Select, Space, Tag } from 'tdesign-react';

import {
  createAuthorizationAPIEndpoint,
  deleteAuthorizationAPIEndpoint,
  listAuthorizationActions,
  listAuthorizationAPIEndpoints,
  listAuthorizationResources,
  updateAuthorizationAPIEndpoint,
  type AuthorizationAction,
  type AuthorizationAPIEndpoint,
  type AuthorizationResource,
  type EnforcementMode,
} from '../../../../api/pdp';
import { formatDateTime, getRequestErrorMessage, PageHeader, useApplicationScope } from '../../shared';
import {
  EMPTY_ENDPOINT_FORM,
  enforcementLabel,
  enforcementOptions,
  LoadingRow,
  methodOptions,
  notify,
  StatusTag,
  type EndpointForm,
} from '../shared';
import '../../style.less';
import '../style.less';

const ApiEndpointManagementPage = () => {
  const { application } = useApplicationScope();
  const [endpoints, setEndpoints] = useState<AuthorizationAPIEndpoint[]>([]);
  const [resources, setResources] = useState<AuthorizationResource[]>([]);
  const [actions, setActions] = useState<AuthorizationAction[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogVisible, setDialogVisible] = useState(false);
  const [editingID, setEditingID] = useState('');
  const [form, setForm] = useState<EndpointForm>({ ...EMPTY_ENDPOINT_FORM });
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<AuthorizationAPIEndpoint | null>(null);
  const [deleting, setDeleting] = useState(false);

  const loadData = useCallback(async (scope: string) => {
    setLoading(true);
    try {
      const [nextEndpoints, nextResources, nextActions] = await Promise.all([
        listAuthorizationAPIEndpoints(scope),
        listAuthorizationResources(scope),
        listAuthorizationActions(scope),
      ]);
      setEndpoints(nextEndpoints);
      setResources(nextResources);
      setActions(nextActions);
    } catch (error) {
      setEndpoints([]);
      setResources([]);
      setActions([]);
      notify('error', getRequestErrorMessage(error, '加载 API 端点配置失败'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadData(application);
  }, [application, loadData]);

  const resourceOptions = useMemo(
    () => resources.filter((item) => item.enabled && item.resourceType === 'api').map((item) => ({ label: `${item.name || item.code} · ${item.code}`, value: item.id })),
    [resources],
  );
  const actionOptions = useMemo(
    () => actions.filter((item) => item.enabled).map((item) => ({ label: `${item.name || item.code} · ${item.code}`, value: item.id })),
    [actions],
  );

  const resetForm = () => {
    setEditingID('');
    setForm({ ...EMPTY_ENDPOINT_FORM });
  };

  const openCreateDialog = () => {
    resetForm();
    setForm({
      ...EMPTY_ENDPOINT_FORM,
      resourceId: resourceOptions[0]?.value || '',
      actionId: actionOptions[0]?.value || '',
    });
    setDialogVisible(true);
  };

  const openEditDialog = (item: AuthorizationAPIEndpoint) => {
    setEditingID(item.id);
    setForm({
      serviceCode: item.serviceCode,
      method: item.method || 'GET',
      pathTemplate: item.pathTemplate,
      resourceId: item.resourceId,
      actionId: item.actionId,
      enforcementMode: item.enforcementMode,
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
    const serviceCode = form.serviceCode.trim();
    const pathTemplate = form.pathTemplate.trim();
    if (!serviceCode || !pathTemplate || !form.resourceId || !form.actionId) {
      notify('warning', '请完整填写服务、路径、资源和动作');
      return;
    }
    setSaving(true);
    try {
      const payload = {
        service_code: serviceCode,
        method: form.method,
        path_template: pathTemplate,
        resource_id: form.resourceId,
        action_id: form.actionId,
        enforcement_mode: form.enforcementMode,
        enabled: form.enabled,
      };
      if (editingID) {
        await updateAuthorizationAPIEndpoint(editingID, payload);
        notify('success', 'API 端点已更新');
      } else {
        await createAuthorizationAPIEndpoint({ application, ...payload });
        notify('success', 'API 端点已创建');
      }
      setDialogVisible(false);
      resetForm();
      await loadData(application);
    } catch (error) {
      notify('error', getRequestErrorMessage(error, editingID ? '更新 API 端点失败' : '创建 API 端点失败'));
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
      await deleteAuthorizationAPIEndpoint(target.id);
      setDeleteTarget(null);
      notify('success', 'API 端点已删除');
      await loadData(application);
    } catch (error) {
      notify('error', getRequestErrorMessage(error, '删除 API 端点失败'));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="permission-page permission-pdp-page">
      <PageHeader
        title="API 端点管理"
        description="把业务服务路由绑定到 API 资源和动作，端点授权只判断接口能否被调用。"
        actions={(
          <Space>
            <Button variant="outline" loading={loading} type="button" onClick={() => void loadData(application)}>刷新</Button>
            <Button theme="primary" type="button" onClick={openCreateDialog}>新建端点</Button>
          </Space>
        )}
      />

      <div className="permission-toolbar">
        <span className="permission-toolbar-meta">当前应用共 {endpoints.length} 个 API 端点</span>
      </div>

      <Card className="permission-card" bordered>
        <div className="permission-table-wrap">
          <table className="permission-table permission-pdp-table">
            <thead><tr><th>服务与路由</th><th>资源</th><th>动作</th><th>执行模式</th><th>状态</th><th>更新时间</th><th>操作</th></tr></thead>
            <tbody>
              {loading ? <LoadingRow colSpan={7} loading empty="当前应用暂无 API 端点" /> : endpoints.length === 0 ? <LoadingRow colSpan={7} loading={false} empty="当前应用暂无 API 端点" /> : endpoints.map((item) => {
                const resource = resources.find((resourceItem) => resourceItem.id === item.resourceId);
                const action = actions.find((actionItem) => actionItem.id === item.actionId);
                return (
                  <tr key={item.id}>
                    <td><div className="permission-table-name"><Tag theme="default" variant="light-outline">{item.method}</Tag> {item.serviceCode || '-'}</div><div className="permission-table-code">{item.pathTemplate || '-'}</div></td>
                    <td><div>{resource?.name || '-'}</div><div className="permission-table-code">{resource?.code || item.resourceId}</div></td>
                    <td><div>{action?.name || '-'}</div><div className="permission-table-code">{action?.code || item.actionId}</div></td>
                    <td><Tag theme={item.enforcementMode === 'enforce' ? 'primary' : item.enforcementMode === 'audit' ? 'warning' : 'default'} variant="light-outline">{enforcementLabel(item.enforcementMode)}</Tag></td>
                    <td><StatusTag enabled={item.enabled} /></td>
                    <td className="permission-table-muted">{formatDateTime(item.updatedAt || item.createdAt)}</td>
                    <td><Space size="small"><Button variant="text" theme="primary" type="button" onClick={() => openEditDialog(item)}>编辑</Button><Button variant="text" theme="danger" type="button" onClick={() => setDeleteTarget(item)}>删除</Button></Space></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </Card>

      <Dialog
        visible={dialogVisible}
        header={editingID ? '编辑 API 端点' : '新建 API 端点'}
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
          <Form.FormItem label="服务编码">
            <Input value={form.serviceCode} maxlength={100} placeholder="例如 forum-api" onChange={(value) => setForm((prev) => ({ ...prev, serviceCode: value }))} />
          </Form.FormItem>
          <Form.FormItem label="HTTP 方法">
            <Select value={form.method} options={methodOptions} onChange={(value) => setForm((prev) => ({ ...prev, method: String(value) }))} />
          </Form.FormItem>
          <Form.FormItem label="路径模板" className="full-width" help="必须使用业务路由框架提供的标准模板，不要保存实际 path 参数。">
            <Input value={form.pathTemplate} placeholder="例如 /v1/posts/{postId}" onChange={(value) => setForm((prev) => ({ ...prev, pathTemplate: value }))} />
          </Form.FormItem>
          <Form.FormItem label="API 资源">
            <Select value={form.resourceId} options={resourceOptions} placeholder={resourceOptions.length ? '请选择资源' : '请先创建 API 资源'} filterable onChange={(value) => setForm((prev) => ({ ...prev, resourceId: String(value) }))} />
          </Form.FormItem>
          <Form.FormItem label="动作">
            <Select value={form.actionId} options={actionOptions} placeholder={actionOptions.length ? '请选择动作' : '请先创建动作'} filterable onChange={(value) => setForm((prev) => ({ ...prev, actionId: String(value) }))} />
          </Form.FormItem>
          <Form.FormItem label="执行模式">
            <Select value={form.enforcementMode} options={enforcementOptions} onChange={(value) => setForm((prev) => ({ ...prev, enforcementMode: String(value) as EnforcementMode }))} />
          </Form.FormItem>
          <Form.FormItem label="状态">
            <Select value={form.enabled ? 'enabled' : 'disabled'} options={[{ label: '启用', value: 'enabled' }, { label: '停用', value: 'disabled' }]} onChange={(value) => setForm((prev) => ({ ...prev, enabled: value === 'enabled' }))} />
          </Form.FormItem>
        </Form>
      </Dialog>

      <Dialog
        visible={Boolean(deleteTarget)}
        header="删除 API 端点"
        width={460}
        confirmBtn={{ content: '确认删除', theme: 'danger', loading: deleting }}
        cancelBtn="取消"
        confirmLoading={deleting}
        onConfirm={() => void confirmDelete()}
        onCancel={() => { if (!deleting) setDeleteTarget(null); }}
        onClose={() => { if (!deleting) setDeleteTarget(null); }}
        destroyOnClose
      >
        <p className="permission-delete-warning">确认删除端点 <strong>{deleteTarget ? `${deleteTarget.method} ${deleteTarget.pathTemplate}` : ''}</strong>？业务服务后续调用该路由将无法匹配此端点配置。</p>
      </Dialog>
    </div>
  );
};

export default ApiEndpointManagementPage;
