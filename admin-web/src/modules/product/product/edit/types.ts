export type ProductFormData = {
  name: string;
  subtitle?: string;
  category_id?: number;
  main_image_id?: number | null;
  sort?: number;
  status: number;
  virtual_sales?: number;
  detail?: string;
  freight_type?: number;
  freight_fee?: string;
  freight_template_id?: number;
};
