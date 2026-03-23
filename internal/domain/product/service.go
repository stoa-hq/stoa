package product

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

var identifierRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// TaxRateFn looks up the integer basis-point tax rate for a given tax rule ID.
// Returns an error if the rule is not found or unavailable.
type TaxRateFn func(ctx context.Context, id uuid.UUID) (int, error)

// Service implements business logic for the product domain.
type Service struct {
	repo       ProductRepository
	hooks      *sdk.HookRegistry
	logger     zerolog.Logger
	mediaURLFn func(string) string // optional; computes public URL from storage path
	taxRateFn  TaxRateFn           // optional; looks up tax rate by tax rule ID
}

// NewService constructs a product Service.
// mediaURLFn computes public media URLs from storage paths (may be nil).
// taxRateFn looks up tax rates by tax rule ID for automatic price calculation (may be nil).
func NewService(repo ProductRepository, hooks *sdk.HookRegistry, logger zerolog.Logger, mediaURLFn func(string) string, taxRateFn TaxRateFn) *Service {
	return &Service{
		repo:       repo,
		hooks:      hooks,
		logger:     logger,
		mediaURLFn: mediaURLFn,
		taxRateFn:  taxRateFn,
	}
}

func calcNetFromGross(gross, rate int) int {
	return int(math.Round(float64(gross) * 10000 / float64(10000+rate)))
}

func calcGrossFromNet(net, rate int) int {
	return int(math.Round(float64(net) * float64(10000+rate) / 10000))
}

// setMediaURLs populates the URL field of each ProductMedia using the configured URL function.
func (s *Service) setMediaURLs(p *Product) {
	if s.mediaURLFn == nil {
		return
	}
	for i := range p.Media {
		p.Media[i].URL = s.mediaURLFn(p.Media[i].StoragePath)
	}
}

// --------------------------------------------------------------------------
// GetByID
// --------------------------------------------------------------------------

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("service GetByID: %w", err)
	}
	s.setMediaURLs(p)
	return p, nil
}

// --------------------------------------------------------------------------
// List
// --------------------------------------------------------------------------

func (s *Service) List(ctx context.Context, filter ProductFilter) ([]Product, int, error) {
	products, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("service List: %w", err)
	}
	for i := range products {
		s.setMediaURLs(&products[i])
	}
	return products, total, nil
}

// --------------------------------------------------------------------------
// GetBySlug
// --------------------------------------------------------------------------

func (s *Service) GetBySlug(ctx context.Context, slug, locale string) (*Product, error) {
	p, err := s.repo.FindBySlug(ctx, slug, locale)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("service GetBySlug: %w", err)
	}
	s.setMediaURLs(p)
	return p, nil
}

// --------------------------------------------------------------------------
// Create
// --------------------------------------------------------------------------

func (s *Service) Create(ctx context.Context, p *Product) error {
	// Auto-calculate missing price from tax rule.
	if p.TaxRuleID != nil && s.taxRateFn != nil {
		if rate, err := s.taxRateFn(ctx, *p.TaxRuleID); err == nil && rate > 0 {
			if p.PriceGross > 0 && p.PriceNet == 0 {
				p.PriceNet = calcNetFromGross(p.PriceGross, rate)
			} else if p.PriceNet > 0 && p.PriceGross == 0 {
				p.PriceGross = calcGrossFromNet(p.PriceNet, rate)
			}
		}
	}

	// Fire before-hook – handlers may cancel by returning an error.
	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookBeforeProductCreate,
		Entity: p,
	}); err != nil {
		return fmt.Errorf("hook %s: %w", sdk.HookBeforeProductCreate, err)
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return fmt.Errorf("service Create: %w", err)
	}

	// Fire after-hook – errors are logged but do not fail the operation.
	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookAfterProductCreate,
		Entity: p,
	}); err != nil {
		s.logger.Warn().Err(err).Str("hook", sdk.HookAfterProductCreate).Msg("after-hook error")
	}

	return nil
}

// --------------------------------------------------------------------------
// Update
// --------------------------------------------------------------------------

func (s *Service) Update(ctx context.Context, p *Product) error {
	// Auto-calculate missing price from tax rule.
	if p.TaxRuleID != nil && s.taxRateFn != nil {
		if rate, err := s.taxRateFn(ctx, *p.TaxRuleID); err == nil && rate > 0 {
			if p.PriceGross > 0 && p.PriceNet == 0 {
				p.PriceNet = calcNetFromGross(p.PriceGross, rate)
			} else if p.PriceNet > 0 && p.PriceGross == 0 {
				p.PriceGross = calcGrossFromNet(p.PriceNet, rate)
			}
		}
	}

	// Verify the product exists first so hooks get the full current state.
	existing, err := s.repo.FindByID(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("service Update FindByID: %w", err)
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookBeforeProductUpdate,
		Entity: existing,
		Changes: map[string]interface{}{
			"incoming": p,
		},
	}); err != nil {
		return fmt.Errorf("hook %s: %w", sdk.HookBeforeProductUpdate, err)
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("service Update: %w", err)
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookAfterProductUpdate,
		Entity: p,
	}); err != nil {
		s.logger.Warn().Err(err).Str("hook", sdk.HookAfterProductUpdate).Msg("after-hook error")
	}

	return nil
}

