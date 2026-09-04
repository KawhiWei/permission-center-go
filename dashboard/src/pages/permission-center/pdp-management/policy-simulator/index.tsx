import { useState } from 'react';
import { Button, Card, Form, Input, Select, Tag } from 'tdesign-react';

import { decideAuthorization, type Decision, type ResourceType } from '../../../../api/pdp';
import { getRequestErrorMessage, PageHeader, useApplicationScope } from '../../shared';
import { methodOptions, notify, resourceTypeOptions } from '../shared';
import '../../style.less';
import '../style.less';

type SimulationForm = {
  subjectID: string;
  resourceCode: string;
  resourceType: ResourceType;
  resourceID: string;
  action: string;
  serviceCode: string;
  method: string;
  pathTemplate: string;
};

const EMPTY_FORM: SimulationForm = {
  subjectID: '',
  resourceCode: '',
  resourceType: 'entity',
  resourceID: '',
  action: '',
  serviceCode: '',
  method: 'GET',
  pathTemplate: '',
};

const PolicySimulatorPage = () => {
  const { application } = useApplicationScope();
  const [form, setForm] = useState<SimulationForm>({ ...EMPTY_FORM });
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<Decision | null>(null);

  const reset = () => {
    setForm({ ...EMPTY_FORM });
    setResult(null);
  };

  const simulate = async () => {
    const subjectID = form.subjectID.trim();
    const resourceCode = form.resourceCode.trim();
    const action = form.action.trim();
    if (!subjectID || !resourceCode || !action) {
      notify('warning', '请填写 subject、资源编码和动作');
      return;
    }
    setLoading(true);
    setResult(null);
    try {
      setResult(await decideAuthorization({
        application,
        subject_id: subjectID,
        resource_code: resourceCode,
        resource_type: form.resourceType,
        resource_id: form.resourceID.trim() || undefined,
        action,
        service_code: form.serviceCode.trim() || undefined,
        method: form.method || undefined,
        path_template: form.pathTemplate.trim() || undefined,
      }));
    } catch (error) {
      notify('error', getRequestErrorMessage(error, '策略模拟失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="permission-page permission-pdp-page">
      <PageHeader
        title="策略模拟"
        description="使用当前应用和指定主体验证一次业务授权决策，模拟请求不会写入业务数据。"
      />

      <Card className="permission-card permission-pdp-simulator-card" bordered>
        <div className="permission-pdp-simulation">
          <div className="permission-pdp-simulation-copy">
            <strong>验证一个主体是否能执行指定动作</strong>
            <span>PDP 会从当前应用读取主体角色；API 端点决策与实体实例决策可以分别模拟。</span>
          </div>
          <Form labelAlign="top" className="permission-pdp-simulation-form">
            <Form.FormItem label="subject ID">
              <Input value={form.subjectID} placeholder="例如 nexus-user-001" onChange={(value) => setForm((prev) => ({ ...prev, subjectID: value }))} />
            </Form.FormItem>
            <Form.FormItem label="资源编码">
              <Input value={form.resourceCode} placeholder="例如 post" onChange={(value) => setForm((prev) => ({ ...prev, resourceCode: value }))} />
            </Form.FormItem>
            <Form.FormItem label="资源类型">
              <Select value={form.resourceType} options={resourceTypeOptions} onChange={(value) => setForm((prev) => ({ ...prev, resourceType: String(value) as ResourceType }))} />
            </Form.FormItem>
            <Form.FormItem label="资源实例 ID" help="端点授权可留空；实体级授权建议填写业务实例 ID。">
              <Input value={form.resourceID} placeholder="例如 post-42" onChange={(value) => setForm((prev) => ({ ...prev, resourceID: value }))} />
            </Form.FormItem>
            <Form.FormItem label="动作编码">
              <Input value={form.action} placeholder="例如 update" onChange={(value) => setForm((prev) => ({ ...prev, action: value }))} />
            </Form.FormItem>
            <Form.FormItem label="服务编码">
              <Input value={form.serviceCode} placeholder="端点决策时填写，例如 forum-api" onChange={(value) => setForm((prev) => ({ ...prev, serviceCode: value }))} />
            </Form.FormItem>
            <Form.FormItem label="HTTP 方法">
              <Select value={form.method} options={methodOptions} onChange={(value) => setForm((prev) => ({ ...prev, method: String(value) }))} />
            </Form.FormItem>
            <Form.FormItem label="路径模板">
              <Input value={form.pathTemplate} placeholder="例如 /v1/posts/{postId}" onChange={(value) => setForm((prev) => ({ ...prev, pathTemplate: value }))} />
            </Form.FormItem>
          </Form>
          <div className="permission-pdp-simulation-actions">
            <Button theme="primary" type="button" loading={loading} onClick={() => void simulate()}>执行模拟</Button>
            <Button variant="outline" type="button" onClick={reset}>重置</Button>
          </div>
          {result ? (
            <div className={`permission-pdp-decision ${result.allow ? 'is-allow' : 'is-deny'}`}>
              <div className="permission-pdp-decision-head">
                <Tag theme={result.allow ? 'success' : 'danger'}>{result.allow ? '允许' : '拒绝'}</Tag>
                <strong>{result.reasonCode || (result.allow ? 'ALLOW' : 'DENY')}</strong>
              </div>
              <div className="permission-pdp-decision-meta">命中策略：{result.matchedPolicyIds.length ? result.matchedPolicyIds.join(', ') : '无'}</div>
            </div>
          ) : null}
        </div>
      </Card>
    </div>
  );
};

export default PolicySimulatorPage;
