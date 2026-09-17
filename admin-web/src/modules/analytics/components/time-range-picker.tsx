import React from 'react';
import { DatePicker, Radio, Space } from 'antd';
import type { RadioChangeEvent } from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

export interface TimeRangePickerProps {
  value?: string;
  startTime?: string;
  endTime?: string;
  onChange: (timeRange: string, startTime?: string, endTime?: string) => void;
  size?: 'small' | 'middle' | 'large';
}

const RANGE_OPTIONS = [
  { label: '今日', value: 'today' },
  { label: '昨日', value: 'yesterday' },
  { label: '近7天', value: 'last_7_days' },
  { label: '近30天', value: 'last_30_days' },
  { label: '本月', value: 'this_month' },
  { label: '上月', value: 'last_month' },
  { label: '自定义', value: 'custom' },
];

export const TimeRangePicker: React.FC<TimeRangePickerProps> = ({
  value = 'today',
  startTime,
  endTime,
  onChange,
  size = 'middle',
}) => {
  const handleRadioChange = (e: RadioChangeEvent) => {
    const nextVal = e.target.value as string;
    if (nextVal !== 'custom') {
      onChange(nextVal, undefined, undefined);
    } else {
      // Default custom range: last 7 days
      const s = dayjs().subtract(6, 'day').format('YYYY-MM-DD 00:00:00');
      const eDate = dayjs().format('YYYY-MM-DD 23:59:59');
      onChange(nextVal, s, eDate);
    }
  };

  const handleRangePickerChange = (dates: [Dayjs | null, Dayjs | null] | null) => {
    if (!dates || !dates[0] || !dates[1]) {
      onChange('custom', undefined, undefined);
      return;
    }
    const s = dates[0].startOf('day').format('YYYY-MM-DD HH:mm:ss');
    const e = dates[1].endOf('day').format('YYYY-MM-DD HH:mm:ss');
    onChange('custom', s, e);
  };

  const pickerValue: [Dayjs, Dayjs] | null =
    startTime && endTime ? [dayjs(startTime), dayjs(endTime)] : null;

  return (
    <Space wrap size={12} align="center">
      <Radio.Group
        value={value}
        onChange={handleRadioChange}
        optionType="button"
        buttonStyle="solid"
        size={size}
        options={RANGE_OPTIONS}
      />
      {value === 'custom' && (
        <DatePicker.RangePicker
          size={size}
          value={pickerValue}
          onChange={handleRangePickerChange}
          allowClear={false}
          style={{ width: 260 }}
        />
      )}
    </Space>
  );
};
