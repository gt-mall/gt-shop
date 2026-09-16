import { Button, Divider } from "antd";
import { CheckCard } from "@ant-design/pro-components";

import type { ProductSKURequest, ProductSpec } from "@/framework/http";
import { SKUTable } from "../sku-table";
import { SpecEditor } from "../spec-editor";

export interface SKUTabProps {
  specMode: "single" | "multi";
  hasSavedSKUs: boolean;
  onSpecModeChange: (mode: "single" | "multi") => void;
  specs: ProductSpec[];
  onSpecsChange: (specs: ProductSpec[]) => void;
  skus: ProductSKURequest[];
  onSKUsChange: (skus: ProductSKURequest[]) => void;
  disabled: boolean;
  onPrev: () => void;
  onNext: () => void;
}

const SingleSpecIcon = () => (
  <svg
    width="28"
    height="28"
    viewBox="0 0 32 32"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    style={{ display: "block" }}
  >
    <path
      d="M16 4.5L26 10.2V21.8L16 27.5L6 21.8V10.2Z"
      stroke="#475569"
      strokeWidth="2"
      strokeLinejoin="round"
    />
    <path
      d="M16 16L6 10.2M16 16L26 10.2M16 16V27.5"
      stroke="#475569"
      strokeWidth="2"
      strokeLinejoin="round"
    />
  </svg>
);

const MultiSpecIcon = () => (
  <svg
    width="28"
    height="28"
    viewBox="0 0 32 32"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    style={{ display: "block" }}
  >
    <path
      d="M16 3.3L20.5 5.9V11.1L16 13.7L11.5 11.1V5.9Z"
      stroke="#4f46e5"
      strokeWidth="1.8"
      strokeLinejoin="round"
    />
    <path
      d="M16 8.5L11.5 5.9M16 8.5L20.5 5.9M16 8.5V13.7"
      stroke="#4f46e5"
      strokeWidth="1.8"
      strokeLinejoin="round"
    />
    <path
      d="M10 14.3L14.5 16.9V22.1L10 24.7L5.5 22.1V16.9Z"
      stroke="#4f46e5"
      strokeWidth="1.8"
      strokeLinejoin="round"
    />
    <path
      d="M10 19.5L5.5 16.9M10 19.5L14.5 16.9M10 19.5V24.7"
      stroke="#4f46e5"
      strokeWidth="1.8"
      strokeLinejoin="round"
    />
    <path
      d="M22 14.3L26.5 16.9V22.1L22 24.7L17.5 22.1V16.9Z"
      stroke="#4f46e5"
      strokeWidth="1.8"
      strokeLinejoin="round"
    />
    <path
      d="M22 19.5L17.5 16.9M22 19.5L26.5 16.9M22 19.5V24.7"
      stroke="#4f46e5"
      strokeWidth="1.8"
      strokeLinejoin="round"
    />
  </svg>
);

export function SKUTab({
  specMode,
  hasSavedSKUs,
  onSpecModeChange,
  specs,
  onSpecsChange,
  skus,
  onSKUsChange,
  disabled,
  onPrev,
  onNext,
}: SKUTabProps) {
  return (
    <div className="px-4 pt-6 pb-4">
      <div style={{ marginBottom: 20 }}>
        <Divider titlePlacement="left">规格模式</Divider>
        <CheckCard.Group
          value={specMode}
          disabled={disabled || hasSavedSKUs}
          onChange={(val) => {
            if (val) onSpecModeChange(val as "single" | "multi");
          }}
          style={{ display: "flex", gap: 16, width: "100%" }}
        >
          <CheckCard
            title="单规格"
            description="适用于普通商品"
            value="single"
            avatar={<SingleSpecIcon />}
            style={{ width: 240, margin: 0, borderRadius: 8 }}
          />
          <CheckCard
            title="多规格"
            description="如颜色、尺寸等"
            value="multi"
            avatar={<MultiSpecIcon />}
            style={{ width: 240, margin: 0, borderRadius: 8 }}
          />
        </CheckCard.Group>
      </div>

      {specMode === "multi" ? (
        <div>
          <SpecEditor
            specs={specs}
            skus={skus}
            disabled={disabled}
            onSpecsChange={onSpecsChange}
            onGenerateSKUs={onSKUsChange}
          />
          <Divider titlePlacement="left" >
            SKU 价格与编码明细
          </Divider>
          <SKUTable
            specs={specs}
            skus={skus}
            disabled={disabled}
            onSKUsChange={onSKUsChange}
          />
        </div>
      ) : (
        <div>
          <SKUTable
            specs={[]}
            skus={skus}
            disabled={disabled}
            onSKUsChange={onSKUsChange}
          />
        </div>
      )}

      <Divider style={{ margin: "24px 0 16px" }} />
      <div style={{ display: "flex", justifyContent: "space-between" }}>
        <Button onClick={onPrev}>上一步：基本信息</Button>
        <Button type="primary" onClick={onNext}>
          下一步：配置商品参数
        </Button>
      </div>
    </div>
  );
}
