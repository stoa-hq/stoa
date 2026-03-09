package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// Sentinel errors for the payment domain.
var (
	ErrMethodNotFound  = errors.New("payment method not found")
	ErrInvalidInput    = errors.New("invalid input")
)

// Hook name constants for the payment domain.
const (
	HookBeforePaymentMethodCreate = "payment_method.before_create"
	HookAfterPaymentMethodCreate  = "payment_method.after_create"
	HookBeforePaymentMethodUpdate = "payment_method.before_update"
	HookAfterPaymentMethodUpdate  = "payment_method.after_update"
	HookBeforePaymentMethodDelete = "payment_method.before_delete"
	HookAfterPaymentMethodDelete  = "payment_method.after_delete"
	HookAfterTransactionCreate    = "payment_transaction.after_create"
)

// PaymentMethodService defines the business-logic interface for payment methods.
type PaymentMethodService interface {
	List(ctx context.Context, filter PaymentMethodFilter) ([]PaymentMethod, int, error)
	Create(ctx context.Context, m *PaymentMethod) error
	GetByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error)
	Update(ctx context.Context, m *PaymentMethod) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PaymentTransactionService defines the business-logic interface for payment transactions.
type PaymentTransactionService interface {
	CreateTransaction(ctx context.Context, t *PaymentTransaction) error
	GetTransactionsByOrderID(ctx context.Context, orderID uuid.UUID) ([]PaymentTransaction, error)
}

type methodService struct {
	repo   PaymentMethodRepository
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewMethodService creates a new PaymentMethodService.
func NewMethodService(repo PaymentMethodRepository, hooks *sdk.HookRegistry, logger zerolog.Logger) PaymentMethodService {
	return &methodService{repo: repo, hooks: hooks, logger: logger}
}

func (s *methodService) List(ctx context.Context, filter PaymentMethodFilter) ([]PaymentMethod, int, error) {
	methods, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("payment: List")
		return nil, 0, err
	}
	return methods, total, nil
}

func (s *methodService) Create(ctx context.Context, m *PaymentMethod) error {
	if m.Provider == "" {
		return fmt.Errorf("%w: provider is required", ErrInvalidInput)
	}
	m.ID = uuid.New()
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now

	for i := range m.Translations {
		m.Translations[i].PaymentMethodID = m.ID
	}

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforePaymentMethodCreate,
			Entity: m,
		}); err != nil {
			return fmt.Errorf("payment: before_create hook: %w", err)
		}
	}

	if err := s.repo.Create(ctx, m); err != nil {
		s.logger.Error().Err(err).Msg("payment: Create")
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterPaymentMethodCreate,
			Entity: m,
		})
	}
	return nil
}

func (s *methodService) GetByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrMethodNotFound) {
			s.logger.Error().Err(err).Msg("payment: GetByID")
		}
		return nil, err
	}
	return m, nil
}

func (s *methodService) Update(ctx context.Context, m *PaymentMethod) error {
	if m.Provider == "" {
		return fmt.Errorf("%w: provider is required", ErrInvalidInput)
	}
	m.UpdatedAt = time.Now().UTC()

	for i := range m.Translations {
		m.Translations[i].PaymentMethodID = m.ID
	}

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforePaymentMethodUpdate,
			Entity: m,
		}); err != nil {
			return fmt.Errorf("payment: before_update hook: %w", err)
		}
	}

	if err := s.repo.Update(ctx, m); err != nil {
		if !errors.Is(err, ErrMethodNotFound) {
			s.logger.Error().Err(err).Msg("payment: Update")
		}
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterPaymentMethodUpdate,
			Entity: m,
		})
	}
	return nil
}

func (s *methodService) Delete(ctx context.Context, id uuid.UUID) error {
	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforePaymentMethodDelete,
			Entity: &PaymentMethod{ID: id},
		}); err != nil {
			return fmt.Errorf("payment: before_delete hook: %w", err)
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if !errors.Is(err, ErrMethodNotFound) {
			s.logger.Error().Err(err).Msg("payment: Delete")
		}
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterPaymentMethodDelete,
			Entity: &PaymentMethod{ID: id},
		})
	}
	return nil
}

// --- transaction service ---

type transactionService struct {
	repo   PaymentTransactionRepository
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewTransactionService creates a new PaymentTransactionService.
func NewTransactionService(repo PaymentTransactionRepository, hooks *sdk.HookRegistry, logger zerolog.Logger) PaymentTransactionService {
	return &transactionService{repo: repo, hooks: hooks, logger: logger}
}

func (s *transactionService) CreateTransaction(ctx context.Context, t *PaymentTransaction) error {
	if t.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order_id is required", ErrInvalidInput)
	}
	if t.PaymentMethodID == uuid.Nil {
		return fmt.Errorf("%w: payment_method_id is required", ErrInvalidInput)
	}
	t.ID = uuid.New()
	t.CreatedAt = time.Now().UTC()

	if err := s.repo.Create(ctx, t); err != nil {
		s.logger.Error().Err(err).Msg("payment: CreateTransaction")
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterTransactionCreate,
			Entity: t,
		})
	}
	return nil
}

func (s *transactionService) GetTransactionsByOrderID(ctx context.Context, orderID uuid.UUID) ([]PaymentTransaction, error) {
	transactions, err := s.repo.FindByOrderID(ctx, orderID)
	if err != nil {
		s.logger.Error().Err(err).Msg("payment: GetTransactionsByOrderID")
		return nil, err
	}
	return transactions, nil
}
