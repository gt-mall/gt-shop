import { useState } from 'react';
import type { Editor } from '@tiptap/react';
import {
  DeleteColumnOutlined,
  DeleteOutlined,
  DeleteRowOutlined,
  InsertRowAboveOutlined,
  InsertRowBelowOutlined,
  InsertRowLeftOutlined,
  InsertRowRightOutlined,
  MergeCellsOutlined,
  PlusOutlined,
  SplitCellsOutlined,
  TableOutlined,
} from '@ant-design/icons';
import { Button, Checkbox, Dropdown, InputNumber, Space, Tooltip, type MenuProps } from 'antd';

type Props = {
  editor: Editor;
  disabled?: boolean;
};

export function TableControl({ editor, disabled = false }: Props) {
  const [popoverOpen, setPopoverOpen] = useState(false);
  const [rows, setRows] = useState<number>(3);
  const [cols, setCols] = useState<number>(3);
  const [withHeader, setWithHeader] = useState<boolean>(true);

  const isTableActive = editor.isActive('table');

  // 执行插入表格
  const handleInsert = (r: number, c: number, header: boolean) => {
    editor
      .chain()
      .focus()
      .insertTable({ rows: r, cols: c, withHeaderRow: header })
      .run();
    setPopoverOpen(false);
  };

  // 下拉菜单项（当光标位于表格内时）
  const menuItems: MenuProps['items'] = [
    {
      key: 'row-header',
      type: 'group',
      label: '行操作',
      children: [
        {
          key: 'addRowBefore',
          label: '在上方插入行',
          icon: <InsertRowAboveOutlined />,
          onClick: () => editor.chain().focus().addRowBefore().run(),
        },
        {
          key: 'addRowAfter',
          label: '在下方插入行',
          icon: <InsertRowBelowOutlined />,
          onClick: () => editor.chain().focus().addRowAfter().run(),
        },
        {
          key: 'deleteRow',
          label: '删除当前行',
          danger: true,
          icon: <DeleteRowOutlined />,
          onClick: () => editor.chain().focus().deleteRow().run(),
        },
      ],
    },
    { type: 'divider' },
    {
      key: 'col-header',
      type: 'group',
      label: '列操作',
      children: [
        {
          key: 'addColumnBefore',
          label: '在左侧插入列',
          icon: <InsertRowLeftOutlined />,
          onClick: () => editor.chain().focus().addColumnBefore().run(),
        },
        {
          key: 'addColumnAfter',
          label: '在右侧插入列',
          icon: <InsertRowRightOutlined />,
          onClick: () => editor.chain().focus().addColumnAfter().run(),
        },
        {
          key: 'deleteColumn',
          label: '删除当前列',
          danger: true,
          icon: <DeleteColumnOutlined />,
          onClick: () => editor.chain().focus().deleteColumn().run(),
        },
      ],
    },
    { type: 'divider' },
    {
      key: 'cell-header',
      type: 'group',
      label: '单元格操作',
      children: [
        {
          key: 'mergeCells',
          label: '合并单元格',
          icon: <MergeCellsOutlined />,
          disabled: !editor.can().mergeCells(),
          onClick: () => editor.chain().focus().mergeCells().run(),
        },
        {
          key: 'splitCell',
          label: '拆分单元格',
          icon: <SplitCellsOutlined />,
          disabled: !editor.can().splitCell(),
          onClick: () => editor.chain().focus().splitCell().run(),
        },
        {
          key: 'toggleHeaderRow',
          label: '切换首行表头',
          icon: <TableOutlined />,
          onClick: () => editor.chain().focus().toggleHeaderRow().run(),
        },
      ],
    },
    { type: 'divider' },
    {
      key: 'deleteTable',
      label: '删除整个表格',
      danger: true,
      icon: <DeleteOutlined />,
      onClick: () => editor.chain().focus().deleteTable().run(),
    },
  ];

  // 插入表格浮层内容
  const insertContent = (
    <div style={{ width: 220, padding: 4 }}>
      <div style={{ marginBottom: 8, fontSize: 12, color: '#6B7280' }}>常用预设：</div>
      <Space size={6} style={{ marginBottom: 12, width: '100%' }}>
        <Button size="small" onClick={() => handleInsert(3, 3, true)}>
          3 × 3
        </Button>
        <Button size="small" onClick={() => handleInsert(4, 4, true)}>
          4 × 4
        </Button>
        <Button size="small" onClick={() => handleInsert(2, 5, true)}>
          2 × 5
        </Button>
      </Space>

      <div style={{ marginBottom: 8, fontSize: 12, color: '#6B7280' }}>自定义行列：</div>
      <Space direction="vertical" size={8} style={{ width: '100%' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ fontSize: 12 }}>行数 (1-30):</span>
          <InputNumber
            size="small"
            min={1}
            max={30}
            value={rows}
            onChange={(val) => setRows(val || 3)}
            style={{ width: 90 }}
          />
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ fontSize: 12 }}>列数 (1-12):</span>
          <InputNumber
            size="small"
            min={1}
            max={12}
            value={cols}
            onChange={(val) => setCols(val || 3)}
            style={{ width: 90 }}
          />
        </div>
        <div>
          <Checkbox checked={withHeader} onChange={(e) => setWithHeader(e.target.checked)}>
            <span style={{ fontSize: 12 }}>首行为表头</span>
          </Checkbox>
        </div>
        <Button
          type="primary"
          size="small"
          block
          icon={<PlusOutlined />}
          onClick={() => handleInsert(rows, cols, withHeader)}
        >
          插入表格
        </Button>
      </Space>
    </div>
  );

  return (
    <Space size={2} align="center">
      {isTableActive ? (
        // 表格内部激活状态：点击弹出详细表格菜单
        <Dropdown menu={{ items: menuItems }} trigger={['click']} placement="bottomLeft">
          <Tooltip title="表格操作菜单 (增删行列、合并拆分)">
            <button
              type="button"
              className="toolbar-btn is-active"
              disabled={disabled}
              onMouseDown={(e) => e.preventDefault()}
            >
              <TableOutlined />
            </button>
          </Tooltip>
        </Dropdown>
      ) : (
        // 未在表格内：点击弹出插入表格弹窗
        <Dropdown
          open={popoverOpen}
          onOpenChange={setPopoverOpen}
          trigger={['click']}
          placement="bottomLeft"
          popupRender={() => (
            <div
              style={{
                background: '#FFFFFF',
                borderRadius: 6,
                boxShadow: '0 4px 16px rgba(0, 0, 0, 0.12)',
                border: '1px solid #E5E7EB',
                padding: 12,
              }}
            >
              {insertContent}
            </div>
          )}
        >
          <Tooltip title="插入表格">
            <button
              type="button"
              className="toolbar-btn"
              disabled={disabled}
              onMouseDown={(e) => e.preventDefault()}
            >
              <TableOutlined />
            </button>
          </Tooltip>
        </Dropdown>
      )}

      {/* 当光标在表格内时，展示快捷增减行列小按钮，提升编辑效率 */}
      {isTableActive && (
        <>
          <Tooltip title="在下方加一行">
            <button
              type="button"
              className="toolbar-btn"
              disabled={disabled}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().addRowAfter().run()}
            >
              <InsertRowBelowOutlined />
            </button>
          </Tooltip>
          <Tooltip title="在右侧加一列">
            <button
              type="button"
              className="toolbar-btn"
              disabled={disabled}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().addColumnAfter().run()}
            >
              <InsertRowRightOutlined />
            </button>
          </Tooltip>
          <Tooltip title="删除当前行">
            <button
              type="button"
              className="toolbar-btn danger"
              disabled={disabled}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().deleteRow().run()}
            >
              <DeleteRowOutlined />
            </button>
          </Tooltip>
          <Tooltip title="删除当前列">
            <button
              type="button"
              className="toolbar-btn danger"
              disabled={disabled}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().deleteColumn().run()}
            >
              <DeleteColumnOutlined />
            </button>
          </Tooltip>
          {editor.can().mergeCells() && (
            <Tooltip title="合并选中单元格">
              <button
                type="button"
                className="toolbar-btn"
                disabled={disabled}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => editor.chain().focus().mergeCells().run()}
              >
                <MergeCellsOutlined />
              </button>
            </Tooltip>
          )}
          {editor.can().splitCell() && (
            <Tooltip title="拆分单元格">
              <button
                type="button"
                className="toolbar-btn"
                disabled={disabled}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => editor.chain().focus().splitCell().run()}
              >
                <SplitCellsOutlined />
              </button>
            </Tooltip>
          )}
        </>
      )}
    </Space>
  );
}
