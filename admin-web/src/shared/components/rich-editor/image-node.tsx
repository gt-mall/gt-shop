import { useRef, useState, type MouseEvent as ReactMouseEvent } from 'react';
import { NodeViewWrapper, ReactNodeViewRenderer, type NodeViewProps } from '@tiptap/react';
import Image from '@tiptap/extension-image';
import {
  AlignCenterOutlined,
  AlignLeftOutlined,
  AlignRightOutlined,
  DeleteOutlined,
  EditOutlined,
} from '@ant-design/icons';
import { Button, InputNumber, Popover, Space, Tooltip } from 'antd';

export function ImageNodeView({
  node,
  updateAttributes,
  selected,
  deleteNode,
  editor,
  getPos,
}: NodeViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const boxRef = useRef<HTMLDivElement>(null);
  const [resizing, setResizing] = useState(false);
  const [customWidth, setCustomWidth] = useState<number>(() => {
    const raw = String(node.attrs.width || '100%');
    const parsed = parseInt(raw, 10);
    return isNaN(parsed) ? 300 : parsed;
  });
  const [popoverOpen, setPopoverOpen] = useState(false);

  const { src, alt, width = '100%', alignment = 'center' } = node.attrs;

  // 选中当前节点
  const handleSelectNode = () => {
    if (typeof getPos === 'function') {
      const pos = getPos();
      if (typeof pos === 'number') {
        editor.commands.setNodeSelection(pos);
      }
    }
  };

  // 快捷比例设置
  const setPresetWidth = (preset: string) => {
    updateAttributes({ width: preset });
  };

  // 对齐设置
  const setAlignment = (align: 'left' | 'center' | 'right') => {
    updateAttributes({ alignment: align });
  };

  // 自定义像素宽度
  const handleCustomWidthSubmit = () => {
    if (customWidth >= 50 && customWidth <= 2000) {
      updateAttributes({ width: `${customWidth}px` });
      setPopoverOpen(false);
    }
  };

  // 鼠标拖拽微调尺寸
  const handleMouseDown = (e: ReactMouseEvent, direction: 'right' | 'left') => {
    e.preventDefault();
    e.stopPropagation();
    if (!boxRef.current || !containerRef.current) return;

    const startX = e.clientX;
    const initialWidth = boxRef.current.offsetWidth;
    const parentWidth = containerRef.current.parentElement?.offsetWidth || 800;

    setResizing(true);

    const onMouseMove = (moveEvent: MouseEvent) => {
      const deltaX = moveEvent.clientX - startX;
      const factor = direction === 'right' ? 1 : -1;
      const targetWidth = Math.max(60, Math.min(parentWidth, initialWidth + deltaX * factor));
      const percentage = Math.round((targetWidth / parentWidth) * 100);
      if (boxRef.current) {
        boxRef.current.style.width = `${percentage}%`;
      }
    };

    const onMouseUp = (upEvent: MouseEvent) => {
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
      setResizing(false);

      const deltaX = upEvent.clientX - startX;
      const factor = direction === 'right' ? 1 : -1;
      const targetWidth = Math.max(60, Math.min(parentWidth, initialWidth + deltaX * factor));
      const percentage = Math.round((targetWidth / parentWidth) * 100);
      updateAttributes({ width: `${percentage}%` });
    };

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  };

  const justifyClass =
    alignment === 'left' ? 'flex-start' : alignment === 'right' ? 'flex-end' : 'center';

  const isToolbarVisible = selected || popoverOpen;

  return (
    <NodeViewWrapper
      ref={containerRef}
      className={`resizable-image-wrapper ${selected ? 'is-selected' : ''}`}
      style={{
        display: 'flex',
        justifyContent: justifyClass,
        width: '100%',
        margin: '12px 0',
      }}
    >
      <div
        ref={boxRef}
        className={`resizable-image-box ${selected ? 'is-active' : ''} ${
          resizing ? 'is-resizing' : ''
        }`}
        style={{
          position: 'relative',
          display: 'inline-block',
          width: width,
          maxWidth: '100%',
        }}
        onClick={handleSelectNode}
      >
        <img
          src={src}
          alt={alt || ''}
          style={{
            width: '100%',
            height: 'auto',
            display: 'block',
            borderRadius: 6,
          }}
        />

        {/* 选中时的浮动快捷操作栏 */}
        {isToolbarVisible && (
          <div className="resizable-image-toolbar" onClick={(e) => e.stopPropagation()}>
            <Space size={4}>
              <span className="toolbar-label">尺寸:</span>
              <button
                type="button"
                className={`size-btn ${width === '25%' ? 'is-active' : ''}`}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => setPresetWidth('25%')}
              >
                25%
              </button>
              <button
                type="button"
                className={`size-btn ${width === '50%' ? 'is-active' : ''}`}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => setPresetWidth('50%')}
              >
                50%
              </button>
              <button
                type="button"
                className={`size-btn ${width === '75%' ? 'is-active' : ''}`}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => setPresetWidth('75%')}
              >
                75%
              </button>
              <button
                type="button"
                className={`size-btn ${width === '100%' ? 'is-active' : ''}`}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => setPresetWidth('100%')}
              >
                100%
              </button>

              <Popover
                open={popoverOpen}
                onOpenChange={setPopoverOpen}
                trigger="click"
                placement="bottom"
                content={
                  <Space direction="vertical" size="small">
                    <span style={{ fontSize: 12, color: '#6B7280' }}>自定义像素宽度 (px):</span>
                    <Space size={4}>
                      <InputNumber
                        size="small"
                        min={50}
                        max={1600}
                        value={customWidth}
                        onChange={(val) => setCustomWidth(val || 300)}
                        onPressEnter={handleCustomWidthSubmit}
                      />
                      <Button size="small" type="primary" onClick={handleCustomWidthSubmit}>
                        应用
                      </Button>
                    </Space>
                  </Space>
                }
              >
                <Tooltip title="自定义像素">
                  <button
                    type="button"
                    className={`size-btn ${String(width).endsWith('px') ? 'is-active' : ''}`}
                    onMouseDown={(e) => e.preventDefault()}
                  >
                    <EditOutlined style={{ fontSize: 11 }} />
                  </button>
                </Tooltip>
              </Popover>

              <div className="toolbar-sep" />

              <span className="toolbar-label">对齐:</span>
              <Tooltip title="居左">
                <button
                  type="button"
                  className={`size-btn ${alignment === 'left' ? 'is-active' : ''}`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => setAlignment('left')}
                >
                  <AlignLeftOutlined />
                </button>
              </Tooltip>
              <Tooltip title="居中">
                <button
                  type="button"
                  className={`size-btn ${alignment === 'center' ? 'is-active' : ''}`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => setAlignment('center')}
                >
                  <AlignCenterOutlined />
                </button>
              </Tooltip>
              <Tooltip title="居右">
                <button
                  type="button"
                  className={`size-btn ${alignment === 'right' ? 'is-active' : ''}`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => setAlignment('right')}
                >
                  <AlignRightOutlined />
                </button>
              </Tooltip>

              <div className="toolbar-sep" />

              <Tooltip title="删除图片">
                <button
                  type="button"
                  className="size-btn danger"
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={deleteNode}
                >
                  <DeleteOutlined />
                </button>
              </Tooltip>
            </Space>
          </div>
        )}

        {/* 选中时的拖拽手柄 */}
        {selected && (
          <>
            <div
              className="resize-handle handle-se"
              onMouseDown={(e) => handleMouseDown(e, 'right')}
              title="拖拽调节宽度"
            />
            <div
              className="resize-handle handle-sw"
              onMouseDown={(e) => handleMouseDown(e, 'left')}
              title="拖拽调节宽度"
            />
          </>
        )}
      </div>
    </NodeViewWrapper>
  );
}

