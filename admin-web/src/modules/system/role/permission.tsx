import { useCallback, useEffect, useMemo, useState, type CSSProperties } from "react";
import {
  AppstoreOutlined,
  ArrowLeftOutlined,
  CheckOutlined,
  DownOutlined,
  InfoCircleOutlined,
  RightOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
  SettingOutlined,
  UpOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Avatar,
  Badge,
  Button,
  Card,
  Checkbox,
  Empty,
  Input,
  message,
  Space,
  Spin,
  Tag,
  Typography,
  theme,
} from "antd";
import { useNavigate, useParams } from "react-router-dom";

import { api } from "@/framework/http";
import type { PermissionDefinition, RoleRow } from "@/framework/http/generated_api";
import { formatTime } from "@/shared/utils/format";

import "./permission.less";

type PermissionItem = {
  code: string;
  name: string;
  groupKey: string;
  groupName: string;
  moduleKey: string;
};

type PermissionGroup = {
  key: string;
  name: string;
  permissions: PermissionItem[];
};

type PermissionModule = {
  key: string;
  name: string;
  groups: PermissionGroup[];
};

const moduleNames: Record<string, string> = {
  system: "系统管理",
};

function normalizePermission(item: PermissionDefinition): PermissionItem | undefined {
  if (!item.code) return;
  const [moduleKey, featureKey] = item.code.split(":");
  if (!moduleKey || !featureKey) return;
  return {
    code: item.code,
    name: item.name || item.code,
    groupKey: `${moduleKey}:${featureKey}`,
    groupName: item.module || featureKey,
    moduleKey,
  };
}

function buildPermissionModules(permissions: PermissionItem[]): PermissionModule[] {
  const modules = new Map<string, PermissionModule>();

  permissions.forEach((permission) => {
    let moduleItem = modules.get(permission.moduleKey);
    if (!moduleItem) {
      moduleItem = {
        key: permission.moduleKey,
        name: moduleNames[permission.moduleKey] || permission.moduleKey,
        groups: [],
      };
      modules.set(permission.moduleKey, moduleItem);
    }

    let group = moduleItem.groups.find((item) => item.key === permission.groupKey);
    if (!group) {
      group = {
        key: permission.groupKey,
        name: permission.groupName,
        permissions: [],
      };
      moduleItem.groups.push(group);
    }
    group.permissions.push(permission);
  });

  return Array.from(modules.values());
}

