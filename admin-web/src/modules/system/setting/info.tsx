import { useEffect, useState } from 'react';
import { Card, Descriptions, Spin, Typography } from 'antd';
import { api, type VersionData } from '@/framework/http';

export default function SystemInfoPage() {
  const [loading, setLoading] = useState(true);
  const [info, setInfo] = useState<VersionData | null>(null);

  useEffect(() => {
    let active = true;
    const load = async () => {
      setLoading(true);
      const result = await api.api.getSystemVersion();
      if (!active) return;
      setLoading(false);
      if (result.code === 1 && result.data) {
        setInfo(result.data);
      }
    };
    void load();
    return () => {
      active = false;
    };
  }, []);

  return (
    <Spin spinning={loading}>
      <Card title="系统信息" style={{ maxWidth: 800 }}>
        <Descriptions column={1} bordered size="middle">
          <Descriptions.Item label="系统版本">
            {info?.version ? `v${info.version}` : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="构建版本">
            {info?.build_time || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Git Commit">
            {info?.git_commit ? (
              <Typography.Text copyable code>
                {info.git_commit}
              </Typography.Text>
            ) : (
              '-'
            )}
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </Spin>
  );
}