/** 扩展支持宽度与对齐属性的自定义 Tiptap Image 扩展 */
export const ResizableImage = Image.extend({
  name: 'image',

  addAttributes() {
    return {
      ...this.parent?.(),
      width: {
        default: '100%',
        parseHTML: (element) =>
          element.getAttribute('data-width') ||
          element.getAttribute('width') ||
          element.style.width ||
          '100%',
        renderHTML: (attributes) => ({
          'data-width': attributes.width || '100%',
        }),
      },
      alignment: {
        default: 'center',
        parseHTML: (element) => {
          if (element.getAttribute('data-alignment')) {
            return element.getAttribute('data-alignment');
          }
          const margin = element.style.margin || '';
          if (margin.includes('0 0 0 auto')) return 'right';
          if (margin.includes('0 auto 0 0')) return 'left';
          return 'center';
        },
        renderHTML: (attributes) => ({
          'data-alignment': attributes.alignment || 'center',
        }),
      },
    };
  },

  renderHTML({ HTMLAttributes }) {
    const {
      width = '100%',
      alignment = 'center',
      'data-width': dw,
      'data-alignment': da,
      style: customStyle = '',
      ...rest
    } = HTMLAttributes;

    const finalWidth = dw || width;
    const finalAlign = da || alignment;
    const margin =
      finalAlign === 'left' ? '0 auto 0 0' : finalAlign === 'right' ? '0 0 0 auto' : '0 auto';

    return [
      'img',
      {
        ...rest,
        'data-width': finalWidth,
        'data-alignment': finalAlign,
        style: `width: ${finalWidth}; max-width: 100%; height: auto; display: block; margin: ${margin}; ${customStyle}`,
      },
    ];
  },

  addNodeView() {
    return ReactNodeViewRenderer(ImageNodeView);
  },
});
