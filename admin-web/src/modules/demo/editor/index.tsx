import { useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  Row,
  Select,
  Space,
  Tabs,
  Typography,
} from 'antd';
import {
  CodeOutlined,
  EyeOutlined,
  FileTextOutlined,
  ReloadOutlined,
  SendOutlined,
} from '@ant-design/icons';
import { RichEditor } from '@/shared/components';

type DemoFormData = {
  title: string;
  category: string;
  content: string;
};

const presetContent = `<h2>🌟 夏季新款透气亚麻休闲衬衫</h2>
<p><span style="color: #4338CA; font-weight: bold;">【限时特惠】</span> 采用 100% 法国进口天然亚麻纤维，<mark data-color="#FEF08A" style="background-color: #FEF08A;">手感柔软亲肤，透气吸汗</mark>，带来极致自然的穿着体验。</p>
<p><img src="https://images.unsplash.com/photo-1596755094514-f87e34085b2c?auto=format&fit=crop&w=600&q=80" alt="夏季休闲衬衫" data-width="50%" data-alignment="center" style="width: 50%; max-width: 100%; height: auto; display: block; margin: 0 auto;" /></p>
<blockquote>
  <p>“简约而不简单，让每一次穿着都成为一场与自然的对话。” —— 首席设计师语录</p>
</blockquote>
<h3>✨ 核心工艺与设计亮点</h3>
<ul>
  <li><strong>自然透气</strong>：立体微孔纤维结构，盛夏体感温度立减 2~3℃。</li>
  <li><strong>环保免烫</strong>：微皱肌理感，面料克重达 185g/m<sup>2</sup>，自然下垂无需费心打理。</li>
  <li><strong>精致细节</strong>：精磨天然贝壳扣，双车线加固领口，经久耐穿不变形。</li>
</ul>
<hr>
<h3>📏 详细规格与尺码对照表</h3>
<table>
  <thead>
    <tr>
      <th>尺码标号</th>
      <th>衣长 (cm)</th>
      <th>胸围 (cm)</th>
      <th>肩宽 (cm)</th>
      <th>推荐身高/体重</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>M</strong></td>
      <td>70</td>
      <td>104</td>
      <td>45</td>
      <td>165-172cm / 50-65kg</td>
    </tr>
    <tr>
      <td><strong>L</strong></td>
      <td>72</td>
      <td>108</td>
      <td>46.5</td>
      <td>170-178cm / 65-75kg</td>
    </tr>
    <tr>
      <td><strong>XL</strong></td>
      <td>74</td>
      <td>112</td>
      <td>48</td>
      <td>175-185cm / 75-88kg</td>
    </tr>
  </tbody>
</table>
<hr>
<h3>📋 品质与服务保障清单</h3>
<ul data-type="taskList">
  <li data-type="taskItem" data-checked="true"><label><input type="checkbox" checked="checked"><span></span></label><div>100% 法国进口天然亚麻纤维权威质检认证</div></li>
  <li data-type="taskItem" data-checked="true"><label><input type="checkbox" checked="checked"><span></span></label><div>环保活性染料染色，甲醛未检出，色牢度达 4 级</div></li>
  <li data-type="taskItem" data-checked="false"><label><input type="checkbox"><span></span></label><div>全国顺丰极速包邮，支持 7 天无理由放心退换</div></li>
</ul>
<hr>
<h3>🎬 产品动态实拍展示</h3>
<div class="video-player-container" style="width: 100%; max-width: 100%; margin: 0 auto;"><video controls="true" playsinline="true" style="width: 100%; height: auto; border-radius: 8px; display: block;" src="https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4"></video></div>
<p>如有尺码或面料疑问，欢迎随时点击右上角客服咨询！</p>`;

const usageCode = `import { Form } from 'antd';
import { RichEditor } from '@/shared/components';

// 1. 在 Ant Design Form.Item 中直接使用，双向数据自动绑定：
<Form.Item
  name="content"
  label="商品图文详情"
  rules={[{ required: true, message: '请输入详情内容' }]}
>
  <RichEditor
    placeholder="请输入商品图文详情或文章内容..."
    minHeight={300}
    allowImageLibrary={true}
  />
</Form.Item>

// 2. 作为独立受控组件使用：
const [html, setHtml] = useState('<p>初始内容</p>');
<RichEditor value={html} onChange={setHtml} />`;

