package executor

import (
	"fmt"
	"uptime/internal/domain"
)

type Factory struct {
	executors map[domain.MonitorType]domain.CheckExecutor
}

func NewFactory() *Factory {
	return &Factory{
		executors: map[domain.MonitorType]domain.CheckExecutor{
			domain.MonitorTypeHTTP: NewHTTPExecutor(),
			domain.MonitorTypeTCP:  NewTCPExecutor(),
			domain.MonitorTypeICMP: NewICMPExecutor(),
		},
	}
}

func (f *Factory) Get(t domain.MonitorType) (domain.CheckExecutor, error) {
	ex, ok := f.executors[t]
	if !ok {
		return nil, fmt.Errorf("unsupported monitor type: %s", t)
	}
	return ex, nil
}
