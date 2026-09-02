package biz

import (
	"context"
	"fmt"
	"strings"
)

type Application struct {
	BaseFields
	Application string `json:"application"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

type ApplicationRepository interface {
	Create(context.Context, *Application) (*Application, error)
	Get(context.Context, string) (*Application, error)
	List(context.Context) ([]*Application, error)
	Update(context.Context, *Application) (*Application, error)
	SoftDelete(context.Context, string) error
}

type ApplicationService struct {
	repository ApplicationRepository
	source     string
}

func NewApplicationService(repository ApplicationRepository, source string) *ApplicationService {
	return &ApplicationService{repository: repository, source: strings.ToLower(strings.TrimSpace(source))}
}

func (s *ApplicationService) localOnly() error {
	if s.source == "" || s.source == "local" {
		return nil
	}
	return fmt.Errorf("%w: application source %q is not implemented", ErrNotImplemented, s.source)
}

func (s *ApplicationService) Create(ctx context.Context, application, name, description string) (*Application, error) {
	if err := s.localOnly(); err != nil {
		return nil, err
	}
	application, err := validateApplication(application)
	if err != nil {
		return nil, err
	}
	_, name, err = validateCodeAndName(application, name)
	if err != nil {
		return nil, err
	}
	return s.repository.Create(ctx, &Application{BaseFields: NewBaseFields(AuditActorFromContext(ctx)), Application: application, Name: name, Description: strings.TrimSpace(description), Enabled: true})
}

func (s *ApplicationService) List(ctx context.Context) ([]*Application, error) {
	if err := s.localOnly(); err != nil {
		return nil, err
	}
	return s.repository.List(ctx)
}
func (s *ApplicationService) Get(ctx context.Context, application string) (*Application, error) {
	if err := s.localOnly(); err != nil {
		return nil, err
	}
	application, err := validateApplication(application)
	if err != nil {
		return nil, err
	}
	return s.repository.Get(ctx, application)
}
func (s *ApplicationService) Update(ctx context.Context, application, name, description string, enabled bool) (*Application, error) {
	if err := s.localOnly(); err != nil {
		return nil, err
	}
	application, err := validateApplication(application)
	if err != nil {
		return nil, err
	}
	_, name, err = validateCodeAndName(application, name)
	if err != nil {
		return nil, err
	}
	return s.repository.Update(ctx, &Application{BaseFields: NewBaseFields(AuditActorFromContext(ctx)), Application: application, Name: name, Description: strings.TrimSpace(description), Enabled: enabled})
}
func (s *ApplicationService) Delete(ctx context.Context, application string) error {
	if err := s.localOnly(); err != nil {
		return err
	}
	application, err := validateApplication(application)
	if err != nil {
		return err
	}
	return s.repository.SoftDelete(ctx, application)
}