// --------------------------------------------------------------------------
// Delete
// --------------------------------------------------------------------------

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("service Delete FindByID: %w", err)
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookBeforeProductDelete,
		Entity: p,
	}); err != nil {
		return fmt.Errorf("hook %s: %w", sdk.HookBeforeProductDelete, err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("service Delete: %w", err)
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookAfterProductDelete,
		Entity: p,
	}); err != nil {
		s.logger.Warn().Err(err).Str("hook", sdk.HookAfterProductDelete).Msg("after-hook error")
	}

	return nil
}

// --------------------------------------------------------------------------
// GenerateVariants
// --------------------------------------------------------------------------

// GenerateVariants creates every combination of the provided option groups.
//
// optionIDs is a slice-of-slices where each inner slice represents one property
// axis (e.g. [[sizeS, sizeM, sizeL], [colorRed, colorBlue]]).  The method
// computes the cartesian product and persists one ProductVariant per combination
// that does not already exist.
func (s *Service) GenerateVariants(ctx context.Context, productID uuid.UUID, optionIDs [][]uuid.UUID) ([]ProductVariant, error) {
	if _, err := s.repo.FindByID(ctx, productID); err != nil {
		return nil, fmt.Errorf("service GenerateVariants FindByID: %w", err)
	}

	combinations := cartesianProduct(optionIDs)
	if len(combinations) == 0 {
		return nil, fmt.Errorf("service GenerateVariants: no option combinations produced")
	}

	var created []ProductVariant
	for _, combo := range combinations {
		v := ProductVariant{
			ID:        uuid.New(),
			ProductID: productID,
			Active:    true,
			Stock:     0,
		}

		// Attach PropertyOption stubs – full data would require additional queries
		// but variant generation only needs the IDs persisted in the pivot table.
		for _, optID := range combo {
			v.Options = append(v.Options, PropertyOption{ID: optID})
		}

		if err := s.persistVariant(ctx, &v); err != nil {
			return nil, fmt.Errorf("service GenerateVariants persistVariant: %w", err)
		}
		created = append(created, v)
	}

	return created, nil
}

// persistVariant inserts a variant and its option pivots via the repository.
func (s *Service) persistVariant(ctx context.Context, v *ProductVariant) error {
	return s.repo.CreateVariant(ctx, v)
}

// --------------------------------------------------------------------------
// CreateVariant
// --------------------------------------------------------------------------

// CreateVariant creates a single product variant with the given options.
func (s *Service) CreateVariant(ctx context.Context, productID uuid.UUID, req CreateVariantRequest) (*ProductVariant, error) {
	if _, err := s.repo.FindByID(ctx, productID); err != nil {
		return nil, fmt.Errorf("service CreateVariant FindByID: %w", err)
	}

	v := &ProductVariant{
		ID:         uuid.New(),
		ProductID:  productID,
		SKU:        req.SKU,
		PriceGross: req.PriceGross,
		PriceNet:   req.PriceNet,
		Stock:      req.Stock,
		Active:     req.Active,
	}
	for _, optID := range req.OptionIDs {
		v.Options = append(v.Options, PropertyOption{ID: optID})
	}

	if err := s.repo.CreateVariant(ctx, v); err != nil {
		return nil, fmt.Errorf("service CreateVariant: %w", err)
	}

	// Re-load to get full option data.
	full, err := s.repo.FindVariantByID(ctx, v.ID)
	if err != nil {
		return nil, fmt.Errorf("service CreateVariant reload: %w", err)
	}
	return full, nil
}

// --------------------------------------------------------------------------
// UpdateVariant
// --------------------------------------------------------------------------

// UpdateVariant updates a product variant by its ID.
func (s *Service) UpdateVariant(ctx context.Context, variantID uuid.UUID, req UpdateVariantRequest) (*ProductVariant, error) {
	v, err := s.repo.FindVariantByID(ctx, variantID)
	if err != nil {
		return nil, fmt.Errorf("service UpdateVariant FindByID: %w", err)
	}

	v.SKU = req.SKU
	v.PriceGross = req.PriceGross
	v.PriceNet = req.PriceNet
	v.Stock = req.Stock
	v.Active = req.Active
	v.Options = nil
	for _, optID := range req.OptionIDs {
		v.Options = append(v.Options, PropertyOption{ID: optID})
	}

	if err := s.repo.UpdateVariant(ctx, v); err != nil {
		return nil, fmt.Errorf("service UpdateVariant: %w", err)
	}

	full, err := s.repo.FindVariantByID(ctx, v.ID)
	if err != nil {
		return nil, fmt.Errorf("service UpdateVariant reload: %w", err)
	}
	return full, nil
}