export default function RolePermissionPage() {
  const { token } = theme.useToken();
  const navigate = useNavigate();
  const { roleId } = useParams<{ roleId: string }>();
  const numericRoleId = Number(roleId);
  const [role, setRole] = useState<RoleRow>();
  const [permissions, setPermissions] = useState<PermissionItem[]>([]);
  const [checkedPermissions, setCheckedPermissions] = useState<string[]>([]);
  const [activeModuleKey, setActiveModuleKey] = useState("");
  const [expandedGroupKeys, setExpandedGroupKeys] = useState<string[]>([]);
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState("");

  const permissionModules = useMemo(() => buildPermissionModules(permissions), [permissions]);
  const checkedPermissionSet = useMemo(() => new Set(checkedPermissions), [checkedPermissions]);
  const activeModule = useMemo(
    () => permissionModules.find((item) => item.key === activeModuleKey) ?? permissionModules[0],
    [activeModuleKey, permissionModules],
  );
  const visibleGroups = useMemo(() => {
    const searchValue = keyword.trim().toLowerCase();
    if (!activeModule || !searchValue) return activeModule?.groups ?? [];
    return activeModule.groups.flatMap((group) => {
      const matchedPermissions = group.permissions.filter(
        (permission) =>
          permission.name.toLowerCase().includes(searchValue) ||
          permission.code.toLowerCase().includes(searchValue) ||
          group.name.toLowerCase().includes(searchValue),
      );
      return matchedPermissions.length > 0 ? [{ ...group, permissions: matchedPermissions }] : [];
    });
  }, [activeModule, keyword]);
  const isSuperAdmin = role?.code === "super_admin";
  const allPermissionCodes = useMemo(() => permissions.map((item) => item.code), [permissions]);
  const allSelected =
    allPermissionCodes.length > 0 && allPermissionCodes.every((code) => checkedPermissionSet.has(code));
  const activeGroupKeys = activeModule?.groups.map((item) => item.key) ?? [];
  const allExpanded =
    activeGroupKeys.length > 0 && activeGroupKeys.every((key) => expandedGroupKeys.includes(key));

  const pageStyle = {
    "--permission-bg": token.colorBgContainer,
    "--permission-fill": token.colorFillAlter,
    "--permission-border": token.colorBorderSecondary,
    "--permission-text": token.colorText,
    "--permission-text-secondary": token.colorTextSecondary,
    "--permission-primary": token.colorPrimary,
    "--permission-primary-bg": token.colorPrimaryBg,
    "--permission-shadow": token.boxShadowTertiary,
  } as CSSProperties;

  const loadAuthorization = useCallback(async () => {
    if (!Number.isSafeInteger(numericRoleId) || numericRoleId <= 0) {
      setLoadError("角色参数无效");
      setLoading(false);
      return;
    }

    setLoading(true);
    setLoadError("");
    const [roleResult, definitionResult, assignedResult] = await Promise.all([
      api.api.getSystemRole({ id: numericRoleId }),
      api.api.listAllPermissions(),
      api.api.getSystemRolePermissions({ id: numericRoleId }),
    ]);
    setLoading(false);

    const failedResult = [roleResult, definitionResult, assignedResult].find((result) => result.code !== 1);
    if (failedResult || !roleResult.data) {
      const errorMessage = failedResult?.message || "获取角色授权信息失败";
      setLoadError(errorMessage);
      message.error(errorMessage);
      return;
    }

    const normalizedPermissions = (definitionResult.data ?? []).flatMap((item) => {
      const permission = normalizePermission(item);
      return permission ? [permission] : [];
    });
    const modules = buildPermissionModules(normalizedPermissions);
    const assignedPermissions =
      roleResult.data.code === "super_admin"
        ? normalizedPermissions.map((item) => item.code)
        : assignedResult.data ?? [];

    setRole(roleResult.data);
    setPermissions(normalizedPermissions);
    setCheckedPermissions(assignedPermissions);
    setActiveModuleKey(modules[0]?.key ?? "");
    setExpandedGroupKeys(modules[0]?.groups.map((item) => item.key) ?? []);
  }, [numericRoleId]);

  useEffect(() => {
    void loadAuthorization();
  }, [loadAuthorization]);

  const backToRoleList = () => navigate("/system/role");

  const selectModule = (moduleItem: PermissionModule) => {
    setActiveModuleKey(moduleItem.key);
    setExpandedGroupKeys(moduleItem.groups.map((item) => item.key));
  };

  const togglePermission = (permissionCode: string, checked: boolean) => {
    setCheckedPermissions((current) => {
      const next = new Set(current);
      if (checked) next.add(permissionCode);
      else next.delete(permissionCode);
      return Array.from(next);
    });
  };

  const toggleGroup = (group: PermissionGroup, checked: boolean) => {
    setCheckedPermissions((current) => {
      const next = new Set(current);
      group.permissions.forEach((permission) => {
        if (checked) next.add(permission.code);
        else next.delete(permission.code);
      });
      return Array.from(next);
    });
  };

  const toggleAllPermissions = () => {
    setCheckedPermissions(allSelected ? [] : allPermissionCodes);
  };

  const toggleAllGroups = () => {
    if (!activeModule) return;
    setExpandedGroupKeys(allExpanded ? [] : activeModule.groups.map((item) => item.key));
  };

  const toggleGroupExpanded = (groupKey: string) => {
    setExpandedGroupKeys((current) =>
      current.includes(groupKey) ? current.filter((key) => key !== groupKey) : [...current, groupKey],
    );
  };

  const savePermissions = async () => {
    if (!role?.id || isSuperAdmin) return;
    setSaving(true);
    const result = await api.api.saveSystemRolePermissions({
      id: role.id,
      permissions: checkedPermissions,
    });
    setSaving(false);
    if (result.code !== 1) {
      message.error(result.message || "保存角色权限失败");
      return;
    }
    message.success("角色权限已保存");
  };

  if (loading && !role) {
    return (
      <div className="role-permission-loading" style={pageStyle}>
        <Spin size="large" description="正在加载角色权限" />
      </div>
    );
  }

  if (!role) {
    return (
      <Card className="role-permission-error" style={pageStyle}>
        <Empty description={loadError || "未找到角色授权信息"}>
          <Space>
            <Button onClick={backToRoleList}>返回角色列表</Button>
            <Button type="primary" onClick={loadAuthorization}>
              重新加载
            </Button>
          </Space>
        </Empty>
      </Card>
    );
  }

  return (
    <div className="role-permission-page" style={pageStyle}>
      <header className="role-permission-header">
        <div>
          <Typography.Title level={2}>角色授权</Typography.Title>
          <Typography.Text type="secondary">为角色分配菜单与操作权限</Typography.Text>
        </div>
        <Space wrap>
          <Button icon={<ArrowLeftOutlined />} onClick={backToRoleList}>
            返回角色列表
          </Button>
          <Button
            type="primary"
            icon={<CheckOutlined />}
            loading={saving}
            disabled={isSuperAdmin}
            onClick={savePermissions}
          >
            保存授权
          </Button>
        </Space>
      </header>

      <Card className="role-summary-card">
        <div className="role-summary-grid">
          <div className="role-summary-identity">
            <Avatar size={48} shape="square" icon={<SafetyCertificateOutlined />} />
            <div>
              <span className="role-summary-label">角色名称</span>
              <strong>{role.name || "-"}</strong>
            </div>
          </div>
          <div className="role-summary-item">
            <span className="role-summary-label">角色编码</span>
            <strong>{role.code || "-"}</strong>
          </div>
          <div className="role-summary-item">
            <span className="role-summary-label">状态</span>
            <Badge status={role.status === 1 ? "success" : "default"} text={role.status === 1 ? "启用" : "禁用"} />
          </div>
          <div className="role-summary-item role-summary-remark">
            <span className="role-summary-label">备注</span>
            <strong>{role.remark || "-"}</strong>
          </div>
          <div className="role-summary-item">
            <span className="role-summary-label">更新时间</span>
            <strong>{formatTime(role.updated_at)}</strong>
          </div>
          <div className="role-summary-item">
            <span className="role-summary-label">已选权限</span>
            <strong>{checkedPermissions.length} 个</strong>
          </div>
        </div>
      </Card>

      {isSuperAdmin ? (
        <Alert
          showIcon
          type="info"
          message="super_admin 自动拥有全部权限，无需写入或保存 Casbin 策略。"
        />
      ) : null}

      <Card className="permission-workbench-card">
        <div className="permission-workbench">
          <aside className="permission-module-panel">
            <div className="permission-panel-title">
              <span>权限模块</span>
              <span>{permissionModules.length}/{permissionModules.length}</span>
            </div>
            <div className="permission-module-list">
              {permissionModules.map((moduleItem) => {
                const moduleCodes = moduleItem.groups.flatMap((group) =>
                  group.permissions.map((permission) => permission.code),
                );
                const selectedCount = moduleCodes.filter((code) => checkedPermissionSet.has(code)).length;
                const active = moduleItem.key === activeModule?.key;
                return (
                  <button
                    key={moduleItem.key}
                    type="button"
                    className={`permission-module-item${active ? " is-active" : ""}`}
                    onClick={() => selectModule(moduleItem)}
                  >
                    <span className="permission-module-icon">
                      {moduleItem.key === "system" ? <SettingOutlined /> : <AppstoreOutlined />}
                    </span>
                    <span className="permission-module-name">{moduleItem.name}</span>
                    <span className="permission-module-count">{selectedCount}/{moduleCodes.length}</span>
                    <RightOutlined className="permission-module-arrow" />
                  </button>
                );
              })}
            </div>
          </aside>

          <section className="permission-content-panel">
            <div className="permission-content-heading">
              <div>
                <Typography.Title level={4}>{activeModule?.name || "权限配置"}</Typography.Title>
                <Typography.Text type="secondary">选择该角色可以访问的功能和操作</Typography.Text>
              </div>
              <Space wrap>
                <Button onClick={toggleAllGroups}>{allExpanded ? "收起全部" : "展开全部"}</Button>
                <Button disabled={isSuperAdmin || permissions.length === 0} onClick={toggleAllPermissions}>
                  {allSelected ? "取消全选" : "全部选择"}
                </Button>
                <Tag color="blue" className="permission-selected-tag">已选 {checkedPermissions.length}</Tag>
              </Space>
            </div>

            <Input
              allowClear
              value={keyword}
              prefix={<SearchOutlined />}
              placeholder="搜索权限名称 / 权限码"
              className="permission-search"
              onChange={(event) => setKeyword(event.target.value)}
            />

            <div className="permission-groups">
              {visibleGroups.length > 0 ? (
                visibleGroups.map((group) => {
                  const groupCodes = group.permissions.map((permission) => permission.code);
                  const selectedCount = groupCodes.filter((code) => checkedPermissionSet.has(code)).length;
                  const groupChecked = groupCodes.length > 0 && selectedCount === groupCodes.length;
                  const groupExpanded = expandedGroupKeys.includes(group.key);
                  return (
                    <div key={group.key} className="permission-group">
                      <div className="permission-group-header">
                        <Checkbox
                          checked={groupChecked}
                          indeterminate={selectedCount > 0 && !groupChecked}
                          disabled={isSuperAdmin}
                          onChange={(event) => toggleGroup(group, event.target.checked)}
                        >
                          {group.name}
                        </Checkbox>
                        <div className="permission-group-meta">
                          <span>{selectedCount}/{groupCodes.length}</span>
                          <Button
                            type="text"
                            size="small"
                            aria-label={groupExpanded ? `收起${group.name}` : `展开${group.name}`}
                            icon={groupExpanded ? <UpOutlined /> : <DownOutlined />}
                            onClick={() => toggleGroupExpanded(group.key)}
                          />
                        </div>
                      </div>
                      {groupExpanded ? (
                        <div className="permission-item-grid">
                          {group.permissions.map((permission) => (
                            <Checkbox
                              key={permission.code}
                              checked={checkedPermissionSet.has(permission.code)}
                              disabled={isSuperAdmin}
                              onChange={(event) => togglePermission(permission.code, event.target.checked)}
                            >
                              <span className="permission-item-label">
                                <span>{permission.name}</span>
                                <code>{permission.code}</code>
                              </span>
                            </Checkbox>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  );
                })
              ) : (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="没有匹配的权限" />
              )}
            </div>
          </section>
        </div>
      </Card>

      <div className="permission-footer-bar">
        <span><InfoCircleOutlined /> 修改授权后，用户刷新页面或重新登录即可生效</span>
        <Space>
          <Button onClick={backToRoleList}>取消</Button>
          <Button
            type="primary"
            icon={<CheckOutlined />}
            loading={saving}
            disabled={isSuperAdmin}
            onClick={savePermissions}
          >
            保存授权
          </Button>
        </Space>
      </div>
    </div>
  );
}
