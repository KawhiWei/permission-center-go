import { useCallback, useEffect, useState } from 'react';
import { Button, Card, Checkbox, Input, MessagePlugin, Space, Tag } from 'tdesign-react';

import { getUserRoleIDs, listRoles, replaceUserRoles, type Role } from '../../../api/permission';
import { getRequestErrorMessage, PageHeader, useApplicationScope } from '../shared';
import '../style.less';

const UserRoleManagementPage = () => {
  const { application } = useApplicationScope();
  const [userID, setUserID] = useState('');
  const [loadedUserID, setLoadedUserID] = useState('');
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedRoleIDs, setSelectedRoleIDs] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const loadRoles = useCallback(async (scope: string) => {
    try {
      setRoles(await listRoles(scope));
    } catch (error) {
      setRoles([]);
      MessagePlugin.error(getRequestErrorMessage(error, '加载角色失败'));
    }
  }, []);

  useEffect(() => {
    setLoadedUserID('');
    setSelectedRoleIDs([]);
    void loadRoles(application);
  }, [application, loadRoles]);

  const loadUserRoles = async (subject: string) => {
    setLoading(true);
    try {
      const [nextRoles, assignedRoleIDs] = await Promise.all([
        listRoles(application),
        getUserRoleIDs(subject, application),
      ]);
      setRoles(nextRoles);
      setSelectedRoleIDs(assignedRoleIDs);
      setLoadedUserID(subject);
    } catch (error) {
      setLoadedUserID('');
      setSelectedRoleIDs([]);
      MessagePlugin.error(getRequestErrorMessage(error, '加载用户角色失败'));
    } finally {
      setLoading(false);
    }
  };

  const queryUserRoles = async () => {
    const nextUserID = userID.trim();
    if (!nextUserID) {
      MessagePlugin.warning('请输入统一登录用户的 subject');
      return;
    }

    await loadUserRoles(nextUserID);
  };

  const saveUserRoles = async () => {
    if (!loadedUserID) {
      MessagePlugin.warning('请先查询用户角色');
      return;
    }

    setSaving(true);
    try {
      await replaceUserRoles(loadedUserID, application, selectedRoleIDs);
      const [nextRoles, assignedRoleIDs] = await Promise.all([
        listRoles(application),
        getUserRoleIDs(loadedUserID, application),
      ]);
      setRoles(nextRoles);
      setSelectedRoleIDs(assignedRoleIDs);
      MessagePlugin.success('用户角色绑定已保存');
    } catch (error) {
      MessagePlugin.error(getRequestErrorMessage(error, '保存用户角色失败'));
    } finally {
      setSaving(false);
    }
  };

  const knownRoleIDs = new Set(roles.map((role) => role.id));
  const unknownAssignedRoleIDs = selectedRoleIDs.filter((roleID) => !knownRoleIDs.has(roleID));

  return (
    <div className="permission-page permission-user-role-page">
      <PageHeader
        title="用户角色绑定"
        description="使用统一登录用户的 subject，维护其在当前应用内的角色集合"
        actions={loadedUserID ? <Tag theme="success" variant="light-outline">已加载用户</Tag> : null}
      />

      <div className="permission-toolbar permission-user-role-toolbar">
        <Space className="permission-user-query">
          <Input
            value={userID}
            placeholder="输入 NexusAuth subject"
            clearable
            style={{ width: 300 }}
            onChange={setUserID}
            onEnter={() => void queryUserRoles()}
          />
          <Button theme="primary" type="button" loading={loading} onClick={() => void queryUserRoles()}>查询角色</Button>
        </Space>
      </div>

      <Card className="permission-card" bordered>
        <div className="permission-card-title">
          <strong>{loadedUserID ? '角色列表' : '查询用户角色'}</strong>
          <span>{loadedUserID ? `用户：${loadedUserID}` : '查询后可编辑绑定关系'}</span>
        </div>
        {loadedUserID ? (
          roles.length === 0 ? (
            <div className="permission-empty">当前应用暂无可用角色，请先创建角色。</div>
          ) : (
            <>
              <Checkbox.Group
                value={selectedRoleIDs}
                onChange={(value) => setSelectedRoleIDs(value.map(String))}
              >
                <div className="permission-role-list">
                  {roles.map((role) => (
                    <div className="permission-role-option" key={role.id}>
                      <Checkbox value={role.id} />
                      <span className="permission-role-option-copy">
                        <strong title={role.name}>{role.name || '-'}</strong>
                        <span title={role.code}>{role.code || role.id}</span>
                      </span>
                    </div>
                  ))}
                </div>
              </Checkbox.Group>
              {unknownAssignedRoleIDs.length > 0 ? (
                <div className="permission-assigned-unknown">
                  当前返回了 {unknownAssignedRoleIDs.length} 个不在启用角色列表中的关联 ID，保存时会保留这些 ID：{unknownAssignedRoleIDs.join(', ')}
                </div>
              ) : null}
              <div className="permission-dialog-footer">
                <Button theme="primary" type="button" loading={saving} onClick={() => void saveUserRoles()}>保存绑定</Button>
              </div>
            </>
          )
        ) : (
          <div className="permission-empty">输入用户 subject 后查询当前应用的角色绑定。</div>
        )}
      </Card>
    </div>
  );
};

export default UserRoleManagementPage;
