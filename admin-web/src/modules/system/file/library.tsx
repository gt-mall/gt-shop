import { useEffect, useMemo, useRef, useState, type DragEvent } from 'react';
import {
  CheckCircleFilled,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  FolderOutlined,
  InboxOutlined,
  InfoCircleOutlined,
  PlusOutlined,
  SwapOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import {
  Button,
  Dropdown,
  Empty,
  Form,
  Image,
  Input,
  message,
  Modal,
  Pagination,
  Popconfirm,
  Select,
  Space,
  Spin,
  Tag,
  Tooltip,
  type MenuProps,
} from 'antd';

import { api, type FileGroupCounts, type FileGroupRow, type FileListRequest, type FileRow } from '@/framework/http';
import { usePermission } from '@/framework/store';
import './style.less';

type Props = {
  value: FileRow[];
  onChange: (files: FileRow[]) => void;
  multiple?: boolean;
  maxCount?: number;
};

const formatFileSize = (bytes?: number) => {
  if (!bytes || bytes <= 0) return '0 B';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
};

const getBaseName = (filename: string, ext?: string) => {
  if (ext && filename.toLowerCase().endsWith(`.${ext.toLowerCase()}`)) {
    return filename.slice(0, -(ext.length + 1));
  }
  const lastDot = filename.lastIndexOf('.');
  return lastDot > 0 ? filename.substring(0, lastDot) : filename;
};

function OverflowTooltipText({ text, className }: { text: string; className?: string }) {
  const textRef = useRef<HTMLDivElement>(null);
  const [isOverflow, setIsOverflow] = useState(false);

  useEffect(() => {
    const el = textRef.current;
    if (!el) return;
    const check = () => {
      setIsOverflow(el.scrollWidth > el.clientWidth);
    };
    check();
    const observer = new ResizeObserver(check);
    observer.observe(el);
    return () => observer.disconnect();
  }, [text]);

  return (
    <Tooltip title={isOverflow ? text : ''}>
      <div ref={textRef} className={`truncate ${className ?? ''}`}>
        {text}
      </div>
    </Tooltip>
  );
}

export function ImageLibrary({ value, onChange, multiple = true, maxCount = 100 }: Props) {
  const hasPermission = usePermission();
  const [query, setQuery] = useState<FileListRequest>({ page: 1, page_size: 25 });
  const [version, setVersion] = useState(0);
  const [rows, setRows] = useState<FileRow[]>([]);
  const [groupCounts, setGroupCounts] = useState<FileGroupCounts>({
    groups: [],
    total: 0,
    total_storage_size: 0,
    ungrouped: 0,
    ungrouped_storage_size: 0,
  });
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [upload, setUpload] = useState({ done: 0, total: 0 });
  const [saving, setSaving] = useState(false);
  const [groupEditor, setGroupEditor] = useState<FileGroupRow | null>();
  const [deletingGroup, setDeletingGroup] = useState<FileGroupRow>();
  const [renaming, setRenaming] = useState<FileRow>();
  const [moving, setMoving] = useState(false);
  const [targetGroup, setTargetGroup] = useState(0);
  const [previewIndex, setPreviewIndex] = useState<number>();
  const [dragging, setDragging] = useState(false);
  const [groupForm] = Form.useForm<{ name: string }>();
  const [renameForm] = Form.useForm<{ name: string }>();
  const inputRef = useRef<HTMLInputElement>(null);
  const selectedIDs = new Set(value.map((file) => file.id));
  const uploading = upload.total > 0;
  const reload = () => setVersion((current) => current + 1);

  const currentGroupStorageSize = useMemo(() => {
    if (query.group_id === undefined) return groupCounts.total_storage_size;
    if (query.group_id === 0) return groupCounts.ungrouped_storage_size;
    const g = groupCounts.groups.find((item) => item.id === query.group_id);
    return g?.storage_size ?? 0;
  }, [query.group_id, groupCounts]);

  useEffect(() => {
    let active = true;
    const load = async () => {
      setLoading(true);
      const result = await api.api.listSystemFiles(query);
      if (!active) return;
      setLoading(false);
      if (result.code !== 1) { message.error(result.message); return; }
      const count = result.total ?? 0;
      const lastPage = Math.max(1, Math.ceil(count / (query.page_size ?? 25)));
      if ((query.page ?? 1) > lastPage) { setQuery((current) => ({ ...current, page: lastPage })); return; }
      setRows(result.data ?? []);
      setTotal(count);
    };
    void load();
    return () => { active = false; };
  }, [query, version]);

  useEffect(() => {
    let active = true;
    const load = async () => {
      const result = await api.api.getSystemFileGroupCounts();
      if (!active) return;
      if (result.code === 1 && result.data) {
        setGroupCounts(result.data);
      } else {
        message.error(result.message);
      }
    };
    void load();
    return () => { active = false; };
  }, [version]);

  const selectGroup = (group_id?: number) => setQuery((current) => ({ ...current, group_id, page: 1 }));

  const toggle = (file: FileRow) => {
    if (selectedIDs.has(file.id)) {
      onChange(value.filter((item) => item.id !== file.id));
      return;
    }
    if (!multiple) {
      onChange([file]);
      return;
    }
    if (value.length >= maxCount) {
      message.info(`最多选择 ${maxCount} 张图片`);
      return;
    }
    onChange([...value, file]);
  };

  const uploadFiles = async (files: File[]) => {
    if (!files.length) return;
    setUpload({ done: 0, total: files.length });
    let success = 0;
    for (const [index, file] of files.entries()) {
      if (file.size > 10 * 1024 * 1024) {
        message.error(`${file.name}：图片不能超过 10 MB`);
      } else {
        const result = await api.api.uploadSystemFile({ file, group_id: query.group_id ?? 0 });
        if (result.code === 1) success++;
        else message.error(`${file.name}：${result.message || '上传失败'}`);
      }
      setUpload({ done: index + 1, total: files.length });
    }
    setUpload({ done: 0, total: 0 });
    if (success) {
      message.success(`已上传 ${success} 张图片`);
      reload();
    }
  };

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    if (!dragging) setDragging(true);
  };

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.currentTarget.contains(e.relatedTarget as Node)) return;
    setDragging(false);
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setDragging(false);
    if (!hasPermission('system:file:upload')) return;
    const droppedFiles = Array.from(e.dataTransfer.files).filter((file) =>
      /\.(jpe?g|png|gif|webp)$/i.test(file.name)
    );
    if (!droppedFiles.length) {
      message.warning('请拖入 JPG、PNG、GIF、WebP 格式的图片文件');
      return;
    }
    void uploadFiles(droppedFiles);
  };

  const editGroup = (group: FileGroupRow | null) => {
    groupForm.setFieldsValue({ name: group?.name ?? '' });
    setGroupEditor(group);
  };

  const saveGroup = async ({ name }: { name: string }) => {
    setSaving(true);
    const result = await api.api.saveSystemFileGroup({ id: groupEditor?.id ?? 0, name: name.trim() });
    setSaving(false);
    if (result.code !== 1) { message.error(result.message); return; }
    setGroupEditor(undefined);
    reload();
  };

  const deleteGroup = async (group: FileGroupRow) => {
    if (group.count > 0) {
      message.warning(`分组“${group.name}”下仍有 ${group.count} 张图片（共 ${formatFileSize(group.storage_size)}），不允许删除，请先清空或转移组内图片`);
      return;
    }
    setSaving(true);
    const result = await api.api.deleteSystemFileGroup({ id: group.id });
    setSaving(false);
    if (result.code !== 1) { message.error(result.message); return; }
    message.success('分组已删除');
    setDeletingGroup(undefined);
    if (query.group_id === group.id) selectGroup();
    reload();
  };

  const rename = async ({ name }: { name: string }) => {
    if (!renaming) return;
    const trimmed = name.trim();
    if (!trimmed) return;
    const fullName = renaming.ext ? `${trimmed}.${renaming.ext}` : trimmed;
    setSaving(true);
    const result = await api.api.renameSystemFile({ id: renaming.id, name: fullName });
    setSaving(false);
    if (result.code !== 1) { message.error(result.message); return; }
    onChange(value.map((file) => (file.id === renaming.id ? { ...file, name: fullName } : file)));
    setRenaming(undefined);
    reload();
  };

  const move = async () => {
    setSaving(true);
    const result = await api.api.moveSystemFiles({ ids: value.map((file) => file.id), group_id: targetGroup });
    setSaving(false);
    if (result.code !== 1) { message.error(result.message); return; }
    onChange(value.map((file) => ({ ...file, group_id: targetGroup })));
    setMoving(false);
    reload();
  };

  const remove = async () => {
    setSaving(true);
    const result = await api.api.deleteSystemFiles({ ids: value.map((file) => file.id) });
    setSaving(false);
    if (result.code !== 1) { message.error(result.message); return; }
    onChange([]);
    reload();
  };

  const getGroupMenuItems = (group: FileGroupRow): MenuProps['items'] => [
    {
      key: 'edit',
      label: '修改分组名称',
      icon: <EditOutlined />,
      onClick: () => editGroup(group),
    },
    {
      key: 'delete',
      label: '删除分组',
      icon: <DeleteOutlined />,
      danger: true,
      disabled: group.count > 0,
      onClick: () => {
        if (group.count > 0) {
          message.warning(`分组“${group.name}”下仍有 ${group.count} 张图片（共 ${formatFileSize(group.storage_size)}），不允许删除，请先转移或清空组内图片`);
          return;
        }
        setDeletingGroup(group);
      },
    },
  ];

  return (
    <div className="image-library flex h-[500px] bg-white rounded overflow-hidden">
      <aside className="image-library-sidebar w-[170px] shrink-0 flex flex-col bg-[#fafafa] border-r border-gray-100">
        <div className="image-library-sidebar-header px-2.5 py-2 flex justify-between items-center border-b border-gray-100">
          <div className="sidebar-title flex items-center gap-1.5 font-semibold text-xs text-gray-800">
            <FolderOutlined className="sidebar-title-icon text-blue-500" />
            <span>图片分组</span>
          </div>
          {hasPermission('system:file:group') && (
            <Tooltip title="新增分组">
              <Button
                size="small"
                type="text"
                shape="circle"
                aria-label="新增分组"
                className="w-6 h-6 min-w-0 p-0 text-xs"
                icon={<PlusOutlined style={{ fontSize: 11 }} />}
                onClick={() => editGroup(null)}
              />
            </Tooltip>
          )}
        </div>

        <div className="image-library-group-list flex-1 overflow-y-auto p-1.5 flex flex-col gap-0.5">
          <div
            className={`image-library-group-item flex items-center justify-between px-2 py-1.5 rounded cursor-pointer text-xs transition-colors select-none ${query.group_id === undefined ? 'is-active bg-blue-50 text-blue-600 font-medium' : 'text-gray-700 hover:bg-gray-100'}`}
            onClick={() => selectGroup()}
          >
            <span className="group-name truncate">全部图片</span>
            <span className={`group-count text-[11px] px-1.5 py-0.2 rounded-full ${query.group_id === undefined ? 'bg-blue-100 text-blue-600' : 'bg-gray-200 text-gray-500'}`}>
              {groupCounts.total}
            </span>
          </div>

          <div
            className={`image-library-group-item flex items-center justify-between px-2 py-1.5 rounded cursor-pointer text-xs transition-colors select-none ${query.group_id === 0 ? 'is-active bg-blue-50 text-blue-600 font-medium' : 'text-gray-700 hover:bg-gray-100'}`}
            onClick={() => selectGroup(0)}
          >
            <span className="group-name truncate">未分组</span>
            <span className={`group-count text-[11px] px-1.5 py-0.2 rounded-full ${query.group_id === 0 ? 'bg-blue-100 text-blue-600' : 'bg-gray-200 text-gray-500'}`}>
              {groupCounts.ungrouped}
            </span>
          </div>

          {groupCounts.groups.map((group) => {
            const isActive = query.group_id === group.id;
            const itemElement = (
              <div
                className={`image-library-group-item flex items-center justify-between px-2 py-1.5 rounded cursor-pointer text-xs transition-colors select-none ${isActive ? 'is-active bg-blue-50 text-blue-600 font-medium' : 'text-gray-700 hover:bg-gray-100'}`}
                onClick={() => selectGroup(group.id)}
              >
                <span className="group-name truncate">{group.name}</span>
                <span className={`group-count text-[11px] px-1.5 py-0.2 rounded-full ${isActive ? 'bg-blue-100 text-blue-600' : 'bg-gray-200 text-gray-500'}`}>
                  {group.count}
                </span>
              </div>
            );

            if (!hasPermission('system:file:group')) {
              return <div key={group.id}>{itemElement}</div>;
            }

            return (
              <Dropdown
                key={group.id}
                menu={{ items: getGroupMenuItems(group) }}
                trigger={['contextMenu']}
              >
                {itemElement}
              </Dropdown>
            );
          })}
        </div>
      </aside>

      <main className="image-library-main flex-1 min-w-0 flex flex-col p-3 bg-white">
        <div className="image-library-toolbar flex justify-between items-center gap-2 mb-2 shrink-0">
          <div className="toolbar-left flex items-center gap-2 flex-wrap">
            {hasPermission('system:file:upload') && (
              <Button
                type="primary"
                icon={<UploadOutlined />}
                loading={uploading}
                onClick={() => inputRef.current?.click()}
              >
                {uploading ? `上传中 ${upload.done}/${upload.total}` : '上传图片'}
              </Button>
            )}
            <input
              ref={inputRef}
              type="file"
              hidden
              multiple
              accept=".jpg,.jpeg,.png,.gif,.webp"
              onChange={(event) => {
                const files = Array.from(event.target.files ?? []);
                event.target.value = '';
                void uploadFiles(files);
              }}
            />

            {value.length > 0 ? (
              <Space size="small">
                <Tag color="processing" style={{ marginInlineEnd: 0 }}>
                  已选 {value.length}{multiple ? ` / ${maxCount}` : ''}
                </Tag>
                {hasPermission('system:file:update') && (
                  <Button
                    icon={<SwapOutlined />}
                    disabled={saving}
                    onClick={() => {
                      setTargetGroup(query.group_id ?? 0);
                      setMoving(true);
                    }}
                  >
                    移动分组
                  </Button>
                )}
                {hasPermission('system:file:delete') && (
                  <Popconfirm
                    title={`删除选中的 ${value.length} 张图片？`}
                    description="从素材库移除，已被业务引用的图片仍可访问。"
                    onConfirm={remove}
                    okText="删除"
                    cancelText="取消"
                    okButtonProps={{ danger: true }}
                    disabled={!value.length}
                  >
                    <Button
                      danger
                      icon={<DeleteOutlined />}
                      disabled={!value.length}
                      loading={saving && !moving && !renaming}
                    >
                      删除
                    </Button>
                  </Popconfirm>
                )}
                <Button type="link" onClick={() => onChange([])} style={{ padding: '0 4px' }}>
                  清空选择
                </Button>
              </Space>
            ) : (
              <span className="toolbar-hint text-xs text-gray-400 inline-flex items-center">
                <InfoCircleOutlined style={{ marginRight: 4 }} />
                右键分组可管理 · 支持拖拽多图上传
              </span>
            )}
          </div>

          <div className="toolbar-right flex items-center">
            <Input.Search
              placeholder="搜索图片名称"
              allowClear
              style={{ width: 200 }}
              onSearch={(keyword) => setQuery((current) => ({ ...current, keyword, page: 1 }))}
            />
          </div>
        </div>

        <div
          className="image-library-grid-container flex-1 min-h-0 overflow-y-auto relative p-1"
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
        >
          {dragging && (
            <div className="image-library-drop-overlay">
              <InboxOutlined className="drop-icon" />
              <div className="drop-text">释放图片文件以立即上传</div>
              <div className="drop-subtext">支持 JPG、PNG、GIF、WebP，单张 ≤ 10 MB</div>
            </div>
          )}

          <Spin spinning={loading}>
            <div className="image-library-grid">
              {rows.map((file) => {
                const isSelected = selectedIDs.has(file.id);
                return (
                  <article
                    className={`image-library-card${isSelected ? ' is-selected' : ''}`}
                    key={file.id}
                    onClick={() => toggle(file)}
                  >
                    <div className="image-card-thumb">
                      <img src={file.url} alt={file.name} loading="lazy" />

                      <div className="image-card-badge">
                        {isSelected ? (
                          <CheckCircleFilled className="image-card-checkbox checked" />
                        ) : (
                          <div className="image-card-checkbox unchecked" />
                        )}
                      </div>

                      <div className="image-card-actions">
                        <Tooltip title="预览大图">
                          <button
                            type="button"
                            className="card-action-btn"
                            aria-label={`预览${file.name}`}
                            onClick={(e) => {
                              e.stopPropagation();
                              const index = rows.findIndex((item) => item.id === file.id);
                              setPreviewIndex(index >= 0 ? index : 0);
                            }}
                          >
                            <EyeOutlined />
                          </button>
                        </Tooltip>

                        {hasPermission('system:file:update') && (
                          <Tooltip title="重命名">
                            <button
                              type="button"
                              className="card-action-btn"
                              aria-label={`重命名${file.name}`}
                              onClick={(e) => {
                                e.stopPropagation();
                                renameForm.setFieldsValue({ name: getBaseName(file.name, file.ext) });
                                setRenaming(file);
                              }}
                            >
                              <EditOutlined />
                            </button>
                          </Tooltip>
                        )}
                      </div>
                    </div>

                    <div className="image-card-info px-1.5 py-1 bg-white border-t border-gray-100">
                      <OverflowTooltipText text={file.name} className="image-card-name text-xs text-gray-700 leading-tight" />
                    </div>
                  </article>
                );
              })}
            </div>

            {!rows.length && !loading && (
              <div className="image-library-empty py-16 flex justify-center items-center">
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={query.keyword ? '未搜索到匹配的图片' : '当前分组下暂无图片'}
                >
                  {hasPermission('system:file:upload') && !query.keyword && (
                    <Button
                      type="dashed"
                      icon={<UploadOutlined />}
                      onClick={() => inputRef.current?.click()}
                    >
                      立即上传图片
                    </Button>
                  )}
                </Empty>
              </div>
            )}
          </Spin>
        </div>

        <div className="image-library-pagination pt-2 mt-1 border-t border-gray-100 flex justify-end items-center shrink-0">
          <Pagination
            current={query.page}
            pageSize={query.page_size}
            total={total}
            showSizeChanger
            size="small"
            pageSizeOptions={[25, 50, 100]}
            showTotal={(count) =>
              query.keyword
                ? `共 ${count} 张图片`
                : `共 ${count} 张图片 · 占用 ${formatFileSize(currentGroupStorageSize)}`
            }
            onChange={(page, page_size) =>
              setQuery((current) => ({
                ...current,
                page: page_size === current.page_size ? page : 1,
                page_size,
              }))
            }
          />
        </div>
      </main>

      {previewIndex !== undefined && (
        <Image.PreviewGroup
          items={rows.map((r) => r.url)}
          preview={{
            open: true,
            current: previewIndex,
            onChange: (current) => setPreviewIndex(current),
            onOpenChange: (open) => {
              if (!open) setPreviewIndex(undefined);
            },
          }}
        />
      )}

      <Modal
        title={groupEditor ? '修改分组名称' : '新增分组'}
        open={groupEditor !== undefined}
        onCancel={() => setGroupEditor(undefined)}
        onOk={() => groupForm.submit()}
        confirmLoading={saving}
        cancelButtonProps={{ disabled: saving }}
      >
        <Form form={groupForm} onFinish={saveGroup} layout="vertical">
          <Form.Item name="name" label="分组名称" rules={[{ required: true, whitespace: true, max: 64 }]}>
            <Input maxLength={64} placeholder="请输入分组名称" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`确定删除分组“${deletingGroup?.name}”？`}
        open={!!deletingGroup}
        onCancel={() => setDeletingGroup(undefined)}
        onOk={() => deletingGroup && deleteGroup(deletingGroup)}
        confirmLoading={saving}
        okButtonProps={{ danger: true }}
        okText="删除"
        cancelText="取消"
      >
        <p>删除后分组将无法恢复，确认删除吗？</p>
      </Modal>

      <Modal
        title="重命名图片"
        open={!!renaming}
        onCancel={() => setRenaming(undefined)}
        onOk={() => renameForm.submit()}
        confirmLoading={saving}
        cancelButtonProps={{ disabled: saving }}
      >
        <Form form={renameForm} onFinish={rename} layout="vertical">
          <Form.Item
            name="name"
            label="图片名称"
            rules={[{ required: true, whitespace: true, max: 200, message: '请输入图片名称' }]}
            extra={renaming?.ext ? `格式后缀固定为 .${renaming.ext}（无需输入）` : undefined}
          >
            <Input
              maxLength={200}
              placeholder="请输入图片名称"
              addonAfter={renaming?.ext ? `.${renaming.ext}` : undefined}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`移动 ${value.length} 张图片到`}
        open={moving}
        onCancel={() => setMoving(false)}
        onOk={move}
        confirmLoading={saving}
        cancelButtonProps={{ disabled: saving }}
      >
        <Select
          value={targetGroup}
          onChange={setTargetGroup}
          style={{ width: '100%' }}
          options={[{ value: 0, label: '未分组' }, ...groupCounts.groups.map((group) => ({ value: group.id, label: group.name }))]}
        />
      </Modal>
    </div>
  );
}
