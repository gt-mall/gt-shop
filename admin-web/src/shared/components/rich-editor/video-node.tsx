import { useState } from 'react';
import { Node, NodeViewWrapper, ReactNodeViewRenderer, mergeAttributes, type NodeViewProps } from '@tiptap/react';
import {
  AlignCenterOutlined,
  AlignLeftOutlined,
  AlignRightOutlined,
  DeleteOutlined,
  EditOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons';
import { Space, Tooltip } from 'antd';
import { VideoModal } from './video-modal';

declare module '@tiptap/react' {
  interface Commands<ReturnType> {
    video: {
      setVideo: (options: { src: string; width?: string; alignment?: string }) => ReturnType;
    };
  }
}

export function VideoNodeView({
  node,
  updateAttributes,
  selected,
  deleteNode,
  editor,
  getPos,
}: NodeViewProps) {
  const [modalOpen, setModalOpen] = useState(false);
  const [interactive, setInteractive] = useState(false);

  const { src = '', width = '100%', alignment = 'center', poster } = node.attrs;

  const handleSelectNode = () => {
    if (typeof getPos === 'function') {
      const pos = getPos();
      if (typeof pos === 'number') {
        editor.commands.setNodeSelection(pos);
      }
    }
  };

  const isIframe =
    src.includes('bilibili.com') ||
    src.includes('youtube.com') ||
    src.includes('youtu.be') ||
    src.includes('youku.com') ||
    src.includes('player.');

  let iframeSrc = src;
  if (src.includes('bilibili.com/video/')) {
    const match = src.match(/video\/(BV[a-zA-Z0-9]+)/);
    if (match) {
      iframeSrc = `//player.bilibili.com/player.html?bvid=${match[1]}&page=1&high_quality=1&danmaku=0`;
    }
  } else if (src.includes('youtube.com/watch?v=')) {
    const match = src.match(/v=([a-zA-Z0-9_-]+)/);
    if (match) {
      iframeSrc = `https://www.youtube.com/embed/${match[1]}`;
    }
  }

  const justifyClass =
    alignment === 'left' ? 'flex-start' : alignment === 'right' ? 'flex-end' : 'center';

  const isToolbarVisible = selected || modalOpen;

  return (
    <NodeViewWrapper
      className={`resizable-video-wrapper ${selected ? 'is-selected' : ''}`}
      style={{
        display: 'flex',
        justifyContent: justifyClass,
        width: '100%',
        margin: '16px 0',
      }}
    >
      <div
        className={`resizable-video-box ${selected ? 'is-active' : ''}`}
        style={{
          position: 'relative',
          display: 'inline-block',
          width: width,
          maxWidth: '100%',
        }}
        onClick={handleSelectNode}
      >
        {/* 浮动快捷操作栏 */}
        {isToolbarVisible && (
          <div className="resizable-video-toolbar" onClick={(e) => e.stopPropagation()}>
            <Space size={4}>
              <Tooltip title="修改视频链接与设置">
                <button
                  type="button"
                  className="size-btn is-active"
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => setModalOpen(true)}
                >
                  <EditOutlined /> 修改
                </button>
              </Tooltip>

              <div className="toolbar-sep" />

              <span className="toolbar-label">尺寸:</span>
              <button
                type="button"
                className={`size-btn ${width === '100%' ? 'is-active' : ''}`}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => updateAttributes({ width: '100%' })}
              >
                100%
              </button>
              <button
                type="button"
                className={`size-btn ${width === '80%' ? 'is-active' : ''}`}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => updateAttributes({ width: '80%' })}
              >
                80%
              </button>
              <button
                type="button"
                className={`size-btn ${width === '60%' ? 'is-active' : ''}`}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => updateAttributes({ width: '60%' })}
              >
                60%
              </button>

              <div className="toolbar-sep" />

              <span className="toolbar-label">对齐:</span>
              <Tooltip title="居左">
                <button
                  type="button"
                  className={`size-btn ${alignment === 'left' ? 'is-active' : ''}`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => updateAttributes({ alignment: 'left' })}
                >
                  <AlignLeftOutlined />
                </button>
              </Tooltip>
              <Tooltip title="居中">
                <button
                  type="button"
                  className={`size-btn ${alignment === 'center' ? 'is-active' : ''}`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => updateAttributes({ alignment: 'center' })}
                >
                  <AlignCenterOutlined />
                </button>
              </Tooltip>
              <Tooltip title="居右">
                <button
                  type="button"
                  className={`size-btn ${alignment === 'right' ? 'is-active' : ''}`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => updateAttributes({ alignment: 'right' })}
                >
                  <AlignRightOutlined />
                </button>
              </Tooltip>

              <div className="toolbar-sep" />

              <Tooltip
                title={
                  interactive
                    ? '锁定视频（恢复防跳转保护）'
                    : '解锁试看（允许在编辑器内点击播放）'
                }
              >
                <button
                  type="button"
                  className={`size-btn ${interactive ? 'is-active' : ''}`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => setInteractive(!interactive)}
                >
                  <PlayCircleOutlined /> {interactive ? '锁定' : '试看'}
                </button>
              </Tooltip>

              <div className="toolbar-sep" />

              <Tooltip title="删除视频">
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

        {/* 视频内容 */}
        <div
          style={{
            position: 'relative',
            width: '100%',
            borderRadius: 8,
            overflow: 'hidden',
          }}
        >
          {isIframe ? (
            <div
              className="video-iframe-container"
              style={{
                position: 'relative',
                paddingBottom: '56.25%',
                height: 0,
                overflow: 'hidden',
                borderRadius: 8,
              }}
            >
              <iframe
                src={iframeSrc}
                scrolling="no"
                frameBorder="0"
                allowFullScreen={true}
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  width: '100%',
                  height: '100%',
                  border: 'none',
                  borderRadius: 8,
                }}
              />
            </div>
          ) : (
            <video
              controls
              playsInline
              src={src}
              poster={poster}
              style={{ width: '100%', height: 'auto', borderRadius: 8, display: 'block' }}
            />
          )}

          {/* 编辑态防误触遮罩层：默认覆盖在视频上，防止点击直接跳转 B 站；点击触发节点选中 */}
          {!interactive && (
            <div
              className="video-click-guard"
              onClick={handleSelectNode}
              title="点击选中视频进行修改或调整"
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                right: 0,
                bottom: 0,
                zIndex: 10,
                cursor: 'pointer',
                background: selected ? 'rgba(67, 56, 202, 0.04)' : 'transparent',
              }}
            />
          )}
        </div>

        {/* 修改弹窗 */}
        <VideoModal
          open={modalOpen}
          initialUrl={src}
          initialWidth={width}
          initialAlignment={alignment}
          onConfirm={(vals) => {
            updateAttributes({
              src: vals.src,
              width: vals.width,
              alignment: vals.alignment,
            });
            setModalOpen(false);
          }}
          onCancel={() => setModalOpen(false)}
        />
      </div>
    </NodeViewWrapper>
  );
}