// --------------------------------------------------------------------------
// DeleteVariant
// --------------------------------------------------------------------------

// DeleteVariant removes a product variant by its ID.
func (s *Service) DeleteVariant(ctx context.Context, variantID uuid.UUID) error {
	if err := s.repo.DeleteVariant(ctx, variantID); err != nil {
		return fmt.Errorf("service DeleteVariant: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Property Groups
// --------------------------------------------------------------------------

// ListPropertyGroups returns all property groups with their options.
func (s *Service) ListPropertyGroups(ctx context.Context) ([]PropertyGroup, error) {
	groups, err := s.repo.FindAllPropertyGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("service ListPropertyGroups: %w", err)
	}
	return groups, nil
}

// GetPropertyGroupByID returns a single property group.
func (s *Service) GetPropertyGroupByID(ctx context.Context, id uuid.UUID) (*PropertyGroup, error) {
	g, err := s.repo.FindPropertyGroupByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service GetPropertyGroupByID: %w", err)
	}
	return g, nil
}

// ErrInvalidIdentifier is returned when an identifier does not match the expected format.
var ErrInvalidIdentifier = errors.New("product: invalid identifier format")

// validateIdentifier checks that the identifier matches the allowed pattern.
func validateIdentifier(id string) error {
	if !identifierRe.MatchString(id) {
		return ErrInvalidIdentifier
	}
	return nil
}

// CreatePropertyGroup creates a new property group.
func (s *Service) CreatePropertyGroup(ctx context.Context, g *PropertyGroup) error {
	if err := validateIdentifier(g.Identifier); err != nil {
		return err
	}
	if err := s.repo.CreatePropertyGroup(ctx, g); err != nil {
		if errors.Is(err, ErrDuplicateIdentifier) {
			return ErrDuplicateIdentifier
		}
		return fmt.Errorf("service CreatePropertyGroup: %w", err)
	}
	return nil
}

// UpdatePropertyGroup updates an existing property group.
func (s *Service) UpdatePropertyGroup(ctx context.Context, g *PropertyGroup) error {
	if err := validateIdentifier(g.Identifier); err != nil {
		return err
	}
	if err := s.repo.UpdatePropertyGroup(ctx, g); err != nil {
		if errors.Is(err, ErrDuplicateIdentifier) {
			return ErrDuplicateIdentifier
		}
		return fmt.Errorf("service UpdatePropertyGroup: %w", err)
	}
	return nil
}

// GetPropertyGroupByIdentifier returns a single property group by its identifier.
func (s *Service) GetPropertyGroupByIdentifier(ctx context.Context, identifier string) (*PropertyGroup, error) {
	g, err := s.repo.FindPropertyGroupByIdentifier(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("service GetPropertyGroupByIdentifier: %w", err)
	}
	return g, nil
}

// DeletePropertyGroup removes a property group (and cascades to options).
func (s *Service) DeletePropertyGroup(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeletePropertyGroup(ctx, id); err != nil {
		return fmt.Errorf("service DeletePropertyGroup: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Property Options
// --------------------------------------------------------------------------

// CreatePropertyOption adds an option to a group.
func (s *Service) CreatePropertyOption(ctx context.Context, o *PropertyOption) error {
	if err := s.repo.CreatePropertyOption(ctx, o); err != nil {
		return fmt.Errorf("service CreatePropertyOption: %w", err)
	}
	return nil
}

// UpdatePropertyOption updates an existing property option.
func (s *Service) UpdatePropertyOption(ctx context.Context, o *PropertyOption) error {
	if err := s.repo.UpdatePropertyOption(ctx, o); err != nil {
		return fmt.Errorf("service UpdatePropertyOption: %w", err)
	}
	return nil
}

// DeletePropertyOption removes a property option.
func (s *Service) DeletePropertyOption(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeletePropertyOption(ctx, id); err != nil {
		return fmt.Errorf("service DeletePropertyOption: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// cartesianProduct computes the cartesian product of a slice of UUID slices.
func cartesianProduct(sets [][]uuid.UUID) [][]uuid.UUID {
	result := [][]uuid.UUID{{}}
	for _, set := range sets {
		var next [][]uuid.UUID
		for _, existing := range result {
			for _, val := range set {
				combo := make([]uuid.UUID, len(existing)+1)
				copy(combo, existing)
				combo[len(existing)] = val
				next = append(next, combo)
			}
		}
		result = next
	}
	// Remove the empty seed if no sets were provided.
	if len(result) == 1 && len(result[0]) == 0 {
		return nil
	}
	return result
}
