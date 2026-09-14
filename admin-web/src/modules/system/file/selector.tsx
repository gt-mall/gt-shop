import { useState } from 'react';
import { PictureOutlined } from '@ant-design/icons';
import { Modal } from 'antd';
import type { FileRow } from '@/framework/http';
import { ImageLibrary } from './library';

type Props = {
  value?: FileRow[];
  multiple?: boolean;
  maxCount?: number;
  onConfirm: (files: FileRow[]) => void;
  onCancel: () => void;
};

/** 按需挂载的图片选择弹窗；每次打开从 value 恢复选择。 */
export function ImageSelector({ value = [], multiple = false, maxCount = 20, onConfirm, onCancel }: Props) {
  const [selected, setSelected] = useState(value);
  return (
    <Modal
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, fontWeight: 500 }}>
          <PictureOutlined style={{ color: 'var(--ant-color-primary, #4338CA)', fontSize: 16 }} />
          <span>{multiple ? `选择图片（最多可选 ${maxCount} 张）` : '选择图片'}</span>
        </div>
      }
      open
      width={960}
      centered
      destroyOnClose
      className="image-selector-modal"
      styles={{
        header: {
          padding: '6px 14px',
          margin: 0,
          borderBottom: '1px solid var(--ant-color-border-secondary, #f0f0f0)',
        },
        body: { padding: 0 },
        footer: {
          padding: '6px 14px',
          margin: 0,
          borderTop: '1px solid var(--ant-color-border-secondary, #f0f0f0)',
        },
      }}
      onCancel={onCancel}
      onOk={() => onConfirm(selected)}
      okText={selected.length > 0 ? `确定选择 (${selected.length})` : '确定'}
      okButtonProps={{  disabled: !selected.length }}
      cancelButtonProps={{  }}
      cancelText="取消"
    >
      <ImageLibrary
        value={selected}
        onChange={setSelected}
        multiple={multiple}
        maxCount={maxCount}
        fixedFileType="image"
      />
    </Modal>
  );
}
