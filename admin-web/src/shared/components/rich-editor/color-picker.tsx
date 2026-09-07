import { useState } from 'react';
import type { Editor } from '@tiptap/react';
import { BgColorsOutlined, FontColorsOutlined } from '@ant-design/icons';
import { Button, Popover, Space, Tooltip } from 'antd';

const PRESET_TEXT_COLORS = [
  { label: '默认暗灰', value: '#374151' },
  { label: '深黑', value: '#111827' },
  { label: '品牌靛蓝', value: '#4338CA' },
  { label: '科技蓝', value: '#2563EB' },
  { label: '薄荷绿', value: '#16A34A' },
  { label: '活力橙', value: '#EA580C' },
  { label: '警示红', value: '#DC2626' },
  { label: '梦幻紫', value: '#7C3AED' },
  { label: '优雅粉', value: '#DB2777' },
  { label: '低调灰', value: '#6B7280' },
];

const PRESET_HIGHLIGHT_COLORS = [
  { label: '荧光黄', value: '#FEF08A' },
  { label: '浅草绿', value: '#BBF7D0' },
  { label: '天际蓝', value: '#BAE6FD' },
  { label: '柔粉红', value: '#FECDD3' },
  { label: '薰衣草紫', value: '#DDD6FE' },
  { label: '暖蜜橙', value: '#FED7AA' },
];

type Props = {
  editor: Editor;
  disabled?: boolean;
};

export function TextColorPicker({ editor, disabled = false }: Props) {
  const [open, setOpen] = useState(false);
  const currentColor = editor.getAttributes('textStyle').color || '#374151';

  const handleSelect = (color: string) => {
    editor.chain().focus().setColor(color).run();
    setOpen(false);
  };

  const handleClear = () => {
    editor.chain().focus().unsetColor().run();
    setOpen(false);
  };

  const content = (
    <div style={{ width: 180, padding: 4 }}>
      <div style={{ marginBottom: 8, fontSize: 12, color: '#6B7280' }}>文字颜色</div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: 6, marginBottom: 12 }}>
        {PRESET_TEXT_COLORS.map((c) => (
          <Tooltip key={c.value} title={c.label}>
            <button
              type="button"
              onClick={() => handleSelect(c.value)}
              style={{
                width: 24,
                height: 24,
                borderRadius: 4,
                backgroundColor: c.value,
                border: currentColor === c.value ? '2px solid #4338CA' : '1px solid #D1D5DB',
                cursor: 'pointer',
                outline: 'none',
                padding: 0,
                transform: currentColor === c.value ? 'scale(1.15)' : 'none',
              }}
            />
          </Tooltip>
        ))}
      </div>
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <input
          type="color"
          value={currentColor}
          onChange={(e) => editor.chain().focus().setColor(e.target.value).run()}
          style={{ width: 32, height: 24, padding: 0, border: 'none', cursor: 'pointer', background: 'transparent' }}
          title="自定义取色"
        />
        <Button size="small" type="link" onClick={handleClear} style={{ fontSize: 12, padding: 0 }}>
          恢复默认
        </Button>
      </Space>
    </div>
  );

  return (
    <Popover open={open} onOpenChange={setOpen} content={content} trigger="click" placement="bottom">
      <Tooltip title="文字颜色">
        <button
          type="button"
          className="toolbar-btn"
          disabled={disabled}
          onMouseDown={(e) => e.preventDefault()}
          style={{ position: 'relative' }}
        >
          <FontColorsOutlined />
          <span
            style={{
              position: 'absolute',
              bottom: 2,
              left: 4,
              right: 4,
              height: 3,
              borderRadius: 1,
              backgroundColor: currentColor,
            }}
          />
        </button>
      </Tooltip>
    </Popover>
  );
}

export function HighlightColorPicker({ editor, disabled = false }: Props) {
  const [open, setOpen] = useState(false);
  const currentHighlight = editor.getAttributes('highlight').color;

  const handleSelect = (color: string) => {
    editor.chain().focus().toggleHighlight({ color }).run();
    setOpen(false);
  };

  const handleClear = () => {
    editor.chain().focus().unsetHighlight().run();
    setOpen(false);
  };

  const content = (
    <div style={{ width: 180, padding: 4 }}>
      <div style={{ marginBottom: 8, fontSize: 12, color: '#6B7280' }}>高亮背景色</div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 8, marginBottom: 12 }}>
        {PRESET_HIGHLIGHT_COLORS.map((c) => (
          <Tooltip key={c.value} title={c.label}>
            <button
              type="button"
              onClick={() => handleSelect(c.value)}
              style={{
                height: 24,
                borderRadius: 4,
                backgroundColor: c.value,
                border: currentHighlight === c.value ? '2px solid #4338CA' : '1px solid #D1D5DB',
                cursor: 'pointer',
                padding: 0,
              }}
            />
          </Tooltip>
        ))}
      </div>
      <Button size="small" block onClick={handleClear} style={{ fontSize: 12 }}>
        清除高亮
      </Button>
    </div>
  );

  return (
    <Popover open={open} onOpenChange={setOpen} content={content} trigger="click" placement="bottom">
      <Tooltip title="文本高亮色">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('highlight') ? 'is-active' : ''}`}
          disabled={disabled}
          onMouseDown={(e) => e.preventDefault()}
          style={{ position: 'relative' }}
        >
          <BgColorsOutlined />
          {currentHighlight && (
            <span
              style={{
                position: 'absolute',
                bottom: 2,
                left: 4,
                right: 4,
                height: 3,
                borderRadius: 1,
                backgroundColor: currentHighlight,
              }}
            />
          )}
        </button>
      </Tooltip>
    </Popover>
  );
}
