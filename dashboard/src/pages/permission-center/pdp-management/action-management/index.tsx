import { useCallback, useEffect, useState } from 'react';
import { Button, Card, Dialog, Form, Input, Select, Space, Textarea } from 'tdesign-react';

import {
  createAuthorizationAction,
  deleteAuthorizationAction,
  listAuthorizationActions,
  updateAuthorizationAction,
  type AuthorizationAction,
} from '../../../../api/pdp';
import { formatDateTime, getRequestErrorMessage, PageHeader, useServiceResourceScope } from '../../shared';
import {
  EMPTY_ACTION_FORM,
  LoadingRow,
  notify,
  StatusTag,
  type ActionForm,
} from '../shared';
import '../../style.less';
import '../style.less';

const ActionManagementPage = () => {
  const { serviceResource } = useServiceResourceScope();
  const [actions, setActions] = useState<AuthorizationAction[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogVisible, setDialogVisible] = useState(false);
  const [editingID, setEditingID] = useState('');
  const [form, setForm] = useState<ActionForm>({ ...EMPTY_ACTION_FORM });
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<AuthorizationAction | null>(null);
  const [deleting, setDeleting] = useState(false);

  const loadActions = useCallback(async (scope: string) => {
    setLoading(true);
    try {
      setActions(await listAuthorizationActions(scope));
    } catch (error) {
      setActions([]);
      notify('error', getRequestErrorMessage(error, '加载动作列表失败'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadActions(serviceResource);
  }, [serviceResource, loadActions]);

  const resetForm = () => {
    setEditingID('');
    setForm({ ...EMPTY_ACTION_FORM });
  };

  const openCreateDialog = () => {
    resetForm();
    setDialogVisible(true);
  };

  const openEditDialog = (item: AuthorizationAction) => {
    setEditingID(item.id);
    setForm({
      code: item.code,
      name: item.name,
      description: item.description,
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
      notify('warning', '请填写动作编码和名称');
      return;
    }
    setSaving(true);
    try {
      if (editingID) {
        await updateAuthorizationAction(editingID, {
          name,
          description: form.description.trim(),
          enabled: form.enabled,
        });
        notify('success', '动作已更新');
      } else {
        await createAuthorizationAction({
          service_resource: serviceResource,
          code,
          name,
          description: form.description.trim(),
          enabled: form.enabled,
        });
        notify('success', '动作已创建');
      }
      setDialogVisible(false);
      resetForm();
      await loadActions(serviceResource);
    } catch (error) {
      notify('error', getRequestErrorMessage(error, editingID ? '更新动作失败' : '创建动作失败'));
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
      await deleteAuthorizationAction(target.id);
      setDeleteTarget(null);
      notify('success', '动作已删除');
      await loadActions(serviceResource);
    } catch (error) {
      notify('error', getRequestErrorMessage(error, '删除动作失败'));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="permission-page permission-pdp-page">
      <PageHeader
        title="动作管理"
        description="定义 read、create、update、delete、publish 等业务动作，策略使用动作编码进行匹配。"
        actions={(
          <Space>
            <Button variant="outline" loading={loading} type="button" onClick={() => void loadActions(serviceResource)}>刷新</Button>
            <Button theme="primary" type="button" onClick={openCreateDialog}>新建动作</Button>
          </Space>
        )}
      />

      <div className="permission-toolbar">
        <span className="permission-toolbar-meta">当前服务资源共 {actions.length} 个动作</span>
      </div>

      <Card className="permission-card" bordered>
        <div className="permission-table-wrap">
          <table className="permission-table permission-pdp-table">
            <thead><tr><th>动作编码</th><th>名称</th><th>描述</th><th>状态</th><th>更新时间</th><th>操作</th></tr></thead>
            <tbody>
              {loading ? <LoadingRow colSpan={6} loading empty="当前服务资源暂无动作" /> : actions.length === 0 ? <LoadingRow colSpan={6} loading={false} empty="当前服务资源暂无动作" /> : actions.map((item) => (
                <tr key={item.id}>
                  <td><div className="permission-table-name">{item.name || '-'}</div><div className="permission-table-code">{item.code || item.id}</div></td>
                  <td>{item.name || '-'}</td>
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
        header={editingID ? '编辑动作' : '新建动作'}
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
          <Form.FormItem label="动作编码" help="例如 read、create、update、delete、publish。">
            <Input value={form.code} disabled={Boolean(editingID)} maxlength={100} placeholder="例如 update" onChange={(value) => setForm((prev) => ({ ...prev, code: value }))} />
          </Form.FormItem>
          <Form.FormItem label="动作名称">
            <Input value={form.name} maxlength={100} placeholder="例如 编辑" onChange={(value) => setForm((prev) => ({ ...prev, name: value }))} />
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
        header="删除动作"
        width={460}
        confirmBtn={{ content: '确认删除', theme: 'danger', loading: deleting }}
        cancelBtn="取消"
        confirmLoading={deleting}
        onConfirm={() => void confirmDelete()}
        onCancel={() => { if (!deleting) setDeleteTarget(null); }}
        onClose={() => { if (!deleting) setDeleteTarget(null); }}
        destroyOnClose
      >
        <p className="permission-delete-warning">确认删除动作 <strong>{deleteTarget?.name || deleteTarget?.code}</strong>？仍被 API 端点或策略引用时，后端可能拒绝本次删除。</p>
      </Dialog>
    </div>
  );
};

export default ActionManagementPage;
