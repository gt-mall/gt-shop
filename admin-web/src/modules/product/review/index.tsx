import { useCallback, useEffect, useState } from 'react';
import {
  Button,
  DatePicker,
  Descriptions,
  Drawer,
  Form,
  Image,
  Input,
  Modal,
  Pagination,
  Popconfirm,
  Rate,
  Select,
  Space,
  Table,
  Tag,
  Timeline,
  Tooltip,
  Typography,
  message,
  theme,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  CommentOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  ReloadOutlined,
  SearchOutlined,
  StarOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';

import { api } from '@/framework/http';
import type {
  ReviewDetailResponse,
  ReviewListItemResponse,
  ReviewLogResponse,
} from '@/framework/http';
import { usePermission } from '@/framework/store';
import { ContentCard, PageHeader, adminListVars } from '@/shared/admin-ui';

const { RangePicker } = DatePicker;
const { Paragraph, Text } = Typography;

export default function ProductReviewPage() {
  const { token } = theme.useToken();
  const hasPermission = usePermission();

  const canAudit = hasPermission('product:review:audit');
  const canReply = hasPermission('product:review:reply');

  // 列表状态
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ReviewListItemResponse[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // 筛选条件状态
  const [statusFilter, setStatusFilter] = useState<number | undefined>(undefined);
  const [ratingFilter, setRatingFilter] = useState<number | undefined>(undefined);
  const [hasImageFilter, setHasImageFilter] = useState<number | undefined>(undefined);
  const [keyword, setKeyword] = useState('');
  const [keywordInput, setKeywordInput] = useState('');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null] | null>(null);

  // 详情 Drawer
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerLoading, setDrawerLoading] = useState(false);
  const [detailData, setDetailData] = useState<ReviewDetailResponse | null>(null);

  // 驳回 Modal
  const [rejectModalOpen, setRejectModalOpen] = useState(false);
  const [rejectTarget, setRejectTarget] = useState<ReviewListItemResponse | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  const [rejectSubmitting, setRejectSubmitting] = useState(false);

  // 商家回复 Modal
  const [replyModalOpen, setReplyModalOpen] = useState(false);
  const [replyTarget, setReplyTarget] = useState<ReviewListItemResponse | null>(null);
  const [replyContent, setReplyContent] = useState('');
  const [replySubmitting, setReplySubmitting] = useState(false);

  const loadData = useCallback(
    async (
      currentPage = page,
      size = pageSize,
      currentStatus = statusFilter,
      currentRating = ratingFilter,
      currentHasImage = hasImageFilter,
      currentKeyword = keyword,
      currentDates = dateRange,
    ) => {
      setLoading(true);
      const res = await api.api.listProductReviews({
        page: currentPage,
        page_size: size,
        status: currentStatus,
        rating: currentRating,
        has_image: currentHasImage,
        keyword: currentKeyword.trim() || undefined,
        start_date: currentDates && currentDates[0] ? currentDates[0].format('YYYY-MM-DD') : undefined,
        end_date: currentDates && currentDates[1] ? currentDates[1].format('YYYY-MM-DD') : undefined,
      });
      setLoading(false);

      if (res.code === 1 && res.data) {
        setList(res.data);
        setTotal(res.total || 0);
      } else {
        message.error(res.message || '加载商品评价列表失败');
      }
    },
    [page, pageSize, statusFilter, ratingFilter, hasImageFilter, keyword, dateRange],
  );

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSearch = () => {
    setKeyword(keywordInput);
    setPage(1);
    void loadData(1, pageSize, statusFilter, ratingFilter, hasImageFilter, keywordInput, dateRange);
  };

  const handleReset = () => {
    setKeywordInput('');
    setKeyword('');
    setStatusFilter(undefined);
    setRatingFilter(undefined);
    setHasImageFilter(undefined);
    setDateRange(null);
    setPage(1);
    void loadData(1, pageSize, undefined, undefined, undefined, '', null);
  };

  // 查看详情
  const handleOpenDetail = async (record: ReviewListItemResponse) => {
    if (!record.id) return;
    setDrawerOpen(true);
    setDrawerLoading(true);
    const res = await api.api.getProductReviewDetail({ id: record.id });
    setDrawerLoading(false);
    if (res.code === 1 && res.data) {
      setDetailData(res.data);
    } else {
      message.error(res.message || '获取评价详情失败');
    }
  };

  // 审核：通过
  const handleApprove = async (record: ReviewListItemResponse) => {
    if (!record.id || record.status === undefined) return;
    const res = await api.api.auditProductReview({
      id: record.id,
      action: 'approve',
      expected_status: record.status,
    });
    if (res.code === 1) {
      message.success('审核已通过，评价已公开展示');
      void loadData();
    } else {
      message.error(res.message || '审核操作失败');
      void loadData();
    }
  };

  // 审核：打开驳回弹窗
  const handleOpenReject = (record: ReviewListItemResponse) => {
    setRejectTarget(record);
    setRejectReason('');
    setRejectModalOpen(true);
  };

  // 审核：提交驳回
  const handleConfirmReject = async () => {
    if (!rejectTarget?.id || rejectTarget.status === undefined) return;
    if (!rejectReason.trim()) {
      message.warning('请填写驳回原因');
      return;
    }
    setRejectSubmitting(true);
    const res = await api.api.auditProductReview({
      id: rejectTarget.id,
      action: 'reject',
      expected_status: rejectTarget.status,
      reason: rejectReason.trim(),
    });
    setRejectSubmitting(false);

    if (res.code === 1) {
      message.success('已驳回该评价');
      setRejectModalOpen(false);
      setRejectTarget(null);
      void loadData();
    } else {
      message.error(res.message || '驳回操作失败');
      void loadData();
    }
  };

  // 审核：隐藏评价
  const handleHide = async (record: ReviewListItemResponse) => {
    if (!record.id || record.status === undefined) return;
    const res = await api.api.auditProductReview({
      id: record.id,
      action: 'hide',
      expected_status: record.status,
    });
    if (res.code === 1) {
      message.success('评价已隐藏');
      void loadData();
    } else {
      message.error(res.message || '隐藏操作失败');
      void loadData();
    }
  };

  // 审核：恢复公开
  const handleRestore = async (record: ReviewListItemResponse) => {
    if (!record.id || record.status === undefined) return;
    const res = await api.api.auditProductReview({
      id: record.id,
      action: 'restore',
      expected_status: record.status,
    });
    if (res.code === 1) {
      message.success('评价已恢复公开');
      void loadData();
    } else {
      message.error(res.message || '恢复操作失败');
      void loadData();
    }
  };

  // 回复：打开回复弹窗
  const handleOpenReply = (record: ReviewListItemResponse) => {
    setReplyTarget(record);
    setReplyContent(record.reply_content || '');
    setReplyModalOpen(true);
  };

  // 回复：提交回复
  const handleConfirmReply = async () => {
    if (!replyTarget?.id) return;
    if (!replyContent.trim()) {
      message.warning('回复内容不能为空');
      return;
    }
    if (replyContent.trim().length > 500) {
      message.warning('回复内容不能超过500字');
      return;
    }

    setReplySubmitting(true);
    const res = await api.api.replyProductReview({
      id: replyTarget.id,
      content: replyContent.trim(),
    });
    setReplySubmitting(false);

    if (res.code === 1) {
      message.success('已回复该评价');
      setReplyModalOpen(false);
      setReplyTarget(null);
      void loadData();
    } else {
      message.error(res.message || '回复失败');
    }
  };

  // 状态 Tag 映射
  const renderStatusTag = (status?: number) => {
    switch (status) {
      case 10:
        return <Tag color="orange">待审核</Tag>;
      case 20:
        return <Tag color="green">已公开</Tag>;
      case 30:
        return <Tag color="red">已驳回</Tag>;
      case 40:
        return <Tag color="default">已隐藏</Tag>;
      default:
        return <Tag>未知</Tag>;
    }
  };

  const columns: ColumnsType<ReviewListItemResponse> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: '商品信息',
      dataIndex: 'product_name',
      width: 200,
      render: (text: string, record) => (
        <div>
          <div className="font-medium text-gray-800 line-clamp-2" title={text}>
            {text || '—'}
          </div>
          {record.sku_spec_values && (
            <div className="text-xs text-gray-400 mt-1">
              规格: {record.sku_spec_values}
            </div>
          )}
          <div className="text-xs text-gray-400">商品ID: {record.product_id}</div>
        </div>
      ),
    },
    {
      title: '会员',
      dataIndex: 'member_nickname',
      width: 150,
      render: (text: string, record) => (
        <div>
          <div className="flex items-center gap-1">
            <span className="font-medium">{text || '—'}</span>
            {record.is_anonymous === 1 && (
              <Tag color="cyan" className="text-xs scale-90">
                匿名
              </Tag>
            )}
          </div>
          <div className="text-xs text-gray-400">{record.member_mobile || '—'}</div>
        </div>
      ),
    },
    {
      title: '评分',
      dataIndex: 'rating',
      width: 140,
      render: (val: number) => (
        <Rate disabled value={val || 5} style={{ fontSize: 14, color: '#ff9800' }} />
      ),
    },
    {
      title: '评价内容',
      dataIndex: 'content',
      ellipsis: true,
      render: (text: string, record) => (
        <div>
          <Paragraph ellipsis={{ rows: 2, tooltip: text }} className="!mb-0 text-sm">
            {text || '—'}
          </Paragraph>
          {record.images && record.images.length > 0 && (
            <div className="mt-2" onClick={(e) => e.stopPropagation()}>
              <Image.PreviewGroup>
                <Space size={4} wrap>
                  {record.images.map((img, idx) => (
                    <Image
                      key={img.id || idx}
                      src={img.url}
                      width={40}
                      height={40}
                      className="rounded object-cover"
                    />
                  ))}
                </Space>
              </Image.PreviewGroup>
            </div>
          )}
          {record.reply_content && (
            <div className="mt-1 text-xs text-red-600 bg-red-50 p-1 rounded">
              掌柜回复: {record.reply_content}
            </div>
          )}
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (val?: number) => renderStatusTag(val),
    },
    {
      title: '评价时间',
      dataIndex: 'created_at',
      width: 150,
      render: (val?: string) => (val ? dayjs(val).format('YYYY-MM-DD HH:mm') : '—'),
    },
    {
      title: '操作',
      key: 'actions',
      fixed: 'right',
      width: 200,
      render: (_, record) => (
        <Space size={6} wrap>
          <Button type="link" size="small" onClick={() => handleOpenDetail(record)}>
            详情
          </Button>

          {canAudit && record.status === 10 && (
            <>
              <Popconfirm
                title="确定审核通过该评价吗？"
                description="通过后评价将在商品详情公开展示"
                onConfirm={() => handleApprove(record)}
              >
                <Button type="link" size="small" className="!text-green-600">
                  通过
                </Button>
              </Popconfirm>
              <Button
                type="link"
                size="small"
                danger
                onClick={() => handleOpenReject(record)}
              >
                驳回
              </Button>
            </>
          )}

          {canAudit && record.status === 20 && (
            <Popconfirm
              title="确定隐藏该评价吗？"
              description="隐藏后仅管理员与作者自己可见"
              onConfirm={() => handleHide(record)}
            >
              <Button type="link" size="small" className="!text-gray-500">
                隐藏
              </Button>
            </Popconfirm>
          )}

          {canAudit && record.status === 40 && (
            <Popconfirm
              title="确定恢复公开该评价吗？"
              onConfirm={() => handleRestore(record)}
            >
              <Button type="link" size="small" className="!text-blue-600">
                恢复
              </Button>
            </Popconfirm>
          )}

          {canReply && (
            <Button
              type="link"
              size="small"
              onClick={() => handleOpenReply(record)}
            >
              {record.reply_content ? '改回复' : '回复'}
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className="admin-list-page" style={adminListVars(token)}>
      <PageHeader
        title="商品评价"
        description="管理买家提交的商品评价与晒单，支持审核流转与商家官方回复"
      />

      <ContentCard>
        {/* 筛选区 */}
        <div className="admin-list-filters mb-4">
          <Space wrap size={[12, 12]}>
            <Input
              placeholder="搜索评价内容 / 商品名称"
              prefix={<SearchOutlined />}
              value={keywordInput}
              onChange={(e) => setKeywordInput(e.target.value)}
              onPressEnter={handleSearch}
              style={{ width: 220 }}
              allowClear
            />

            <Select
              placeholder="评价状态"
              value={statusFilter}
              onChange={(v) => setStatusFilter(v)}
              style={{ width: 120 }}
              allowClear
              options={[
                { label: '全部状态', value: undefined },
                { label: '待审核', value: 10 },
                { label: '已公开', value: 20 },
                { label: '已驳回', value: 30 },
                { label: '已隐藏', value: 40 },
              ]}
            />

            <Select
              placeholder="评分"
              value={ratingFilter}
              onChange={(v) => setRatingFilter(v)}
              style={{ width: 110 }}
              allowClear
              options={[
                { label: '全部评分', value: undefined },
                { label: '5 星', value: 5 },
                { label: '4 星', value: 4 },
                { label: '3 星', value: 3 },
                { label: '2 星', value: 2 },
                { label: '1 星', value: 1 },
              ]}
            />

            <Select
              placeholder="是否有图"
              value={hasImageFilter}
              onChange={(v) => setHasImageFilter(v)}
              style={{ width: 110 }}
              allowClear
              options={[
                { label: '全部', value: undefined },
                { label: '有图', value: 1 },
                { label: '无图', value: 0 },
              ]}
            />

            <RangePicker
              value={dateRange}
              onChange={(dates) => setDateRange(dates as any)}
              style={{ width: 240 }}
            />

            <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleReset}>
              重置
            </Button>
          </Space>
        </div>

        {/* 表格区 */}
        <Table<ReviewListItemResponse>
          rowKey="id"
          columns={columns}
          dataSource={list}
          loading={loading}
          pagination={false}
          scroll={{ x: 1100 }}
          size="middle"
        />

        {/* 分页栏 */}
        <div className="admin-list-pagination flex justify-end mt-4">
          <Pagination
            current={page}
            pageSize={pageSize}
            total={total}
            showTotal={(t) => `共 ${t} 条评价`}
            showSizeChanger
            pageSizeOptions={['10', '20', '50']}
            onChange={(p, ps) => {
              setPage(p);
              setPageSize(ps);
              void loadData(p, ps);
            }}
          />
        </div>
      </ContentCard>

      {/* 评价详情 Drawer */}
      <Drawer
        title="评价详情"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={640}
        destroyOnClose
      >
        {drawerLoading ? (
          <div className="py-12 text-center text-gray-400">正在加载评价详情...</div>
        ) : detailData ? (
          <div className="space-y-6">
            <Descriptions title="基本信息" bordered size="small" column={2}>
              <Descriptions.Item label="评价 ID">{detailData.id}</Descriptions.Item>
              <Descriptions.Item label="状态">
                {renderStatusTag(detailData.status)}
              </Descriptions.Item>
              <Descriptions.Item label="商品名称" span={2}>
                <span className="font-medium">{detailData.product_name}</span>
                {detailData.sku_spec_values && (
                  <span className="text-gray-400 text-xs ml-2">
                    ({detailData.sku_spec_values})
                  </span>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="订单号">{detailData.order_no || '—'}</Descriptions.Item>
              <Descriptions.Item label="订单明细 ID">
                {detailData.order_item_id || '—'}
              </Descriptions.Item>
              <Descriptions.Item label="会员">
                {detailData.member_nickname} ({detailData.member_mobile})
              </Descriptions.Item>
              <Descriptions.Item label="匿名评价">
                {detailData.is_anonymous ? '是' : '否'}
              </Descriptions.Item>
              <Descriptions.Item label="评分">
                <Rate disabled value={detailData.rating} style={{ fontSize: 14, color: '#ff9800' }} />
              </Descriptions.Item>
              <Descriptions.Item label="评价时间">
                {detailData.created_at ? dayjs(detailData.created_at).format('YYYY-MM-DD HH:mm:ss') : '—'}
              </Descriptions.Item>
            </Descriptions>

            <div>
              <div className="text-sm font-semibold mb-2">评价内容</div>
              <div className="p-3 bg-gray-50 rounded text-gray-800 leading-relaxed text-sm">
                {detailData.content || '—'}
              </div>
            </div>

            {detailData.images && detailData.images.length > 0 && (
              <div>
                <div className="text-sm font-semibold mb-2">晒单图片 ({detailData.images.length})</div>
                <Image.PreviewGroup>
                  <Space size={8} wrap>
                    {detailData.images.map((img) => (
                      <Image
                        key={img.id}
                        src={img.url}
                        width={90}
                        height={90}
                        className="rounded border object-cover"
                      />
                    ))}
                  </Space>
                </Image.PreviewGroup>
              </div>
            )}

            {detailData.status === 30 && detailData.audit_reason && (
              <div className="p-3 bg-red-50 border border-red-200 rounded">
                <div className="text-sm font-semibold text-red-700 mb-1">审核驳回原因</div>
                <div className="text-sm text-red-600">{detailData.audit_reason}</div>
              </div>
            )}

            {detailData.reply_content && (
              <div className="p-3 bg-blue-50 border border-blue-200 rounded">
                <div className="flex justify-between items-center mb-1">
                  <span className="text-sm font-semibold text-blue-700">商家回复</span>
                  <span className="text-xs text-gray-400">
                    {detailData.reply_user_name} (
                    {detailData.reply_at ? dayjs(detailData.reply_at).format('YYYY-MM-DD HH:mm') : ''}
                    )
                  </span>
                </div>
                <div className="text-sm text-blue-800">{detailData.reply_content}</div>
              </div>
            )}

            {detailData.logs && detailData.logs.length > 0 && (
              <div>
                <div className="text-sm font-semibold mb-3">流转记录</div>
                <Timeline
                  items={detailData.logs.map((log: ReviewLogResponse) => ({
                    color: log.action === 'approve' ? 'green' : log.action === 'reject' ? 'red' : 'blue',
                    children: (
                      <div className="text-xs">
                        <div className="text-gray-800 font-medium">
                          {log.operator_type_name}: {log.operator_name || '系统'} —{' '}
                          {log.from_status_text} → {log.to_status_text}
                        </div>
                        {log.remark && <div className="text-gray-500 mt-0.5">{log.remark}</div>}
                        <div className="text-gray-400 mt-0.5">
                          {log.created_at ? dayjs(log.created_at).format('YYYY-MM-DD HH:mm:ss') : ''}
                        </div>
                      </div>
                    ),
                  }))}
                />
              </div>
            )}
          </div>
        ) : null}
      </Drawer>

      {/* 驳回 Modal */}
      <Modal
        title="审核驳回评价"
        open={rejectModalOpen}
        onOk={handleConfirmReject}
        onCancel={() => setRejectModalOpen(false)}
        confirmLoading={rejectSubmitting}
        okText="确认驳回"
        okButtonProps={{ danger: true }}
      >
        <div className="mb-3 text-sm text-gray-600">
          请输入驳回该评价的具体原因（必填，将反馈给买家）：
        </div>
        <Input.TextArea
          rows={4}
          maxLength={255}
          showCount
          placeholder="例如：评价内容包含不文明用语或与商品无关的信息..."
          value={rejectReason}
          onChange={(e) => setRejectReason(e.target.value)}
        />
      </Modal>

      {/* 商家回复 Modal */}
      <Modal
        title="回复评价"
        open={replyModalOpen}
        onOk={handleConfirmReply}
        onCancel={() => setReplyModalOpen(false)}
        confirmLoading={replySubmitting}
        okText="保存回复"
      >
        <div className="mb-3 text-sm text-gray-600">
          以官方身份回复该买家评价（公开展示于评价正文下方，最多500字）：
        </div>
        <Input.TextArea
          rows={5}
          maxLength={500}
          showCount
          placeholder="感谢您的认可，我们将持续为您提供更优质的商品与服务..."
          value={replyContent}
          onChange={(e) => setReplyContent(e.target.value)}
        />
      </Modal>
    </div>
  );
}