export default function EditorDemoPage() {
  const [form] = Form.useForm<DemoFormData>();
  const [submittedData, setSubmittedData] = useState<DemoFormData>();

  const handleFinish = (values: DemoFormData) => {
    setSubmittedData(values);
  };

  const handleLoadPreset = () => {
    form.setFieldsValue({
      title: '2026 夏季新款透气亚麻休闲衬衫',
      category: 'clothing',
      content: presetContent,
    });
  };

  const handleReset = () => {
    form.resetFields();
    setSubmittedData(undefined);
  };

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      {/* 头部特性介绍 */}
      <Alert
        type="info"
        showIcon
        title="Tiptap 富文本编辑器 Demo"
        description="基于 ProseMirror 与 Tiptap 核心封装，采用 Headless 架构 100% 融入 KK-CRM 风格与 Ant Design 组件库。全面支持精细字号选择、文本着色与荧光高亮、段落行距调节、上标/下标、任务待办清单、多源视频嵌入 (MP4/B站/YouTube)、原厂表格 (增删行列/合并拆分/列宽拖拽)、图片尺寸与对齐排版、素材库一键插图，内置文本查找与替换以及全屏沉浸编辑模式。"
      />

      <Row gutter={16}>
        {/* 左侧：实时表单操作 */}
        <Col xs={24} lg={14}>
          <Card
            title={
              <Space>
                <FileTextOutlined style={{ color: '#4338CA' }} />
                <span>图文内容编辑</span>
              </Space>
            }
            extra={
              <Button size="small" icon={<ReloadOutlined />} onClick={handleLoadPreset}>
                载入预设图文
              </Button>
            }
          >
            <Form
              form={form}
              layout="vertical"
              onFinish={handleFinish}
              initialValues={{
                title: '2026 夏季新款透气亚麻休闲衬衫',
                category: 'clothing',
                content: presetContent,
              }}
            >
              <Row gutter={12}>
                <Col span={16}>
                  <Form.Item
                    name="title"
                    label="文章 / 商品标题"
                    rules={[{ required: true, message: '请输入标题' }]}
                  >
                    <Input placeholder="请输入标题" maxLength={100} showCount />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item
                    name="category"
                    label="所属分类"
                    rules={[{ required: true, message: '请选择分类' }]}
                  >
                    <Select
                      placeholder="请选择"
                      options={[
                        { value: 'clothing', label: '服饰鞋包' },
                        { value: 'digital', label: '数码家电' },
                        { value: 'food', label: '食品生鲜' },
                        { value: 'notice', label: '商城公告' },
                      ]}
                    />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item
                name="content"
                label="正文详情内容"
                extra="💡 快捷功能提示：①支持字号(12~32px)、行距(1.25~2.5倍)、文字颜色与荧光高亮；②待办复选框可勾选；③支持插入 MP4/B站/YouTube 视频；④点击放大镜或按 Ctrl+F 唤起查找与替换；⑤右侧全屏按钮可进入沉浸编辑；⑥表格支持拖拽列宽与增删合并；⑦图片可拖拽缩放。"
                rules={[{ required: true, message: '请输入详情内容' }]}
              >
                <RichEditor
                  placeholder="请输入商品图文详情或公告内容..."
                  minHeight={320}
                  allowImageLibrary={true}
                />
              </Form.Item>

              <Space style={{ marginTop: 8 }}>
                <Button type="primary" htmlType="submit" icon={<SendOutlined />}>
                  查看提交数据
                </Button>
                <Button onClick={handleReset}>清空表单</Button>
              </Space>
            </Form>
          </Card>
        </Col>

        {/* 右侧：实时结果查看 */}
        <Col xs={24} lg={10}>
          <Card
            title={
              <Space>
                <EyeOutlined style={{ color: '#6366F1' }} />
                <span>输出与渲染效果</span>
              </Space>
            }
          >
            {submittedData ? (
              <Tabs
                defaultActiveKey="preview"
                items={[
                  {
                    key: 'preview',
                    label: '页面实际渲染',
                    children: (
                      <div>
                        <Typography.Title level={4} style={{ marginTop: 0 }}>
                          {submittedData.title}
                        </Typography.Title>
                        <div
                          style={{
                            padding: '12px 16px',
                            background: '#F9FAFB',
                            border: '1px solid #E5E7EB',
                            borderRadius: 6,
                            maxHeight: 460,
                            overflowY: 'auto',
                            lineHeight: 1.7,
                          }}
                          dangerouslySetInnerHTML={{ __html: submittedData.content }}
                        />
                      </div>
                    ),
                  },
                  {
                    key: 'html',
                    label: 'HTML 源码',
                    children: (
                      <pre
                        style={{
                          background: '#111827',
                          color: '#F9FAFB',
                          padding: 12,
                          borderRadius: 6,
                          fontSize: 12,
                          maxHeight: 460,
                          overflow: 'auto',
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-all',
                        }}
                      >
                        {submittedData.content}
                      </pre>
                    ),
                  },
                  {
                    key: 'json',
                    label: '表单提交 JSON',
                    children: (
                      <pre
                        style={{
                          background: '#F3F4F6',
                          color: '#111827',
                          padding: 12,
                          borderRadius: 6,
                          fontSize: 12,
                          maxHeight: 460,
                          overflow: 'auto',
                        }}
                      >
                        {JSON.stringify(submittedData, null, 2)}
                      </pre>
                    ),
                  },
                ]}
              />
            ) : (
              <div
                style={{
                  padding: '40px 20px',
                  textAlign: 'center',
                  color: '#9CA3AF',
                  background: '#F9FAFB',
                  borderRadius: 6,
                  border: '1px dashed #E5E7EB',
                }}
              >
                <p>点击左侧“查看提交数据”按钮，可在此实时预览生成的 HTML 及其渲染结果。</p>
              </div>
            )}
          </Card>

          <Card
            title={
              <Space>
                <CodeOutlined style={{ color: '#10B981' }} />
                <span>开发接入代码示例</span>
              </Space>
            }
            style={{ marginTop: 16 }}
          >
            <Typography.Paragraph copyable={{ text: usageCode }}>
              <pre
                style={{
                  background: '#1E293B',
                  color: '#E2E8F0',
                  padding: 12,
                  borderRadius: 6,
                  fontSize: 12,
                  overflowX: 'auto',
                }}
              >
                {usageCode}
              </pre>
            </Typography.Paragraph>
          </Card>
        </Col>
      </Row>
    </Space>
  );
}
