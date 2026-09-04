import { useCallback, useEffect, useState } from 'react';
import { Button, Card, Dialog, Form, Input, Select, Space, Tag, Textarea } from 'tdesign-react';

import {
  createAuthorizationResource,
  deleteAuthorizationResource,
  listAuthorizationResources,
  updateAuthorizationResource,
  type AuthorizationResource,
  type ResourceType,
} from '../../../../api/pdp';
import { formatDateTime, getRequestErrorMessage, PageHeader, useServiceResourceScope } from '../../shared';
import {
  EMPTY_RESOURCE_FORM,
  LoadingRow,
  notify,
  resourceTypeLabel,
  resourceTypeOptions,
  StatusTag,
  type ResourceForm,
} from '../shared';
import '../../style.less';
import '../style.less';

const ResourceManagementPage = () => {
  const { serviceResource } = useServiceResourceScope();
  const [resources, setResources] = useState<AuthorizationResource[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogVisible, setDialogVisible] = useState(false);
  const [editingID, setEditingID] = useState('');
  const [form, setForm] = useState<ResourceForm>({ ...EMPTY_RESOURCE_FORM });
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<AuthorizationResource | null>(null);
  const [deleting, setDeleting] = useState(false);

  const loadResources = useCallback(async (scope: string) => {
    setLoading(true);
    try {
      setResources(await listAuthorizationResources(scope));
    } catch (error) {
      setResources([]);
      notify('error', getRequestErrorMessage(error, '加载资源列表失败'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadResources(serviceResource);
  }, [serviceResource, loadResources]);

  const resetForm = () => {
    setEditingID('');
    setForm({ ...EMPTY_RESOURCE_FORM });
  };

  const openCreateDialog = () => {
    resetForm();
    setDialogVisible(true);
  };

  const openEditDialog = (item: AuthorizationResource) => {
    setEditingID(item.id);
    setForm({
      code: item.code,
      name: item.name,
      description: item.description,
      resourceType: item.resourceType,
      matcher: item.matcher || '',
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
    if (!code || !name) {
      notify('warning', '请填写资源编码和名称');
      return;
    }
    setSaving(true);
    try {
      if (editingID) {
        await updateAuthorizationResource(editingID, {
          code,
          resource_type: form.resourceType,
          name,
          description: form.description.trim(),
          matcher: form.matcher.trim(),
          enabled: form.enabled,
        });
        notify('success', '资源已更新');
      } else {
        await createAuthorizationResource({
          service_resource: serviceResource,
          code,
          resource_type: form.resourceType,
          name,
          description: form.description.trim(),
          matcher: form.matcher.trim(),
          enabled: form.enabled,
        });
        notify('success', '资源已创建');
      }
      setDialogVisible(false);
      resetForm();
      await loadResources(serviceResource);
    } catch (error) {
      notify('error', getRequestErrorMessage(error, editingID ? '更新资源失败' : '创建资源失败'));
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
      await deleteAuthorizationResource(target.id);
      setDeleteTarget(null);
      notify('success', '资源已删除');
      await loadResources(serviceResource);
    } catch (error) {
      notify('error', getRequestErrorMessage(error, '删除资源失败'));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="permission-page permission-pdp-page">
      <PageHeader
        title="资源管理"
        description="注册 API 接口或业务实体资源，策略通过资源编码匹配，业务实例数据仍由业务服务维护。"
        actions={(
          <Space>
            <Button variant="outline" loading={loading} type="button" onClick={() => void loadResources(serviceResource)}>刷新</Button>
            <Button theme="primary" type="button" onClick={openCreateDialog}>新建资源</Button>
          </Space>
        )}
      />

      <div className="permission-toolbar">
        <span className="permission-toolbar-meta">当前服务资源共 {resources.length} 个资源</span>
      </div>

      <Card className="permission-card" bordered>
        <div className="permission-table-wrap">
          <table className="permission-table permission-pdp-table">
            <thead><tr><th>资源</th><th>类型</th><th>匹配器</th><th>描述</th><th>状态</th><th>更新时间</th><th>操作</th></tr></thead>
            <tbody>
              {loading ? <LoadingRow colSpan={7} loading empty="当前服务资源暂无资源" /> : resources.length === 0 ? <LoadingRow colSpan={7} loading={false} empty="当前服务资源暂无资源" /> : resources.map((item) => (
                <tr key={item.id}>
                  <td><div className="permission-table-name">{item.name || '-'}</div><div className="permission-table-code">{item.code || item.id}</div></td>
                  <td><Tag theme="primary" variant="light-outline">{resourceTypeLabel(item.resourceType)}</Tag></td>
                  <td className="permission-table-code">{item.matcher || '-'}</td>
                  <td className="permission-table-muted">{item.description || '-'}</td>
                  <td><StatusTag enabled={item.enabled} /></td>
                  <td className="permission-table-muted">{formatDateTime(item.updatedAt || item.createdAt)}</td>
                  <td><Space size="small"><Button variant="text" theme="primary" type="button" onClick={() => openEditDialog(item)}>编辑</Button><Button variant="text" theme="danger" type="button" onClick={() => setDeleteTarget(item)}>删除</Button></Space></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <Dialog
        visible={dialogVisible}
        header={editingID ? '编辑资源' : '新建资源'}
        width={560}
        confirmBtn={{ content: editingID ? '保存' : '创建', theme: 'primary', loading: saving }}
        cancelBtn="取消"
        confirmLoading={saving}
        onConfirm={() => void submitForm()}
        onCancel={closeDialog}
        onClose={closeDialog}
        destroyOnClose
      >
        <Form labelAlign="top" className="permission-pdp-dialog-form">
          <Form.FormItem label="资源类型">
            <Select value={form.resourceType} options={resourceTypeOptions} onChange={(value) => setForm((prev) => ({ ...prev, resourceType: String(value) as ResourceType }))} />
          </Form.FormItem>
          <Form.FormItem label="资源编码" help="服务资源内稳定唯一，策略通过该编码匹配。">
            <Input value={form.code} disabled={Boolean(editingID)} maxlength={100} placeholder="例如 post" onChange={(value) => setForm((prev) => ({ ...prev, code: value }))} />
          </Form.FormItem>
          <Form.FormItem label="资源名称">
            <Input value={form.name} maxlength={100} placeholder="例如 帖子" onChange={(value) => setForm((prev) => ({ ...prev, name: value }))} />
          </Form.FormItem>
          <Form.FormItem label="匹配器" help="API 资源可填写标准路径模板；实体资源通常留空。">
            <Input value={form.matcher} placeholder="例如 /v1/posts/{postId}" onChange={(value) => setForm((prev) => ({ ...prev, matcher: value }))} />
          </Form.FormItem>
          <Form.FormItem label="描述">
            <Textarea value={form.description} maxlength={500} placeholder="可选" autosize={{ minRows: 3, maxRows: 6 }} onChange={(value) => setForm((prev) => ({ ...prev, description: value }))} />
          </Form.FormItem>
          <Form.FormItem label="状态">
            <Select value={form.enabled ? 'enabled' : 'disabled'} options={[{ label: '启用', value: 'enabled' }, { label: '停用', value: 'disabled' }]} onChange={(value) => setForm((prev) => ({ ...prev, enabled: value === 'enabled' }))} />
          </Form.FormItem>
        </Form>
      </Dialog>

      <Dialog
        visible={Boolean(deleteTarget)}
        header="删除资源"
        width={460}
        confirmBtn={{ content: '确认删除', theme: 'danger', loading: deleting }}
        cancelBtn="取消"
        confirmLoading={deleting}
        onConfirm={() => void confirmDelete()}
        onCancel={() => { if (!deleting) setDeleteTarget(null); }}
        onClose={() => { if (!deleting) setDeleteTarget(null); }}
        destroyOnClose
      >
        <p className="permission-delete-warning">确认删除资源 <strong>{deleteTarget?.name || deleteTarget?.code}</strong>？已经被端点或策略引用时，后端可能拒绝本次删除。</p>
      </Dialog>
    </div>
  );
};

export default ResourceManagementPage;
