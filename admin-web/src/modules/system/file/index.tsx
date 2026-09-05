import { useState } from 'react';
import { Card } from 'antd';
import type { FileRow } from '@/framework/http';
import { ImageLibrary } from './library';

export { ImagePicker } from './picker';
export type { ImagePickerProps } from './picker';
export { ImageSelector } from './selector';

export default function FilePage() {
  const [selected, setSelected] = useState<FileRow[]>([]);
  return <Card title="图片素材"><ImageLibrary value={selected} onChange={setSelected} /></Card>;
}