export const Video = Node.create({
  name: 'video',
  group: 'block',
  selectable: true,
  draggable: true,
  atom: true,

  addAttributes() {
    return {
      src: {
        default: null,
      },
      width: {
        default: '100%',
      },
      alignment: {
        default: 'center',
      },
      poster: {
        default: null,
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: 'video',
        getAttrs: (element) => {
          const el = element as HTMLElement;
          return {
            src: el.getAttribute('src'),
            poster: el.getAttribute('poster'),
            width: el.getAttribute('width') || el.style.width || '100%',
          };
        },
      },
      {
        tag: 'div.video-iframe-container',
        getAttrs: (element) => {
          const el = element as HTMLElement;
          const iframe = el.querySelector('iframe');
          return {
            src: iframe?.getAttribute('src') || el.getAttribute('data-src'),
            width: el.getAttribute('data-width') || el.style.width || '100%',
          };
        },
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    const { src = '', width = '100%', alignment = 'center', poster } = HTMLAttributes;
    const margin =
      alignment === 'left' ? '0 auto 0 0' : alignment === 'right' ? '0 0 0 auto' : '0 auto';

    // 检查是否为 iframe 视频源 (如 B站 或 YouTube)
    const isIframe =
      src.includes('bilibili.com') ||
      src.includes('youtube.com') ||
      src.includes('youtu.be') ||
      src.includes('youku.com') ||
      src.includes('player.');

    if (isIframe) {
      let iframeSrc = src;
      if (src.includes('bilibili.com/video/')) {
        const match = src.match(/video\/(BV[a-zA-Z0-9]+)/);
        if (match) {
          iframeSrc = `//player.bilibili.com/player.html?bvid=${match[1]}&page=1&high_quality=1&danmaku=0`;
        }
      } else if (src.includes('youtube.com/watch?v=')) {
        const match = src.match(/v=([a-zA-Z0-9_-]+)/);
        if (match) {
          iframeSrc = `https://www.youtube.com/embed/${match[1]}`;
        }
      }

      return [
        'div',
        {
          class: 'video-iframe-container',
          'data-src': iframeSrc,
          'data-width': width,
          style: `width: ${width}; max-width: 100%; margin: ${margin}; position: relative; padding-bottom: 56.25%; height: 0; overflow: hidden; border-radius: 8px;`,
        },
        [
          'iframe',
          {
            src: iframeSrc,
            scrolling: 'no',
            border: '0',
            frameborder: 'no',
            framespacing: '0',
            allowfullscreen: 'true',
            style:
              'position: absolute; top: 0; left: 0; width: 100%; height: 100%; border: none; border-radius: 8px;',
          },
        ],
      ];
    }

    // 普通 HTML5 MP4/WebM 视频
    return [
      'div',
      {
        class: 'video-player-container',
        style: `width: ${width}; max-width: 100%; margin: ${margin};`,
      },
      [
        'video',
        mergeAttributes(HTMLAttributes, {
          controls: 'true',
          playsinline: 'true',
          style: 'width: 100%; height: auto; border-radius: 8px; display: block;',
          poster,
        }),
      ],
    ];
  },

  addNodeView() {
    return ReactNodeViewRenderer(VideoNodeView);
  },

  addCommands() {
    return {
      setVideo:
        (options: { src: string; width?: string; alignment?: string }) =>
        ({ commands }) => {
          return commands.insertContent({
            type: this.name,
            attrs: options,
          });
        },
    };
  },
});
