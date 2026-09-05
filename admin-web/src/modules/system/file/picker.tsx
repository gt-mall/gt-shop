import { useEffect, useState } from 'react';
import { DeleteOutlined, EyeOutlined, PlusOutlined, SwapOutlined } from '@ant-design/icons';
import { Image, message, Spin, Tooltip } from 'antd';
import { api, type FileRow } from '@/framework/http';
import { usePermission } from '@/framework/store';
import { ImageSelector } from './selector';
import './style.less';

type CommonProps = { disabled?: boolean; maxCount?: number };
export type ImagePickerProps = CommonProps & (
  | { multiple?: false; value?: number; onChange?: (id: number | undefined, file?: FileRow) => void }
  | { multiple: true; value?: number[]; onChange?: (ids: number[], files: FileRow[]) => void }
);

/** Form.Item 可直接绑定；value 保存 file_id，预览地址由接口动态解析。 */
export function ImagePicker(props: ImagePickerProps) {
  const hasPermission = usePermission();
  const [files, setFiles] = useState<FileRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [previewUrl, setPreviewUrl] = useState<string>();
  const ids = props.multiple ? props.value ?? [] : props.value ? [props.value] : [];
  const idKey = ids.join(',');

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
      if (result.code === 1) setFiles(result.data ?? []);
      else message.error(result.message);
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

  const canAdd =
    !props.disabled &&
    (!props.multiple ? files.length === 0 : !props.maxCount || files.length < props.maxCount);

  return (
    <Spin spinning={loading}>
      <div className="image-picker-container">
        {files.map((file) => (
          <div key={file.id} className="image-picker-card">
            <img src={file.url} alt={file.name} className="image-picker-img" />
            <div className="image-picker-mask">
              {!props.multiple && !props.disabled && hasPermission('system:file:list') && (
                <Tooltip title="更换图片">
                  <button
                    type="button"
                    className="picker-mask-btn"
                    aria-label="更换图片"
                    onClick={() => setOpen(true)}
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
                  onClick={() => setPreviewUrl(file.url)}
                >
                  <EyeOutlined />
                </button>
              </Tooltip>
              {!props.disabled && (
                <Tooltip title="移除">
                  <button
                    type="button"
                    className="picker-mask-btn delete"
                    aria-label={`移除${file.name}`}
                    onClick={() => removeFile(file.id)}
                  >
                    <DeleteOutlined />
                  </button>
                </Tooltip>
              )}
            </div>
          </div>
        ))}

        {canAdd && (
          <div
            className={`image-picker-card upload-trigger ${props.disabled ? 'disabled' : ''}`}
            onClick={() => !props.disabled && hasPermission('system:file:list') && setOpen(true)}
          >
            <PlusOutlined className="trigger-icon" />
            <span className="trigger-text">
              {props.multiple ? (files.length ? '添加图片' : '选择图片') : '选择图片'}
            </span>
          </div>
        )}
      </div>

      {previewUrl && (
        <Image
          style={{ display: 'none' }}
          preview={{
            open: !!previewUrl,
            src: previewUrl,
            onOpenChange: (open) => {
              if (!open) setPreviewUrl(undefined);
            },
          }}
        />
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
