import { useEffect, useRef, useState } from 'react';
import type { Editor } from '@tiptap/react';
import {
  CloseOutlined,
  DownOutlined,
  SearchOutlined,
  SwapOutlined,
  UpOutlined,
} from '@ant-design/icons';
import { Button, Checkbox, Input, Space, Tooltip, message } from 'antd';

type Props = {
  editor: Editor;
  open: boolean;
  onClose: () => void;
};

export function FindReplaceBar({ editor, open, onClose }: Props) {
  const [searchTerm, setSearchTerm] = useState('');
  const [replaceTerm, setReplaceTerm] = useState('');
  const [matchCase, setMatchCase] = useState(false);
  const [currentIndex, setCurrentIndex] = useState(0);
  const searchInputRef = useRef<any>(null);

  // 检索所有匹配位置
  const getMatches = () => {
    if (!searchTerm || !editor) return [];
    const matches: Array<{ from: number; to: number }> = [];
    editor.state.doc.descendants((node, pos) => {
      if (!node.isText || !node.text) return;
      const text = matchCase ? node.text : node.text.toLowerCase();
      const query = matchCase ? searchTerm : searchTerm.toLowerCase();
      let index = text.indexOf(query);
      while (index !== -1) {
        matches.push({ from: pos + index, to: pos + index + query.length });
        index = text.indexOf(query, index + query.length);
      }
    });
    return matches;
  };

  const matches = getMatches();

  useEffect(() => {
    if (open) {
      setTimeout(() => searchInputRef.current?.focus(), 80);
    } else {
      setSearchTerm('');
      setReplaceTerm('');
      setCurrentIndex(0);
    }
  }, [open]);

  // 当搜索关键词变化时，跳转到第一个匹配项
  useEffect(() => {
    if (matches.length > 0) {
      setCurrentIndex(0);
      const target = matches[0];
      editor.chain().setTextSelection(target).scrollIntoView().run();
    } else {
      setCurrentIndex(0);
    }
  }, [searchTerm, matchCase]);

  if (!open) return null;

  const handleNext = () => {
    if (!matches.length) return;
    const nextIdx = (currentIndex + 1) % matches.length;
    setCurrentIndex(nextIdx);
    const target = matches[nextIdx];
    editor.chain().setTextSelection(target).scrollIntoView().run();
  };

  const handlePrev = () => {
    if (!matches.length) return;
    const prevIdx = (currentIndex - 1 + matches.length) % matches.length;
    setCurrentIndex(prevIdx);
    const target = matches[prevIdx];
    editor.chain().setTextSelection(target).scrollIntoView().run();
  };

  const handleReplace = () => {
    if (!matches.length) {
      message.info('未找到匹配文本');
      return;
    }
    const current = matches[currentIndex] || matches[0];
    editor.chain().focus().setTextSelection(current).insertContent(replaceTerm).run();
  };

  const handleReplaceAll = () => {
    if (!matches.length) {
      message.info('未找到匹配文本');
      return;
    }
    const count = matches.length;
    const { state, view } = editor;
    const tr = state.tr;
    // 逆序替换防止 offset 偏移错乱
    [...matches].reverse().forEach(({ from, to }) => {
      tr.insertText(replaceTerm, from, to);
    });
    view.dispatch(tr);
    message.success(`已替换 ${count} 处匹配文本`);
  };

  return (
    <div className="rich-editor-find-replace">
      <Space size={8} wrap align="center">
        {/* 查找区域 */}
        <Input
          ref={searchInputRef}
          size="small"
          prefix={<SearchOutlined style={{ color: '#9CA3AF' }} />}
          placeholder="查找文本..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          onPressEnter={(e) => {
            if (e.shiftKey) handlePrev();
            else handleNext();
          }}
          style={{ width: 170 }}
        />

        <Checkbox checked={matchCase} onChange={(e) => setMatchCase(e.target.checked)}>
          <span style={{ fontSize: 12, color: '#4B5563' }}>区分大小写</span>
        </Checkbox>

        <span style={{ fontSize: 12, color: matches.length > 0 ? '#4338CA' : '#9CA3AF', minWidth: 40 }}>
          {searchTerm ? `${matches.length > 0 ? currentIndex + 1 : 0} / ${matches.length}` : '0 / 0'}
        </span>

        <Tooltip title="上一个 (Shift+Enter)">
          <Button
            size="small"
            icon={<UpOutlined />}
            disabled={!matches.length}
            onClick={handlePrev}
          />
        </Tooltip>
        <Tooltip title="下一个 (Enter)">
          <Button
            size="small"
            icon={<DownOutlined />}
            disabled={!matches.length}
            onClick={handleNext}
          />
        </Tooltip>

        <div className="find-replace-sep" />

        {/* 替换区域 */}
        <Input
          size="small"
          prefix={<SwapOutlined style={{ color: '#9CA3AF' }} />}
          placeholder="替换为..."
          value={replaceTerm}
          onChange={(e) => setReplaceTerm(e.target.value)}
          onPressEnter={handleReplace}
          style={{ width: 170 }}
        />

        <Button size="small" disabled={!matches.length} onClick={handleReplace}>
          替换
        </Button>
        <Button size="small" disabled={!matches.length} onClick={handleReplaceAll}>
          全部替换
        </Button>

        <Tooltip title="关闭查找与替换 (Esc)">
          <Button
            type="text"
            size="small"
            icon={<CloseOutlined />}
            onClick={onClose}
            style={{ color: '#6B7280' }}
          />
        </Tooltip>
      </Space>
    </div>
  );
}
