package domain

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/galayx-future/costpilot/internal/services"
	"github.com/galayx-future/costpilot/internal/types"
	"github.com/pkg/errors"
)

type CostAnalysisDomain struct {
	nowT              time.Time
	monthsBillingList []*sync.Map
	daysBillingList   []*sync.Map
}

func NewCostAnalysisDomain() *CostAnalysisDomain {
	return &CostAnalysisDomain{
		nowT:              time.Now(),
		monthsBillingList: []*sync.Map{},
		daysBillingList:   []*sync.Map{},
	}
}

// GetBillingList
func (s *CostAnalysisDomain) GetBillingList(ctx context.Context) error {
	accounts := services.NewAccountService().GetAccounts()
	if len(accounts) == 0 {
		log.Println("E! cloud account is not configured, please check conf/config.yml")
		return errors.New("cloud account is not configured")
	}
	for _, a := range accounts {
		monthsBilling, daysBilling, err := s.GetBilling(ctx, a)
		if err != nil {
			log.Printf("E! get cloud-acount[%v] billing error", a.Name)
			return err
		}
		s.monthsBillingList = append(s.monthsBillingList, monthsBilling)
		s.daysBillingList = append(s.daysBillingList, daysBilling)
	}
	return nil
}

// GetBilling
func (s *CostAnalysisDomain) GetBilling(ctx context.Context, a types.CloudAccount) (monthsBilling, daysBilling *sync.Map, err error) {
	viewSvc := services.NewViewService(a, s.nowT)
	err = viewSvc.RunPipeline(ctx)
	if err != nil {
		return nil, nil, err
	}
	monthsBilling, daysBilling = viewSvc.GetBillingMap()
	log.Printf("I! get cloud-account[%s] billing success", a.Name)
	return
}

// ExportStatisticData 导出到静态文件
func (s *CostAnalysisDomain) ExportStatisticData(ctx context.Context) error {
	formatSvc := services.NewTemplateService(nil, nil, s.nowT)
	err := formatSvc.CombineBilling(ctx, s.monthsBillingList, s.daysBillingList)
	if err != nil {
		return err
	}
	err = formatSvc.ExportCostAnalysis(ctx)
	if err != nil {
		log.Printf("E! export cost-analysis data failed: %v\n", err)
		return err
	}
	return nil
}

func (s *CostAnalysisDomain) GetCostAnalysisPipeline() []func(context.Context) error {
	return []func(context.Context) error{
		s.GetBillingList,
		s.ExportStatisticData,
	}
}

func (s *CostAnalysisDomain) RunPipeline(ctx context.Context) error {
	var err error
	for _, f := range s.GetCostAnalysisPipeline() {
		err = f(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
