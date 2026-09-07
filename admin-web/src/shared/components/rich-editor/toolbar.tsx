import { useEffect, useState } from 'react';
import type { Editor } from '@tiptap/react';
import {
  AlignCenterOutlined,
  AlignLeftOutlined,
  AlignRightOutlined,
  BoldOutlined,
  CheckSquareOutlined,
  ClearOutlined,
  CodeOutlined,
  CommentOutlined,
  DisconnectOutlined,
  FullscreenExitOutlined,
  FullscreenOutlined,
  ItalicOutlined,
  LinkOutlined,
  MinusOutlined,
  OrderedListOutlined,
  PictureOutlined,
  RedoOutlined,
  SearchOutlined,
  StrikethroughOutlined,
  UnderlineOutlined,
  UndoOutlined,
  UnorderedListOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons';
import { Select, Tooltip } from 'antd';
import { ImageSelector } from '@/modules/system/file';
import type { FileRow } from '@/framework/http';
import { LinkModal } from './link-modal';
import { TableControl } from './table-control';
import { TextColorPicker, HighlightColorPicker } from './color-picker';
import { VideoModal } from './video-modal';

type Props = {
  editor: Editor | null;
  disabled?: boolean;
  allowImageLibrary?: boolean;
  isFullscreen?: boolean;
  onToggleFullscreen?: () => void;
  findReplaceOpen?: boolean;
  onToggleFindReplace?: () => void;
};

const FONT_SIZE_OPTIONS = [
  { value: '', label: '字号' },
  { value: '12px', label: '12px' },
  { value: '13px', label: '13px' },
  { value: '14px', label: '14px' },
  { value: '15px', label: '15px' },
  { value: '16px', label: '16px' },
  { value: '18px', label: '18px' },
  { value: '20px', label: '20px' },
  { value: '24px', label: '24px' },
  { value: '28px', label: '28px' },
  { value: '32px', label: '32px' },
];

const LINE_HEIGHT_OPTIONS = [
  { value: '', label: '行距' },
  { value: '1.25', label: '1.25' },
  { value: '1.5', label: '1.5' },
  { value: '1.75', label: '1.75' },
  { value: '2', label: '2.0' },
  { value: '2.5', label: '2.5' },
];

export function EditorToolbar({
  editor,
  disabled = false,
  allowImageLibrary = true,
  isFullscreen = false,
  onToggleFullscreen,
  findReplaceOpen = false,
  onToggleFindReplace,
}: Props) {
  const [linkModalOpen, setLinkModalOpen] = useState(false);
  const [imageSelectorOpen, setImageSelectorOpen] = useState(false);
  const [videoModalOpen, setVideoModalOpen] = useState(false);
  const [, setTick] = useState(0);

  // 监听光标与选区变化，确保工具栏各按钮状态即时同步
  useEffect(() => {
    if (!editor) return;
    const handleUpdate = () => {
      setTick((t) => (t + 1) % 10000);
    };
    editor.on('transaction', handleUpdate);
    return () => {
      editor.off('transaction', handleUpdate);
    };
  }, [editor]);

  if (!editor) return null;

  const currentHeading = editor.isActive('heading', { level: 1 })
    ? '1'
    : editor.isActive('heading', { level: 2 })
    ? '2'
    : editor.isActive('heading', { level: 3 })
    ? '3'
    : 'paragraph';

  const currentFontSize = (editor.getAttributes('textStyle').fontSize as string) || '';
  const currentLineHeight = (editor.getAttributes('textStyle').lineHeight as string) || '';

  const handleHeadingChange = (value: string) => {
    if (value === 'paragraph') {
      editor.chain().focus().setParagraph().run();
    } else {
      const level = parseInt(value, 10) as 1 | 2 | 3;
      editor.chain().focus().toggleHeading({ level }).run();
    }
  };

  const handleFontSizeChange = (val: string) => {
    if (!val) {
      editor.chain().focus().unsetFontSize().run();
    } else {
      editor.chain().focus().setFontSize(val).run();
    }
  };

  const handleLineHeightChange = (val: string) => {
    if (!val) {
      editor.chain().focus().unsetLineHeight().run();
    } else {
      editor.chain().focus().setLineHeight(val).run();
    }
  };

  const handleLinkConfirm = (href: string) => {
    editor.chain().focus().extendMarkRange('link').setLink({ href }).run();
    setLinkModalOpen(false);
  };

  const handleUnlink = () => {
    editor.chain().focus().unsetLink().run();
  };

  const handleImagesConfirm = (files: FileRow[]) => {
    setImageSelectorOpen(false);
    if (!files.length) return;
    const chain = editor.chain().focus();
    files.forEach((file) => {
      chain.setImage({ src: file.url, alt: file.name });
    });
    chain.run();
  };

  const handleVideoConfirm = (val: { src: string; width: string; alignment: string }) => {
    editor.chain().focus().setVideo(val).run();
    setVideoModalOpen(false);
  };

  return (
    <div className="rich-editor-toolbar">
      {/* 撤销 / 重做 */}
      <Tooltip title="撤销 (Ctrl+Z)">
        <button
          type="button"
          className="toolbar-btn"
          disabled={disabled || !editor.can().undo()}
          onClick={() => editor.chain().focus().undo().run()}
        >
          <UndoOutlined />
        </button>
      </Tooltip>
      <Tooltip title="重做 (Ctrl+Y)">
        <button
          type="button"
          className="toolbar-btn"
          disabled={disabled || !editor.can().redo()}
          onClick={() => editor.chain().focus().redo().run()}
        >
          <RedoOutlined />
        </button>
      </Tooltip>

      <div className="toolbar-divider" />

      {/* 段落与标题选择 */}
      <Select
        className="toolbar-select heading-select"
        size="small"
        value={currentHeading}
        disabled={disabled}
        onChange={handleHeadingChange}
        options={[
          { value: 'paragraph', label: '正文' },
          { value: '1', label: 'H1' },
          { value: '2', label: 'H2' },
          { value: '3', label: 'H3' },
        ]}
        style={{ width: 68 }}
      />

      {/* 字号选择 */}
      <Select
        className="toolbar-select font-size-select"
        size="small"
        value={currentFontSize}
        disabled={disabled}
        onChange={handleFontSizeChange}
        options={FONT_SIZE_OPTIONS}
        style={{ width: 66 }}
      />

      {/* 行距选择 */}
      <Select
        className="toolbar-select line-height-select"
        size="small"
        value={currentLineHeight}
        disabled={disabled}
        onChange={handleLineHeightChange}
        options={LINE_HEIGHT_OPTIONS}
        style={{ width: 66 }}
      />

      <div className="toolbar-divider" />

      {/* 基础文本格式 */}
      <Tooltip title="粗体 (Ctrl+B)">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('bold') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleBold().run()}
        >
          <BoldOutlined />
        </button>
      </Tooltip>
      <Tooltip title="斜体 (Ctrl+I)">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('italic') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleItalic().run()}
        >
          <ItalicOutlined />
        </button>
      </Tooltip>
      <Tooltip title="下划线 (Ctrl+U)">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('underline') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleUnderline().run()}
        >
          <UnderlineOutlined />
        </button>
      </Tooltip>
      <Tooltip title="删除线">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('strike') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleStrike().run()}
        >
          <StrikethroughOutlined />
        </button>
      </Tooltip>
      <Tooltip title="行内代码">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('code') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleCode().run()}
        >
          <CodeOutlined />
        </button>
      </Tooltip>

      {/* 上标 / 下标 */}
      <Tooltip title="下标 (X₂)">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('subscript') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleSubscript().run()}
        >
          <span style={{ fontSize: 11, fontWeight: 600 }}>X₂</span>
        </button>
      </Tooltip>
      <Tooltip title="上标 (X²)">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('superscript') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleSuperscript().run()}
        >
          <span style={{ fontSize: 11, fontWeight: 600 }}>X²</span>
        </button>
      </Tooltip>

      <div className="toolbar-divider" />

      {/* 文字颜色与背景高亮 */}
      <TextColorPicker editor={editor} disabled={disabled} />
      <HighlightColorPicker editor={editor} disabled={disabled} />

      <div className="toolbar-divider" />

      {/* 对齐方式 */}
      <Tooltip title="左对齐">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive({ textAlign: 'left' }) ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().setTextAlign('left').run()}
        >
          <AlignLeftOutlined />
        </button>
      </Tooltip>
      <Tooltip title="居中对齐">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive({ textAlign: 'center' }) ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().setTextAlign('center').run()}
        >
          <AlignCenterOutlined />
        </button>
      </Tooltip>
      <Tooltip title="右对齐">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive({ textAlign: 'right' }) ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().setTextAlign('right').run()}
        >
          <AlignRightOutlined />
        </button>
      </Tooltip>

      <div className="toolbar-divider" />

      {/* 列表、待办与引用 */}
      <Tooltip title="无序列表">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('bulletList') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleBulletList().run()}
        >
          <UnorderedListOutlined />
        </button>
      </Tooltip>
      <Tooltip title="有序列表">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('orderedList') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleOrderedList().run()}
        >
          <OrderedListOutlined />
        </button>
      </Tooltip>
      <Tooltip title="任务待办清单">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('taskList') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleTaskList().run()}
        >
          <CheckSquareOutlined />
        </button>
      </Tooltip>
      <Tooltip title="引用块">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('blockquote') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleBlockquote().run()}
        >
          <CommentOutlined />
        </button>
      </Tooltip>
      <Tooltip title="代码块">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('codeBlock') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => editor.chain().focus().toggleCodeBlock().run()}
        >
          <span style={{ fontSize: 11, fontWeight: 700 }}>{'{}'}</span>
        </button>
      </Tooltip>
      <Tooltip title="分割线">
        <button
          type="button"
          className="toolbar-btn"
          disabled={disabled}
          onClick={() => editor.chain().focus().setHorizontalRule().run()}
        >
          <MinusOutlined />
        </button>
      </Tooltip>

      <div className="toolbar-divider" />

      {/* 链接 */}
      <Tooltip title="插入/修改超链接">
        <button
          type="button"
          className={`toolbar-btn ${editor.isActive('link') ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={() => setLinkModalOpen(true)}
        >
          <LinkOutlined />
        </button>
      </Tooltip>
      {editor.isActive('link') && (
        <Tooltip title="取消链接">
          <button
            type="button"
            className="toolbar-btn"
            disabled={disabled}
            onClick={handleUnlink}
          >
            <DisconnectOutlined />
          </button>
        </Tooltip>
      )}

      {/* 插入图片素材 */}
      {allowImageLibrary && (
        <Tooltip title="从素材库插入图片">
          <button
            type="button"
            className="toolbar-btn"
            disabled={disabled}
            onClick={() => setImageSelectorOpen(true)}
          >
            <PictureOutlined style={{ color: 'var(--ant-color-primary, #4338CA)' }} />
          </button>
        </Tooltip>
      )}

      {/* 插入视频 */}
      <Tooltip title="插入视频 (MP4 / B站 / YouTube)">
        <button
          type="button"
          className="toolbar-btn"
          disabled={disabled}
          onClick={() => setVideoModalOpen(true)}
        >
          <VideoCameraOutlined />
        </button>
      </Tooltip>

      {/* 表格操作与插入 */}
      <TableControl editor={editor} disabled={disabled} />

      <div className="toolbar-divider" />

      {/* 查找与替换 */}
      <Tooltip title="查找与替换 (Ctrl+F)">
        <button
          type="button"
          className={`toolbar-btn ${findReplaceOpen ? 'is-active' : ''}`}
          disabled={disabled}
          onClick={onToggleFindReplace}
        >
          <SearchOutlined />
        </button>
      </Tooltip>

      {/* 清除格式 */}
      <Tooltip title="清除格式">
        <button
          type="button"
          className="toolbar-btn"
          disabled={disabled}
          onClick={() =>
            editor.chain().focus().unsetAllMarks().unsetFontSize().unsetLineHeight().clearNodes().run()
          }
        >
          <ClearOutlined />
        </button>
      </Tooltip>

      {/* 全屏切换 */}
      {onToggleFullscreen && (
        <Tooltip title={isFullscreen ? '退出全屏 (Esc)' : '全屏编辑'}>
          <button
            type="button"
            className={`toolbar-btn ${isFullscreen ? 'is-active' : ''}`}
            onClick={onToggleFullscreen}
            style={{ marginLeft: 'auto' }}
          >
            {isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
          </button>
        </Tooltip>
      )}

      {/* 弹窗 */}
      <LinkModal
        open={linkModalOpen}
        initialHref={editor.getAttributes('link').href}
        onConfirm={handleLinkConfirm}
        onCancel={() => setLinkModalOpen(false)}
      />

      {imageSelectorOpen && (
        <ImageSelector
          multiple
          maxCount={10}
          onConfirm={handleImagesConfirm}
          onCancel={() => setImageSelectorOpen(false)}
        />
      )}

      <VideoModal
        open={videoModalOpen}
        onConfirm={handleVideoConfirm}
        onCancel={() => setVideoModalOpen(false)}
      />
    </div>
  );
}
