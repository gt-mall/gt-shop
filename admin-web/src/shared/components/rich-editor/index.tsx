import { useEffect, useState, type CSSProperties } from 'react';
import { useEditor, EditorContent } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import Link from '@tiptap/extension-link';
import TextAlign from '@tiptap/extension-text-align';
import Placeholder from '@tiptap/extension-placeholder';
import Underline from '@tiptap/extension-underline';
import { Table, TableRow, TableHeader, TableCell } from '@tiptap/extension-table';
import { TextStyle, FontSize, LineHeight } from '@tiptap/extension-text-style';
import { Color } from '@tiptap/extension-color';
import { Highlight } from '@tiptap/extension-highlight';
import { Subscript } from '@tiptap/extension-subscript';
import { Superscript } from '@tiptap/extension-superscript';
import { TaskList } from '@tiptap/extension-task-list';
import { TaskItem } from '@tiptap/extension-task-item';

import { ResizableImage } from './image-node';
import { Video } from './video-node';
import { EditorToolbar } from './toolbar';
import { FindReplaceBar } from './find-replace';
import './style.less';

export type RichEditorProps = {
  value?: string;
  onChange?: (html: string) => void;
  placeholder?: string;
  disabled?: boolean;
  minHeight?: number | string;
  allowImageLibrary?: boolean;
  className?: string;
  style?: CSSProperties;
};

export function RichEditor({
  value = '',
  onChange,
  placeholder = '请输入正文内容...',
  disabled = false,
  minHeight = 240,
  allowImageLibrary = true,
  className = '',
  style,
}: RichEditorProps) {
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [findReplaceOpen, setFindReplaceOpen] = useState(false);

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: {
          levels: [1, 2, 3],
        },
      }),
      Underline,
      TextStyle,
      FontSize,
      LineHeight,
      Color,
      Highlight.configure({
        multicolor: true,
      }),
      Subscript,
      Superscript,
      TaskList,
      TaskItem.configure({
        nested: true,
      }),
      Table.configure({
        resizable: true,
      }),
      TableRow,
      TableHeader,
      TableCell,
      ResizableImage.configure({
        allowBase64: false,
      }),
      Video,
      Link.configure({
        openOnClick: false,
        autolink: true,
        defaultProtocol: 'https',
      }),
      TextAlign.configure({
        types: ['heading', 'paragraph'],
      }),
      Placeholder.configure({
        placeholder,
      }),
    ],
    content: value,
    editable: !disabled,
    onUpdate: ({ editor: ed }) => {
      const html = ed.getHTML();
      onChange?.(ed.isEmpty ? '' : html);
    },
  });

  // 同步外部 value 变化
  useEffect(() => {
    if (!editor) return;
    const currentHtml = editor.getHTML();
    const targetHtml = value || '';
    if (targetHtml !== currentHtml && (targetHtml !== '' || !editor.isEmpty)) {
      editor.commands.setContent(targetHtml, { emitUpdate: false });
    }
  }, [value, editor]);

  // 同步 disabled 状态
  useEffect(() => {
    if (!editor) return;
    editor.setEditable(!disabled);
  }, [disabled, editor]);

  // Esc 快捷键退出全屏
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (findReplaceOpen) {
          setFindReplaceOpen(false);
        } else if (isFullscreen) {
          setIsFullscreen(false);
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isFullscreen, findReplaceOpen]);

  const minHeightStyle = typeof minHeight === 'number' ? `${minHeight}px` : minHeight;

  return (
    <div
      className={`rich-editor-container ${disabled ? 'is-disabled' : ''} ${
        isFullscreen ? 'is-fullscreen' : ''
      } ${className}`}
      style={style}
      onKeyDown={(e) => {
        if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
          e.preventDefault();
          setFindReplaceOpen((prev) => !prev);
        }
      }}
    >
      <EditorToolbar
        editor={editor}
        disabled={disabled}
        allowImageLibrary={allowImageLibrary}
        isFullscreen={isFullscreen}
        onToggleFullscreen={() => setIsFullscreen((prev) => !prev)}
        findReplaceOpen={findReplaceOpen}
        onToggleFindReplace={() => setFindReplaceOpen((prev) => !prev)}
      />

      {editor && (
        <FindReplaceBar
          editor={editor}
          open={findReplaceOpen}
          onClose={() => setFindReplaceOpen(false)}
        />
      )}

      <div
        className="rich-editor-content"
        style={{ minHeight: isFullscreen ? 'auto' : minHeightStyle }}
        onClick={() => {
          if (!editor?.isFocused && !disabled) {
            editor?.commands.focus();
          }
        }}
      >
        <EditorContent
          editor={editor}
          style={{ minHeight: isFullscreen ? '100%' : minHeightStyle }}
        />
      </div>
    </div>
  );
}

export default RichEditor;
