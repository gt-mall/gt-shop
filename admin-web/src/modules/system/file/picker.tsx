import { useEffect, useState } from 'react';
import {
  DeleteOutlined,
  EyeOutlined,
  PlusOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import { Image, Spin, Tooltip, message } from 'antd';
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  SortableContext,
  arrayMove,
  rectSortingStrategy,
  sortableKeyboardCoordinates,
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

import { api, type FileRow } from '@/framework/http';
import { usePermission } from '@/framework/store';
import { ImageSelector } from './selector';
import './style.less';

type CommonProps = {
  disabled?: boolean;
  maxCount?: number;
  /** 是否支持拖拽排序（仅在 multiple 为 true 时生效，默认为 true） */
  sortable?: boolean;
};

export type ImagePickerProps = CommonProps & (
  | { multiple?: false; value?: number; onChange?: (id: number | undefined, file?: FileRow) => void }
  | { multiple: true; value?: number[]; onChange?: (ids: number[], files: FileRow[]) => void }
);

interface SortableImageCardProps {
  file: FileRow;
  idx: number;
  multiple?: boolean;
  disabled?: boolean;
  canSort?: boolean;
  canSwap?: boolean;
  onSwap: () => void;
  onPreview: (url: string) => void;
  onRemove: (id: number) => void;
}

function SortableImageCard({
  file,
  idx,
  multiple,
  disabled,
  canSort,
  canSwap,
  onSwap,
  onPreview,
  onRemove,
}: SortableImageCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: file.id,
    disabled: !canSort,
  });

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 99 : undefined,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`image-picker-card ${isDragging ? 'is-dragging' : ''} ${canSort ? 'is-sortable' : ''}`}
      {...attributes}
      {...listeners}
    >
      {multiple && <span className="image-picker-index-badge">{idx + 1}</span>}
      <img src={file.url} alt={file.name} className="image-picker-img" />
      <div className="image-picker-mask">
        {!multiple && !disabled && canSwap && (
          <Tooltip title="更换图片">
            <button
              type="button"
              className="picker-mask-btn"
              aria-label="更换图片"
              onPointerDown={(e) => e.stopPropagation()}
              onClick={(e) => {
                e.stopPropagation();
                onSwap();
              }}
            >
              <SwapOutlined />
            </button>
          </Tooltip>
        )}
        <Tooltip title="预览">
          <button
            type="button"
            className="picker-mask-btn"
            aria-label={`预览${file.name}`}
            onPointerDown={(e) => e.stopPropagation()}
            onClick={(e) => {
              e.stopPropagation();
              onPreview(file.url);
            }}
          >
            <EyeOutlined />
          </button>
        </Tooltip>
        {!disabled && (
          <Tooltip title="移除">
            <button
              type="button"
              className="picker-mask-btn delete"
              aria-label={`移除${file.name}`}
              onPointerDown={(e) => e.stopPropagation()}
              onClick={(e) => {
                e.stopPropagation();
                onRemove(file.id);
              }}
            >
              <DeleteOutlined />
            </button>
          </Tooltip>
        )}
      </div>
    </div>
  );
}

/** Form.Item 可直接绑定；value 保存 file_id，预览地址由接口动态解析。多图模式下原生支持拖拽排序。 */
export function ImagePicker(props: ImagePickerProps) {
  const hasPermission = usePermission();
  const [files, setFiles] = useState<FileRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [previewUrl, setPreviewUrl] = useState<string>();
  const ids = props.multiple ? props.value ?? [] : props.value ? [props.value] : [];
  const idKey = ids.join(',');

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 5,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  useEffect(() => {
    let active = true;
    const load = async () => {
      if (!idKey) {
        setFiles([]);
        setLoading(false);
        return;
      }
      setLoading(true);
      const result = await api.api.resolveSystemFiles({ ids: idKey.split(',').map(Number) });
      if (!active) return;
      setLoading(false);
      if (result.code === 1) {
        const resolvedList = result.data ?? [];
        const idOrder = idKey.split(',').map(Number);
        const map = new Map<number, FileRow>();
        resolvedList.forEach((f) => map.set(f.id, f));
        const ordered = idOrder.map((id) => map.get(id)).filter(Boolean) as FileRow[];
        setFiles(ordered);
      } else {
        message.error(result.message);
      }
    };
    void load();
    return () => {
      active = false;
    };
  }, [idKey]);

  const change = (selected: FileRow[]) => {
    if (props.multiple) props.onChange?.(selected.map((file) => file.id), selected);
    else props.onChange?.(selected[0]?.id, selected[0]);
    setOpen(false);
  };

  const removeFile = (id: number) => {
    const next = files.filter((item) => item.id !== id);
    change(next);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = files.findIndex((item) => item.id === active.id);
    const newIndex = files.findIndex((item) => item.id === over.id);
    if (oldIndex === -1 || newIndex === -1) return;

    const nextFiles = arrayMove(files, oldIndex, newIndex);
    setFiles(nextFiles);
    if (props.multiple) {
      props.onChange?.(
        nextFiles.map((file) => file.id),
        nextFiles,
      );
    }
  };

  const canAdd =
    !props.disabled &&
    (!props.multiple ? files.length === 0 : !props.maxCount || files.length < props.maxCount);

  const isSortable =
    Boolean(props.multiple) &&
    props.sortable !== false &&
    !props.disabled &&
    files.length > 1;

  const cardListDom = files.map((file, idx) => (
    <SortableImageCard
      key={file.id}
      file={file}
      idx={idx}
      multiple={props.multiple}
      disabled={props.disabled}
      canSort={isSortable}
      canSwap={hasPermission('system:file:list')}
      onSwap={() => setOpen(true)}
      onPreview={(url) => setPreviewUrl(url)}
      onRemove={(id) => removeFile(id)}
    />
  ));

  const uploadTriggerDom = canAdd && (
    <div
      className={`image-picker-card upload-trigger ${props.disabled ? 'disabled' : ''}`}
      onClick={() => !props.disabled && hasPermission('system:file:list') && setOpen(true)}
    >
      <PlusOutlined className="trigger-icon" />
      <span className="trigger-text">
        {props.multiple ? (files.length ? '添加图片' : '选择图片') : '选择图片'}
      </span>
    </div>
  );

  return (
    <Spin spinning={loading}>
      {isSortable ? (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={files.map((f) => f.id)}
            strategy={rectSortingStrategy}
          >
            <div className="image-picker-container">
              {cardListDom}
              {uploadTriggerDom}
            </div>
          </SortableContext>
        </DndContext>
      ) : (
        <div className="image-picker-container">
          {cardListDom}
          {uploadTriggerDom}
        </div>
      )}

      {previewUrl && (
        <Image.PreviewGroup
          preview={{
            open: !!previewUrl,
            current: Math.max(0, files.findIndex((f) => f.url === previewUrl)),
            onOpenChange: (openVal) => {
              if (!openVal) setPreviewUrl(undefined);
            },
          }}
          items={files.map((f) => f.url)}
        >
          <div style={{ display: 'none' }} />
        </Image.PreviewGroup>
      )}

      {open && (
        <ImageSelector
          value={files}
          multiple={props.multiple}
          maxCount={props.maxCount}
          onConfirm={change}
          onCancel={() => setOpen(false)}
        />
      )}
    </Spin>
  );
}
