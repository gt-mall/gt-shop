import { useState } from 'react';
import {
  CustomerServiceOutlined,
  EllipsisOutlined,
  EyeOutlined,
  LeftOutlined,
  ShopOutlined,
  ShoppingCartOutlined,
} from '@ant-design/icons';
import { Segmented, Tag } from 'antd';
import { RichEditor } from '@/shared/components/rich-editor';
import './detail-editor.less';

export type ProductDetailEditorProps = {
  value?: string;
  onChange?: (html: string) => void;
  disabled?: boolean;
  productName?: string;
  productPrice?: string;
  coverUrl?: string;
};

export function ProductDetailEditor({
  value = '',
  onChange,
  disabled = false,
  productName,
  productPrice,
  coverUrl,
}: ProductDetailEditorProps) {
  // 默认使用手机端尺寸 (375px) 显示富文本宽度
  const [widthMode, setWidthMode] = useState<'mobile' | 'full'>('mobile');

  return (
    <div className="product-detail-editor">
      <div className="detail-editor-header">
        <div className="header-left">
          <span className="editor-mode-label">排版视窗：</span>
          <Segmented
            value={widthMode}
            onChange={(val) => setWidthMode(val as 'mobile' | 'full')}
            options={[
              { value: 'mobile', label: '📱 手机端尺寸 (375px)' },
              { value: 'full', label: '💻 全宽自适应' },
            ]}
          />
          <span className="editor-hint">
            {widthMode === 'mobile'
              ? '默认按照手机端尺寸 (375px) 宽度排版，与移动端真实展示 1:1 所见即所得'
              : '全宽自适应排版模式，右侧仍保持移动端实时渲染'}
          </span>
        </div>
        <div className="header-right">
          <Tag color="processing" icon={<EyeOutlined />}>
            实时预览已同步
          </Tag>
        </div>
      </div>

      <div className="detail-editor-body">
        {/* 左侧：富文本编辑器 */}
        <div className={`editor-wrapper ${widthMode === 'mobile' ? 'mode-mobile' : 'mode-full'}`}>
          {disabled ? (
            <div
              style={{
                padding: '16px 20px',
                background: '#F9FAFB',
                border: '1px solid #E5E7EB',
                borderRadius: 8,
                minHeight: 300,
                lineHeight: 1.8,
              }}
              dangerouslySetInnerHTML={{ __html: value || '<p style="color:#9CA3AF">暂无详情描述</p>' }}
            />
          ) : (
            <RichEditor
              value={value}
              onChange={onChange}
              disabled={disabled}
              minHeight={520}
              placeholder="请输入商品详细介绍、规格参数表与图文说明..."
              allowImageLibrary={true}
            />
          )}
        </div>

        {/* 右侧：移动端实时预览手机模拟器 */}
        <div className="preview-wrapper">
          <div className="mobile-phone-device">
            <div className="phone-screen">
              {/* 顶部状态栏 */}
              <div className="phone-status-bar">
                <span className="status-time">09:41</span>
                <div className="dynamic-island">
                  <span className="camera-lens" />
                </div>
                <div className="status-icons">
                  <span style={{ fontSize: 11 }}>●●●</span>
                </div>
              </div>

              {/* 顶部导航栏 */}
              <div className="phone-nav-bar">
                <LeftOutlined className="nav-icon" />
                <span className="nav-title">商品详情</span>
                <EllipsisOutlined className="nav-icon" />
              </div>

              {/* 可滚动视窗 */}
              <div className="phone-scroll-body">
                {/* 商品简要头部信息卡片 */}
                <div className="phone-product-snippet">
                  {coverUrl && (
                    <img src={coverUrl} alt="封面" className="snippet-cover" />
                  )}
                  <div className="snippet-meta">
                    <div className="snippet-price-row">
                      <span className="price-symbol">¥</span>
                      <span className="price-val">{productPrice || '0.00'}</span>
                    </div>
                    <div className="snippet-name">{productName || '未命名商品'}</div>
                  </div>
                </div>

                <div className="phone-detail-divider">图文详情预览</div>

                {/* 实时渲染富文本 HTML */}
                {value ? (
                  <div
                    className="phone-html-viewport"
                    dangerouslySetInnerHTML={{ __html: value }}
                  />
                ) : (
                  <div className="phone-empty-placeholder">
                    <div className="empty-icon">📱</div>
                    <div className="empty-title">暂无图文详情</div>
                    <div className="empty-desc">
                      在左侧编辑器中输入文字或插入图片，此处将实时展示手机端效果
                    </div>
                  </div>
                )}
              </div>

              {/* 底部购买栏 */}
              <div className="phone-bottom-bar">
                <div className="bottom-icons">
                  <div className="bottom-icon-item">
                    <ShopOutlined />
                    <span>店铺</span>
                  </div>
                  <div className="bottom-icon-item">
                    <CustomerServiceOutlined />
                    <span>客服</span>
                  </div>
                  <div className="bottom-icon-item">
                    <ShoppingCartOutlined />
                    <span>购物车</span>
                  </div>
                </div>
                <div className="bottom-buttons">
                  <div className="mini-action-btn cart-btn">加入购物车</div>
                  <div className="mini-action-btn buy-btn">立即购买</div>
                </div>
              </div>

              {/* 底部 Home 提示条 */}
              <div className="phone-home-indicator">
                <div className="home-line" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default ProductDetailEditor;
