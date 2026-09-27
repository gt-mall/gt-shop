import { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Checkbox,
  ConfigProvider,
  Form,
  Input,
  message,
  Tooltip,
} from 'antd';
import {
  LockOutlined,
  SafetyCertificateOutlined,
  ShopOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';

import beianIcon from '@/assets/beian.png';
import { api, type VersionData } from '@/framework/http';
import { getDefaultAuthorizedPathFromPermissions } from '@/framework/router';
import {
  formatVersionSummary,
  useAppStore,
  useAuth,
  useShopSetting,
  useSiteSetting,
} from '@/framework/store';
import { adminThemeConfig } from '@/shared/admin-ui';

import './login.less';

type LoginForm = {
  username: string;
  password: string;
  remember: boolean;
};

// 依据《非经营性互联网信息服务备案管理办法》与全国公安机关互联网站安全管理服务平台规范配置官方查询链接
const MIIT_BEIAN_URL = 'https://beian.miit.gov.cn/';
const POLICE_BEIAN_BASE_URL = 'https://beian.mps.gov.cn/';
const POLICE_QUERY_BASE_URL = 'http://www.beian.gov.cn/portal/registerSystemInfo?recordcode=';

export default function LoginPage() {
  const { login, loading, refreshSettings } = useAuth();
  const navigate = useNavigate();
  const shopSetting = useShopSetting();
  const siteSetting = useSiteSetting();
  const [versionData, setVersionData] = useState<VersionData | null>(null);

  useEffect(() => {
    let active = true;
    const fetchPublicData = async () => {
      void refreshSettings();
      const res = await api.api.getSystemVersion();
      if (active && res.code === 1 && res.data) {
        setVersionData(res.data);
      }
    };
    void fetchPublicData();
    return () => {
      active = false;
    };
  }, [refreshSettings]);

  const submit = async (values: LoginForm) => {
    const result = await login(values);
    if (!result.success) {
      message.error(result.message || '登录失败');
      return;
    }
    const current = useAppStore.getState().current;
    const defaultPath = getDefaultAuthorizedPathFromPermissions(current?.permissions);
    navigate(defaultPath, { replace: true });
  };

  const { summary: verSummary, tooltip } = formatVersionSummary(
    versionData?.version,
    versionData?.build_time,
    versionData?.git_commit,
  );

  // 站点标题与版权
  const siteTitle = siteSetting?.title || shopSetting?.name || 'Mall 管理后台';
  const copyright = siteSetting?.copyright || `© ${dayjs().year()} Mall. All rights reserved.`;

  // 国家合规备案信息（工信部 ICP 备案与全国公安机关网安备案），以设置或环境变量为准，未填写时默认为空
  const icpNumber =
    siteSetting?.icp?.trim() ||
    (import.meta.env.VITE_APP_ICP as string | undefined)?.trim() ||
    '';

  const policeRecord =
    siteSetting?.police_record?.trim() ||
    (import.meta.env.VITE_APP_POLICE_RECORD as string | undefined)?.trim() ||
    '';

  // 根据公安备案号中的数字生成直达查询链接，无数字时回退到公安备案官网
  const policeUrl = useMemo(() => {
    if (!policeRecord) return '';
    const customCode = (import.meta.env.VITE_APP_POLICE_CODE as string | undefined)?.trim();
    if (customCode) {
      return `${POLICE_QUERY_BASE_URL}${customCode}`;
    }
    const digits = policeRecord.replace(/\D/g, '');
    if (digits.length >= 6) {
      return `${POLICE_QUERY_BASE_URL}${digits}`;
    }
    return POLICE_BEIAN_BASE_URL;
  }, [policeRecord]);

  return (
    <ConfigProvider theme={adminThemeConfig(false)}>
      <div className="mall-login-container">
        {/* 中间核心登录卡片 */}
        <main className="mall-login-main">
          <div className="mall-login-card">
            {/* 品牌标识与标题区 */}
            <div className="mall-login-header">
              {shopSetting?.logo ? (
                <img
                  src={shopSetting.logo}
                  alt={siteTitle}
                  className="w-12 h-12 object-contain mx-auto mb-4"
                />
              ) : (
                <div className="mall-login-logo">
                  <ShopOutlined style={{ fontSize: 24 }} />
                </div>
              )}
              <h1 className="mall-login-title">{siteTitle}</h1>
              <p className="mall-login-subtitle">电商运营与全渠道协同管理平台</p>
            </div>

            {/* 登录表单 */}
            <Form<LoginForm>
              layout="vertical"
              requiredMark={false}
              initialValues={{ remember: false }}
              onFinish={submit}
              className="mall-login-form"
            >
              <Form.Item
                name="username"
                label="管理员账号"
                rules={[{ required: true, message: '请输入管理员账号' }]}
              >
                <Input
                  size="large"
                  prefix={<UserOutlined style={{ color: '#94a3b8', marginRight: 6 }} />}
                  placeholder="请输入账号"
                  autoComplete="username"
                  allowClear
                />
              </Form.Item>

              <Form.Item
                name="password"
                label="登录密码"
                rules={[{ required: true, message: '请输入登录密码' }]}
              >
                <Input.Password
                  size="large"
                  prefix={<LockOutlined style={{ color: '#94a3b8', marginRight: 6 }} />}
                  placeholder="请输入密码"
                  autoComplete="current-password"
                />
              </Form.Item>

              <div className="mall-login-options">
                <Form.Item name="remember" valuePropName="checked" noStyle>
                  <Checkbox>记住登录 30 天</Checkbox>
                </Form.Item>
                <span className="mall-login-ssl-badge">
                  <SafetyCertificateOutlined style={{ color: '#10b981' }} />
                  安全加密传输
                </span>
              </div>

              <Button
                type="primary"
                htmlType="submit"
                size="large"
                loading={loading}
                block
                className="mall-login-submit-btn"
              >
                登 录
              </Button>
            </Form>

            {/* 安全与风控合规提示 */}
            <div className="mall-login-security-notice">
              <SafetyCertificateOutlined style={{ fontSize: 13 }} />
              <span>受网络安全等级保护 · 操作全程审计</span>
            </div>
          </div>
        </main>

        {/* 底部版权与国家合规备案信息（工信部 ICP 备案与全国公安机关网安备案） */}
        <footer className="mall-login-footer">
          <div className="footer-copyright">
            <span>{copyright}</span>
            {verSummary && (
              <>
                <span>·</span>
                {tooltip ? (
                  <Tooltip title={tooltip}>
                    <span className="version-tag">{verSummary}</span>
                  </Tooltip>
                ) : (
                  <span className="version-tag">{verSummary}</span>
                )}
              </>
            )}
          </div>

          {Boolean(icpNumber || policeRecord) && (
            <div className="footer-beian">
              {/* 工信部 ICP 备案：按照《非经营性互联网信息服务备案管理办法》链接至工信部备案系统 */}
              {icpNumber && (
                <a
                  href={MIIT_BEIAN_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                  title="工业和信息化部政务服务平台 ICP 备案查询"
                >
                  {icpNumber}
                </a>
              )}

              {Boolean(icpNumber && policeRecord) && (
                <span className="beian-divider">|</span>
              )}

              {/* 全国公安机关互联网站安全管理服务平台备案：按规定展示统一网安图标并链接至查询系统 */}
              {policeRecord && (
                <a
                  href={policeUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  title="全国公安机关互联网站安全管理服务平台备案查询"
                >
                  <img
                    src={beianIcon}
                    alt="全国公安机关网安备案图标"
                    className="police-icon"
                  />
                  <span>{policeRecord}</span>
                </a>
              )}
            </div>
          )}
        </footer>
      </div>
    </ConfigProvider>
  );
}
