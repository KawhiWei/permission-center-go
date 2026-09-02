import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Dialog, Form, Input, MessagePlugin, Select, Space, Tag, Textarea } from 'tdesign-react';

import {
  createApplication,
  deleteApplication,
  getStoredApplication,
  listApplications,
  setStoredApplication,
  updateApplication,
  type Application,
} from '../../../api/permission';
import {
  formatDateTime,
  getRequestErrorMessage,
  notifyApplicationCatalogChanged,
  PageHeader,
} from '../shared';
import '../style.less';

type ApplicationForm = {
  application: string;
  name: string;
  description: string;
  enabled: boolean;
};

const EMPTY_FORM: ApplicationForm = {
  application: '',
  name: '',
  description: '',
  enabled: true,
};

const statusOptions = [
  { label: '启用', value: 'enabled' },
  { label: '停用', value: 'disabled' },
];

const ApplicationManagementPage = () => {
  const [applications, setApplications] = useState<Application[]>([]);
  const [filter, setFilter] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dialogVisible, setDialogVisible] = useState(false);
  const [dialogMode, setDialogMode] = useState<'create' | 'edit'>('create');
  const [form, setForm] = useState<ApplicationForm>(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Application | null>(null);
  const [operationKey, setOperationKey] = useState('');

  const loadApplications = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setApplications(await listApplications());
    } catch (requestError) {
      setApplications([]);
      const message = getRequestErrorMessage(requestError, '加载应用列表失败');
      setError(message);
      MessagePlugin.error(message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadApplications();
  }, [loadApplications]);

  const filteredApplications = useMemo(() => {
    const keyword = filter.trim().toLocaleLowerCase();
    if (!keyword) {
      return applications;
    }
    return applications.filter((item) => (
      item.application.toLocaleLowerCase().includes(keyword)
      || item.name.toLocaleLowerCase().includes(keyword)
      || item.description.toLocaleLowerCase().includes(keyword)
    ));
  }, [applications, filter]);

  const openCreateDialog = () => {
    setDialogMode('create');
    setForm({ ...EMPTY_FORM });
    setDialogVisible(true);
  };

  const openEditDialog = (item: Application) => {
    setDialogMode('edit');
    setForm({
      application: item.application,
      name: item.name,
      description: item.description,
      enabled: item.enabled,
    });
    setDialogVisible(true);
  };

  const closeDialog = () => {
    if (!saving) {
      setDialogVisible(false);
      setForm({ ...EMPTY_FORM });
    }
  };

  const submitForm = async () => {
    const application = form.application.trim();
    const name = form.name.trim();
    if (!application || !name) {
      MessagePlugin.warning('请填写应用标识和应用名称');
      return;
    }

    setSaving(true);
    try {
      if (dialogMode === 'create') {
        await createApplication({ application, name, description: form.description.trim() });
        MessagePlugin.success('应用已创建');
      } else {
        await updateApplication(application, {
          name,
          description: form.description.trim(),
          enabled: form.enabled,
        });
        MessagePlugin.success('应用已更新');
      }
      setDialogVisible(false);
      setForm({ ...EMPTY_FORM });
      notifyApplicationCatalogChanged();
      await loadApplications();
    } catch (requestError) {
      MessagePlugin.error(getRequestErrorMessage(requestError, dialogMode === 'create' ? '创建应用失败' : '更新应用失败'));
    } finally {
      setSaving(false);
    }
  };

  const toggleApplication = async (item: Application) => {
    setOperationKey(`toggle:${item.application}`);
    try {
      await updateApplication(item.application, {
        name: item.name,
        description: item.description,
        enabled: !item.enabled,
      });
      if (item.enabled && item.application === getStoredApplication()) {
        setStoredApplication('');
      }
      MessagePlugin.success(item.enabled ? '应用已停用' : '应用已启用');
      notifyApplicationCatalogChanged();
      await loadApplications();
    } catch (requestError) {
      MessagePlugin.error(getRequestErrorMessage(requestError, item.enabled ? '停用应用失败' : '启用应用失败'));
    } finally {
      setOperationKey('');
    }
  };

  const confirmDelete = async () => {
    if (!deleteTarget) {
      return;
    }
    const target = deleteTarget;
    setDeleting(true);
    try {
      await deleteApplication(target.application);
      if (target.application === getStoredApplication()) {
        setStoredApplication('');
      }
      MessagePlugin.success('应用已删除');
      setDeleteTarget(null);
      notifyApplicationCatalogChanged();
      await loadApplications();
    } catch (requestError) {
      MessagePlugin.error(getRequestErrorMessage(requestError, '删除应用失败'));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="permission-page permission-application-page">
      <PageHeader
        title="应用管理"
        description="维护权限中心的应用目录，角色、菜单和用户角色绑定均按应用隔离。"
        actions={(
          <Space>
            <Button variant="outline" type="button" loading={loading} onClick={() => void loadApplications()}>刷新</Button>
            <Button theme="primary" type="button" onClick={openCreateDialog}>新建应用</Button>
          </Space>
        )}
      />

      <div className="permission-toolbar permission-application-toolbar">
        <Form layout="inline" className="permission-toolbar-form">
          <Form.FormItem label="搜索应用">
            <Input
              value={filter}
              placeholder="标识、名称或描述"
              clearable
              style={{ width: 280 }}
              onChange={setFilter}
            />
          </Form.FormItem>
        </Form>
        <span className="permission-toolbar-meta">共 {applications.length} 个应用，显示 {filteredApplications.length} 个</span>
      </div>

      <Card className="permission-card" bordered>
        {error ? (
          <div className="permission-load-error">
            <span>应用列表加载失败：{error}</span>
            <Button variant="text" theme="primary" type="button" onClick={() => void loadApplications()}>重试</Button>
          </div>
        ) : null}
        <div className="permission-table-wrap">
          <table className="permission-table permission-application-table">
            <thead>
              <tr>
                <th>应用标识</th>
                <th>应用名称</th>
                <th>描述</th>
                <th>状态</th>
                <th>更新时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr><td colSpan={6} className="permission-empty">正在加载应用...</td></tr>
              ) : filteredApplications.length === 0 ? (
                <tr><td colSpan={6} className="permission-empty">{applications.length === 0 ? '暂无应用，请先创建应用。' : '没有匹配的应用。'}</td></tr>
              ) : filteredApplications.map((item) => {
                const operationLoading = operationKey === `toggle:${item.application}`;
                return (
                  <tr key={item.application}>
                    <td>
                      <div className="permission-table-name">{item.application}</div>
                      <div className="permission-table-code">{item.createdByName || item.createdById || '-'}</div>
                    </td>
                    <td>{item.name || '-'}</td>
                    <td className="permission-table-muted">{item.description || '-'}</td>
                    <td><Tag theme={item.enabled ? 'success' : 'default'} variant="light-outline">{item.enabled ? '启用' : '停用'}</Tag></td>
                    <td className="permission-table-muted">{formatDateTime(item.updatedAt || item.createdAt)}</td>
                    <td>
                      <Space size="small">
                        <Button variant="text" theme="primary" type="button" onClick={() => openEditDialog(item)}>编辑</Button>
                        <Button variant="text" theme={item.enabled ? 'warning' : 'success'} type="button" loading={operationLoading} onClick={() => void toggleApplication(item)}>
                          {item.enabled ? '停用' : '启用'}
                        </Button>
                        <Button variant="text" theme="danger" type="button" disabled={operationLoading} onClick={() => setDeleteTarget(item)}>删除</Button>
                      </Space>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </Card>

      <Dialog
        visible={dialogVisible}
        header={dialogMode === 'create' ? '新建应用' : '编辑应用'}
        width={560}
        confirmBtn={{ content: dialogMode === 'create' ? '创建' : '保存', theme: 'primary', loading: saving }}
        cancelBtn="取消"
        confirmLoading={saving}
        onConfirm={() => void submitForm()}
        onCancel={closeDialog}
        onClose={closeDialog}
        destroyOnClose
      >
        <Form labelAlign="top" className="permission-application-form">
          <Form.FormItem label="应用标识" help="应用标识创建后不可修改，建议使用稳定的英文标识。">
            <Input
              value={form.application}
              placeholder="例如 content-platform"
              maxlength={80}
              disabled={dialogMode === 'edit'}
              onChange={(value) => setForm((prev) => ({ ...prev, application: value }))}
            />
          </Form.FormItem>
          <Form.FormItem label="应用名称">
            <Input
              value={form.name}
              placeholder="例如 内容平台"
              maxlength={100}
              onChange={(value) => setForm((prev) => ({ ...prev, name: value }))}
            />
          </Form.FormItem>
          <Form.FormItem label="描述">
            <Textarea
              value={form.description}
              placeholder="可选"
              maxlength={500}
              autosize={{ minRows: 3, maxRows: 6 }}
              onChange={(value) => setForm((prev) => ({ ...prev, description: value }))}
            />
          </Form.FormItem>
          {dialogMode === 'edit' ? (
            <Form.FormItem label="状态">
              <Select
                value={form.enabled ? 'enabled' : 'disabled'}
                options={statusOptions}
                onChange={(value) => setForm((prev) => ({ ...prev, enabled: value === 'enabled' }))}
              />
            </Form.FormItem>
          ) : null}
        </Form>
      </Dialog>

      <Dialog
        visible={Boolean(deleteTarget)}
        header="删除应用"
        width={460}
        confirmBtn={{ content: '确认删除', theme: 'danger', loading: deleting }}
        cancelBtn="取消"
        confirmLoading={deleting}
        onConfirm={() => void confirmDelete()}
        onCancel={() => {
          if (!deleting) {
            setDeleteTarget(null);
          }
        }}
        onClose={() => {
          if (!deleting) {
            setDeleteTarget(null);
          }
        }}
        destroyOnClose
      >
        <p className="permission-delete-warning">
          确认删除应用 <strong>{deleteTarget?.application}</strong>？应用下的角色、菜单及关联关系也将无法继续通过该应用访问，请确认后再操作。
        </p>
      </Dialog>
    </div>
  );
};

export default ApplicationManagementPage;
