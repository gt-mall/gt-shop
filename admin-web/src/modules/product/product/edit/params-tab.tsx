import { Button, Divider } from 'antd';

import type { ProductParam } from '@/framework/http';
import { ParamEditor } from '../param-editor';

export interface ParamsTabProps {
  params: ProductParam[];
  onChange: (params: ProductParam[]) => void;
  disabled: boolean;
  onPrev: () => void;
  onNext: () => void;
}

export function ParamsTab({
  params,
  onChange,
  disabled,
  onPrev,
  onNext,
}: ParamsTabProps) {
  return (
    <div  className=' px-4 py-4 '>
      <ParamEditor params={params} onChange={onChange} disabled={disabled} />
      <Divider style={{ margin: '24px 0 16px' }} />
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <Button onClick={onPrev}>上一步：规格与 SKU</Button>
        <Button type="primary" onClick={onNext}>
          下一步：编辑商品详情
        </Button>
      </div>
    </div>
  );
}
