package aftersale

import (
	"context"
	"io"

	"mall/admin_core/internal/model"
	orderaftersale "mall/admin_core/internal/order/aftersale"
)

type Service struct {
	core *orderaftersale.Service
}

func NewService(core *orderaftersale.Service) *Service {
	return &Service{core: core}
}

func (s *Service) Apply(ctx context.Context, memberID int64, req MiniappApplyAfterSaleRequest) (*model.AfterSale, error) {
	return s.core.Apply(ctx, memberID, orderaftersale.MiniappApplyRequest{
		OrderID:      req.OrderID,
		Type:         req.Type,
		Reason:       req.Reason,
		Description:  req.Description,
		ProofFileIDs: req.ProofFileIDs,
		RequestID:    req.RequestID,
	})
}

func (s *Service) Cancel(ctx context.Context, memberID int64, afterSaleID int64) error {
	return s.core.Cancel(ctx, memberID, afterSaleID)
}

func (s *Service) SubmitReturnLogistics(ctx context.Context, memberID int64, afterSaleID int64, company, trackingNo string) error {
	return s.core.SubmitReturnLogistics(ctx, memberID, afterSaleID, company, trackingNo)
}

func (s *Service) List(ctx context.Context, memberID int64, req MiniappAfterSaleListRequest) ([]MiniappAfterSaleListItemResponse, int64, error) {
	rows, total, err := s.core.MiniappList(ctx, memberID, orderaftersale.MiniappListRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   req.Status,
	})
	if err != nil {
		return nil, 0, err
	}
	result := make([]MiniappAfterSaleListItemResponse, 0, len(rows))
	for _, r := range rows {
		items := make([]MiniappAfterSaleItemResponse, 0, len(r.Items))
		for _, it := range r.Items {
			items = append(items, MiniappAfterSaleItemResponse{
				ID:           it.ID,
				ProductID:    it.ProductID,
				SKUID:        it.SKUID,
				Quantity:     it.Quantity,
				RefundAmount: it.RefundAmount,
				ProductName:  it.ProductName,
				SKUText:      it.SKUText,
				SKUCode:      it.SKUCode,
				SKUImageURL:  it.SKUImageURL,
			})
		}
		result = append(result, MiniappAfterSaleListItemResponse{
			ID:           r.ID,
			AfterSaleNo:  r.AfterSaleNo,
			OrderID:      r.OrderID,
			Type:         r.Type,
			TypeText:     r.TypeText,
			Status:       r.Status,
			StatusText:   r.StatusText,
			RefundAmount: r.RefundAmount,
			Reason:       r.Reason,
			CreatedAt:    r.CreatedAt,
			Items:        items,
		})
	}
	return result, total, nil
}

func (s *Service) Detail(ctx context.Context, memberID int64, afterSaleID int64) (*MiniappAfterSaleDetailResponse, error) {
	d, err := s.core.MiniappDetail(ctx, memberID, afterSaleID)
	if err != nil {
		return nil, err
	}
	items := make([]MiniappAfterSaleItemResponse, 0, len(d.Items))
	for _, it := range d.Items {
		items = append(items, MiniappAfterSaleItemResponse{
			ID:           it.ID,
			ProductID:    it.ProductID,
			SKUID:        it.SKUID,
			Quantity:     it.Quantity,
			RefundAmount: it.RefundAmount,
			ProductName:  it.ProductName,
			SKUText:      it.SKUText,
			SKUCode:      it.SKUCode,
			SKUImageURL:  it.SKUImageURL,
		})
	}
	logs := make([]MiniappAfterSaleLogRow, 0, len(d.Logs))
	for _, l := range d.Logs {
		logs = append(logs, MiniappAfterSaleLogRow{
			Action:         l.Action,
			OperatorType:   l.OperatorType,
			FromStatusText: l.FromStatusText,
			ToStatusText:   l.ToStatusText,
			Remark:         l.Remark,
			CreatedAt:      l.CreatedAt,
		})
	}
	var refundInfo *MiniappAfterSaleRefundInfo
	if d.Refund != nil {
		refundInfo = &MiniappAfterSaleRefundInfo{
			Status:       d.Refund.Status,
			StatusText:   d.Refund.StatusText,
			RefundAmount: d.Refund.RefundAmount,
			RefundedAt:   d.Refund.RefundedAt,
		}
	}
	proofFiles := make([]MiniappAfterSaleProofFile, 0, len(d.ProofFiles))
	for _, pf := range d.ProofFiles {
		proofFiles = append(proofFiles, MiniappAfterSaleProofFile{
			ID:           pf.ID,
			FileType:     pf.FileType,
			OriginalName: pf.OriginalName,
			MIMEType:     pf.MIMEType,
			Size:         pf.Size,
			URL:          pf.URL,
			Width:        pf.Width,
			Height:       pf.Height,
			DurationMS:   pf.DurationMS,
			CoverURL:     pf.CoverURL,
		})
	}
	return &MiniappAfterSaleDetailResponse{
		ID:                 d.ID,
		AfterSaleNo:        d.AfterSaleNo,
		OrderID:            d.OrderID,
		Type:               d.Type,
		TypeText:           d.TypeText,
		Status:             d.Status,
		StatusText:         d.StatusText,
		Reason:             d.Reason,
		Description:        d.Description,
		ProofURLs:          d.ProofURLs,
		ProofFiles:         proofFiles,
		RefundAmount:       d.RefundAmount,
		ReturnReceiver:     d.ReturnReceiver,
		ReturnPhone:        d.ReturnPhone,
		ReturnAddress:      d.ReturnAddress,
		BuyerReturnCompany: d.BuyerReturnCompany,
		BuyerReturnNo:      d.BuyerReturnNo,
		RejectReason:       d.RejectReason,
		AppliedAt:          d.AppliedAt,
		AuditedAt:          d.AuditedAt,
		BuyerReturnedAt:    d.BuyerReturnedAt,
		MerchantReceivedAt: d.MerchantReceivedAt,
		CompletedAt:        d.CompletedAt,
		CancelledAt:        d.CancelledAt,
		CreatedAt:          d.CreatedAt,
		Items:              items,
		Refund:             refundInfo,
		Logs:               logs,
	}, nil
}

func (s *Service) UploadProof(ctx context.Context, memberID int64, filename string, reader io.Reader, declaredSize int64) (MiniappAfterSaleProofUploadResponse, error) {
	resp, err := s.core.UploadProof(ctx, memberID, filename, reader, declaredSize)
	if err != nil {
		return MiniappAfterSaleProofUploadResponse{}, err
	}
	return MiniappAfterSaleProofUploadResponse{
		ID:           resp.ID,
		FileType:     resp.FileType,
		OriginalName: resp.OriginalName,
		MIMEType:     resp.MIMEType,
		Size:         resp.Size,
		URL:          resp.URL,
		Width:        resp.Width,
		Height:       resp.Height,
		DurationMS:   resp.DurationMS,
	}, nil
}
