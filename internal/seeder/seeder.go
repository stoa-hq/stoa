// Package seeder inserts demo data into a freshly migrated database.
// All inserts run inside a single transaction; on error everything rolls back.
package seeder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/internal/auth"
)

// ErrAlreadySeeded is returned when the database already contains data and
// --force was not passed.
var ErrAlreadySeeded = errors.New("database already contains data; use --force to override")

// Seeder inserts demo data.
type Seeder struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// New creates a new Seeder.
func New(pool *pgxpool.Pool, logger zerolog.Logger) *Seeder {
	return &Seeder{pool: pool, logger: logger}
}

// SeedDemo seeds demo data within a single transaction. If force is false and
// the database already contains rows in tax_rules, ErrAlreadySeeded is returned.
func (s *Seeder) SeedDemo(ctx context.Context, force bool) error {
	if !force {
		var count int
		if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM tax_rules").Scan(&count); err != nil {
			return fmt.Errorf("checking existing data: %w", err)
		}
		if count > 0 {
			return ErrAlreadySeeded
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := s.seed(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	s.logger.Info().Msg("demo data seeded successfully")
	return nil
}

// exec is a convenience wrapper around tx.Exec that discards the CommandTag.
func exec(ctx context.Context, tx pgx.Tx, sql string, args ...any) error {
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return nil
}

func (s *Seeder) step(name string) {
	s.logger.Info().Str("step", name).Msg("seeding")
}

// seed performs all inserts in dependency order.
func (s *Seeder) seed(ctx context.Context, tx pgx.Tx) error {
	now := time.Now()

	// ── Tax Rules ────────────────────────────────────────────────────────────
	s.step("tax_rules")
	taxStandardID := uuid.New() // 19 % DE
	taxReducedID := uuid.New()  // 7 % DE
	if err := exec(ctx, tx, `
		INSERT INTO tax_rules (id, name, rate, country_code, type, created_at, updated_at) VALUES
		($1, 'Normaler Steuersatz',   1900, 'DE', 'standard', $3, $3),
		($2, 'Ermäßigter Steuersatz',  700, 'DE', 'reduced',  $3, $3)`,
		taxStandardID, taxReducedID, now,
	); err != nil {
		return fmt.Errorf("tax_rules: %w", err)
	}

	// ── Tags ─────────────────────────────────────────────────────────────────
	s.step("tags")
	tagNeuID := uuid.New()
	tagSaleID := uuid.New()
	tagBestsellerID := uuid.New()
	tagBioID := uuid.New()
	if err := exec(ctx, tx, `
		INSERT INTO tags (id, name, slug) VALUES
		($1, 'Neu',        'neu'),
		($2, 'Sale',       'sale'),
		($3, 'Bestseller', 'bestseller'),
		($4, 'Bio',        'bio')`,
		tagNeuID, tagSaleID, tagBestsellerID, tagBioID,
	); err != nil {
		return fmt.Errorf("tags: %w", err)
	}

	// ── Categories ───────────────────────────────────────────────────────────
	s.step("categories")
	catElektronikID := uuid.New()
	catSmartphonesID := uuid.New()
	catLaptopsID := uuid.New()
	catModeID := uuid.New()
	// parent rows first (catElektronikID, catModeID), then children
	if err := exec(ctx, tx, `
		INSERT INTO categories (id, parent_id, position, active, custom_fields, created_at, updated_at) VALUES
		($1, NULL, 0, true, '{}', $5, $5),
		($4, NULL, 1, true, '{}', $5, $5),
		($2, $1,   0, true, '{}', $5, $5),
		($3, $1,   1, true, '{}', $5, $5)`,
		catElektronikID, catSmartphonesID, catLaptopsID, catModeID, now,
	); err != nil {
		return fmt.Errorf("categories: %w", err)
	}
	if err := exec(ctx, tx, `
		INSERT INTO category_translations (category_id, locale, name, description, slug) VALUES
		($1, 'de-DE', 'Elektronik',  'Elektronische Geräte und Zubehör',     'elektronik'),
		($1, 'en-US', 'Electronics', 'Electronic devices and accessories',   'electronics'),
		($2, 'de-DE', 'Smartphones', 'Aktuelle Smartphones und Handys',       'smartphones'),
		($2, 'en-US', 'Smartphones', 'Latest smartphones and mobile phones',  'smartphones-en'),
		($3, 'de-DE', 'Laptops',     'Notebooks und Ultrabooks',              'laptops'),
		($3, 'en-US', 'Laptops',     'Notebooks and ultrabooks',              'laptops-en'),
		($4, 'de-DE', 'Mode',        'Kleidung und Accessoires',              'mode'),
		($4, 'en-US', 'Fashion',     'Clothing and accessories',              'fashion')`,
		catElektronikID, catSmartphonesID, catLaptopsID, catModeID,
	); err != nil {
		return fmt.Errorf("category_translations: %w", err)
	}

	// ── Property Groups ──────────────────────────────────────────────────────
	s.step("property_groups")
	pgFarbeID := uuid.New()
	pgGroesseID := uuid.New()
	if err := exec(ctx, tx, `
		INSERT INTO property_groups (id, position, created_at, updated_at) VALUES
		($1, 0, $3, $3),
		($2, 1, $3, $3)`,
		pgFarbeID, pgGroesseID, now,
	); err != nil {
		return fmt.Errorf("property_groups: %w", err)
	}
	if err := exec(ctx, tx, `
		INSERT INTO property_group_translations (property_group_id, locale, name) VALUES
		($1, 'de-DE', 'Farbe'),
		($1, 'en-US', 'Color'),
		($2, 'de-DE', 'Größe'),
		($2, 'en-US', 'Size')`,
		pgFarbeID, pgGroesseID,
	); err != nil {
		return fmt.Errorf("property_group_translations: %w", err)
	}

	// ── Property Options ─────────────────────────────────────────────────────
	s.step("property_options")
	poSchwarzID := uuid.New()
	poWeissID := uuid.New()
	poBlauID := uuid.New()
	poSID := uuid.New()
	poMID := uuid.New()
	poLID := uuid.New()
	poXLID := uuid.New()
	// Farbe options: $1-$3 are option IDs, $4=group, $5=now
	if err := exec(ctx, tx, `
		INSERT INTO property_options (id, group_id, color_hex, position, created_at, updated_at) VALUES
		($1, $4, '#1a1a1a', 0, $5, $5),
		($2, $4, '#f5f5f5', 1, $5, $5),
		($3, $4, '#1e3a8a', 2, $5, $5)`,
		poSchwarzID, poWeissID, poBlauID, pgFarbeID, now,
	); err != nil {
		return fmt.Errorf("property_options (farbe): %w", err)
	}
	// Größe options: $1-$4 are option IDs, $5=group, $6=now
	if err := exec(ctx, tx, `
		INSERT INTO property_options (id, group_id, color_hex, position, created_at, updated_at) VALUES
		($1, $5, NULL, 0, $6, $6),
		($2, $5, NULL, 1, $6, $6),
		($3, $5, NULL, 2, $6, $6),
		($4, $5, NULL, 3, $6, $6)`,
		poSID, poMID, poLID, poXLID, pgGroesseID, now,
	); err != nil {
		return fmt.Errorf("property_options (groesse): %w", err)
	}
	// $1-$7 are option IDs
	if err := exec(ctx, tx, `
		INSERT INTO property_option_translations (option_id, locale, name) VALUES
		($1, 'de-DE', 'Schwarz'), ($1, 'en-US', 'Black'),
		($2, 'de-DE', 'Weiß'),   ($2, 'en-US', 'White'),
		($3, 'de-DE', 'Blau'),   ($3, 'en-US', 'Blue'),
		($4, 'de-DE', 'S'),      ($4, 'en-US', 'S'),
		($5, 'de-DE', 'M'),      ($5, 'en-US', 'M'),
		($6, 'de-DE', 'L'),      ($6, 'en-US', 'L'),
		($7, 'de-DE', 'XL'),     ($7, 'en-US', 'XL')`,
		poSchwarzID, poWeissID, poBlauID, poSID, poMID, poLID, poXLID,
	); err != nil {
		return fmt.Errorf("property_option_translations: %w", err)
	}

	// ── Products ─────────────────────────────────────────────────────────────
	s.step("products")

	// Product 1 – Smartphone Pro Max (799.00 € gross, 19 %)
	// net = round(79900 / 1.19) = 67143
	prodPhoneID := uuid.New()
	varPhoneSchwarzID := uuid.New()
	varPhoneWeissID := uuid.New()
	if err := s.insertProduct(ctx, tx, now,
		prodPhoneID, "PHONE-001", taxStandardID, 67143, 79900, 50, 195,
		[][2]string{
			{"de-DE", "Smartphone Pro Max"},
			{"en-US", "Smartphone Pro Max"},
		},
		[][2]string{
			{"de-DE", "Das neueste Flaggschiff-Smartphone mit AMOLED-Display, 5G und 4800-mAh-Akku."},
			{"en-US", "The latest flagship smartphone with AMOLED display, 5G and 4800 mAh battery."},
		},
		[][2]string{
			{"de-DE", "smartphone-pro-max"},
			{"en-US", "smartphone-pro-max-en"},
		},
	); err != nil {
		return fmt.Errorf("product phone: %w", err)
	}
	if err := exec(ctx, tx,
		`INSERT INTO product_categories (product_id, category_id, position) VALUES ($1,$2,0)`,
		prodPhoneID, catSmartphonesID,
	); err != nil {
		return fmt.Errorf("product_categories phone: %w", err)
	}
	if err := exec(ctx, tx,
		`INSERT INTO product_tags (product_id, tag_id) VALUES ($1,$2),($1,$3)`,
		prodPhoneID, tagNeuID, tagBestsellerID,
	); err != nil {
		return fmt.Errorf("product_tags phone: %w", err)
	}
	// Variants: Schwarz, Weiß – same price, 25 units each
	// $1,$2=variant IDs, $3=product, $4=now
	if err := exec(ctx, tx, `
		INSERT INTO product_variants (id, product_id, sku, price_net, price_gross, stock, active, custom_fields, created_at, updated_at) VALUES
		($1, $3, 'PHONE-001-SW', 67143, 79900, 25, true, '{}', $4, $4),
		($2, $3, 'PHONE-001-WS', 67143, 79900, 25, true, '{}', $4, $4)`,
		varPhoneSchwarzID, varPhoneWeissID, prodPhoneID, now,
	); err != nil {
		return fmt.Errorf("product_variants phone: %w", err)
	}
	// $1=varSchwarz, $2=varWeiss, $3=poSchwarz, $4=poWeiss
	if err := exec(ctx, tx,
		`INSERT INTO product_variant_options (variant_id, option_id) VALUES ($1,$3),($2,$4)`,
		varPhoneSchwarzID, varPhoneWeissID, poSchwarzID, poWeissID,
	); err != nil {
		return fmt.Errorf("product_variant_options phone: %w", err)
	}

	// Product 2 – Ultrabook X1 (1199.00 € gross, 19 %)
	// net = round(119900 / 1.19) = 100756
	prodLaptopID := uuid.New()
	if err := s.insertProduct(ctx, tx, now,
		prodLaptopID, "LAPTOP-001", taxStandardID, 100756, 119900, 20, 1800,
		[][2]string{
			{"de-DE", "Ultrabook X1"},
			{"en-US", "Ultrabook X1"},
		},
		[][2]string{
			{"de-DE", "Schlankes Business-Notebook mit 14\"-IPS-Display, Intel Core i7 und 16 GB RAM."},
			{"en-US", "Slim business notebook with 14\" IPS display, Intel Core i7 and 16 GB RAM."},
		},
		[][2]string{
			{"de-DE", "ultrabook-x1"},
			{"en-US", "ultrabook-x1-en"},
		},
	); err != nil {
		return fmt.Errorf("product laptop: %w", err)
	}
	if err := exec(ctx, tx,
		`INSERT INTO product_categories (product_id, category_id, position) VALUES ($1,$2,0)`,
		prodLaptopID, catLaptopsID,
	); err != nil {
		return fmt.Errorf("product_categories laptop: %w", err)
	}
	if err := exec(ctx, tx,
		`INSERT INTO product_tags (product_id, tag_id) VALUES ($1,$2)`,
		prodLaptopID, tagNeuID,
	); err != nil {
		return fmt.Errorf("product_tags laptop: %w", err)
	}

	// Product 3 – T-Shirt Basic (29.99 € gross, 19 %)
	// net = round(2999 / 1.19) = 2521
	prodShirtID := uuid.New()
	varShirtSID := uuid.New()
	varShirtMID := uuid.New()
	varShirtLID := uuid.New()
	if err := s.insertProduct(ctx, tx, now,
		prodShirtID, "SHIRT-001", taxStandardID, 2521, 2999, 200, 200,
		[][2]string{
			{"de-DE", "T-Shirt Basic"},
			{"en-US", "Basic T-Shirt"},
		},
		[][2]string{
			{"de-DE", "Klassisches Baumwoll-T-Shirt in verschiedenen Größen. 100 % Bio-Baumwolle, GOTS-zertifiziert."},
			{"en-US", "Classic cotton t-shirt in various sizes. 100 % organic cotton, GOTS certified."},
		},
		[][2]string{
			{"de-DE", "t-shirt-basic"},
			{"en-US", "basic-t-shirt"},
		},
	); err != nil {
		return fmt.Errorf("product shirt: %w", err)
	}
	if err := exec(ctx, tx,
		`INSERT INTO product_categories (product_id, category_id, position) VALUES ($1,$2,0)`,
		prodShirtID, catModeID,
	); err != nil {
		return fmt.Errorf("product_categories shirt: %w", err)
	}
	if err := exec(ctx, tx,
		`INSERT INTO product_tags (product_id, tag_id) VALUES ($1,$2),($1,$3)`,
		prodShirtID, tagSaleID, tagBioID,
	); err != nil {
		return fmt.Errorf("product_tags shirt: %w", err)
	}
	// Variants: S, M, L – same price, ~65 units each
	// $1-$3=variant IDs, $4=product, $5=now
	if err := exec(ctx, tx, `
		INSERT INTO product_variants (id, product_id, sku, price_net, price_gross, stock, active, custom_fields, created_at, updated_at) VALUES
		($1, $4, 'SHIRT-001-S', 2521, 2999, 70, true, '{}', $5, $5),
		($2, $4, 'SHIRT-001-M', 2521, 2999, 80, true, '{}', $5, $5),
		($3, $4, 'SHIRT-001-L', 2521, 2999, 50, true, '{}', $5, $5)`,
		varShirtSID, varShirtMID, varShirtLID, prodShirtID, now,
	); err != nil {
		return fmt.Errorf("product_variants shirt: %w", err)
	}
	// $1=varS, $2=varM, $3=varL, $4=poS, $5=poM, $6=poL
	if err := exec(ctx, tx, `
		INSERT INTO product_variant_options (variant_id, option_id) VALUES
		($1,$4), ($2,$5), ($3,$6)`,
		varShirtSID, varShirtMID, varShirtLID, poSID, poMID, poLID,
	); err != nil {
		return fmt.Errorf("product_variant_options shirt: %w", err)
	}

	// Product 4 – Leder-Handtasche (89.99 € gross, 19 %)
	// net = round(8999 / 1.19) = 7562
	prodBagID := uuid.New()
	if err := s.insertProduct(ctx, tx, now,
		prodBagID, "BAG-001", taxStandardID, 7562, 8999, 30, 500,
		[][2]string{
			{"de-DE", "Leder-Handtasche"},
			{"en-US", "Leather Handbag"},
		},
		[][2]string{
			{"de-DE", "Elegante Handtasche aus echtem Rindsleder mit Reißverschluss und Innentaschen."},
			{"en-US", "Elegant handbag made from genuine cowhide leather with zipper and inner pockets."},
		},
		[][2]string{
			{"de-DE", "leder-handtasche"},
			{"en-US", "leather-handbag"},
		},
	); err != nil {
		return fmt.Errorf("product bag: %w", err)
	}
	if err := exec(ctx, tx,
		`INSERT INTO product_categories (product_id, category_id, position) VALUES ($1,$2,1)`,
		prodBagID, catModeID,
	); err != nil {
		return fmt.Errorf("product_categories bag: %w", err)
	}
	if err := exec(ctx, tx,
		`INSERT INTO product_tags (product_id, tag_id) VALUES ($1,$2)`,
		prodBagID, tagBestsellerID,
	); err != nil {
		return fmt.Errorf("product_tags bag: %w", err)
	}

	// Product 5 – Kaffeemaschine Deluxe (199.99 € gross, 19 %)
	// net = round(19999 / 1.19) = 16806
	prodCoffeeID := uuid.New()
	if err := s.insertProduct(ctx, tx, now,
		prodCoffeeID, "COFFEE-001", taxStandardID, 16806, 19999, 15, 3500,
		[][2]string{
			{"de-DE", "Kaffeemaschine Deluxe"},
			{"en-US", "Deluxe Coffee Machine"},
		},
		[][2]string{
			{"de-DE", "Vollautomatische Kaffeemaschine mit integriertem Milchaufschäumer und 15-bar-Pumpe."},
			{"en-US", "Fully automatic coffee machine with integrated milk frother and 15-bar pump."},
		},
		[][2]string{
			{"de-DE", "kaffeemaschine-deluxe"},
			{"en-US", "deluxe-coffee-machine"},
		},
	); err != nil {
		return fmt.Errorf("product coffee: %w", err)
	}
	if err := exec(ctx, tx,
		`INSERT INTO product_categories (product_id, category_id, position) VALUES ($1,$2,1)`,
		prodCoffeeID, catElektronikID,
	); err != nil {
		return fmt.Errorf("product_categories coffee: %w", err)
	}
	if err := exec(ctx, tx,
		`INSERT INTO product_tags (product_id, tag_id) VALUES ($1,$2)`,
		prodCoffeeID, tagNeuID,
	); err != nil {
		return fmt.Errorf("product_tags coffee: %w", err)
	}

	// ── Shipping Methods ─────────────────────────────────────────────────────
	s.step("shipping_methods")
	shipStandardID := uuid.New() // 3.99 € gross
	shipExpressID := uuid.New()  // 7.99 € gross
	shipFreeID := uuid.New()     // 0.00 € (free from 50 €)
	// $1-$3=IDs, $4=now; net = round(gross/1.19)
	if err := exec(ctx, tx, `
		INSERT INTO shipping_methods (id, active, price_net, price_gross, custom_fields, created_at, updated_at) VALUES
		($1, true, 335, 399, '{}', $4, $4),
		($2, true, 671, 799, '{}', $4, $4),
		($3, true,   0,   0, '{}', $4, $4)`,
		shipStandardID, shipExpressID, shipFreeID, now,
	); err != nil {
		return fmt.Errorf("shipping_methods: %w", err)
	}
	if err := exec(ctx, tx, `
		INSERT INTO shipping_method_translations (shipping_method_id, locale, name, description) VALUES
		($1, 'de-DE', 'Standardversand',    'Lieferung in 3–5 Werktagen'),
		($1, 'en-US', 'Standard Shipping',  'Delivery in 3–5 business days'),
		($2, 'de-DE', 'Expressversand',     'Lieferung am nächsten Werktag'),
		($2, 'en-US', 'Express Shipping',   'Next business day delivery'),
		($3, 'de-DE', 'Kostenloser Versand','Gratis-Lieferung ab 50 €'),
		($3, 'en-US', 'Free Shipping',      'Free delivery from €50')`,
		shipStandardID, shipExpressID, shipFreeID,
	); err != nil {
		return fmt.Errorf("shipping_method_translations: %w", err)
	}

	// ── Payment Methods ──────────────────────────────────────────────────────
	s.step("payment_methods")
	pmVorkasseID := uuid.New()
	pmKreditkarteID := uuid.New()
	if err := exec(ctx, tx, `
		INSERT INTO payment_methods (id, provider, active, config, custom_fields, created_at, updated_at) VALUES
		($1, 'bank_transfer', true, NULL, '{}', $3, $3),
		($2, 'credit_card',  true, NULL, '{}', $3, $3)`,
		pmVorkasseID, pmKreditkarteID, now,
	); err != nil {
		return fmt.Errorf("payment_methods: %w", err)
	}
	if err := exec(ctx, tx, `
		INSERT INTO payment_method_translations (payment_method_id, locale, name, description) VALUES
		($1, 'de-DE', 'Vorkasse',      'Banküberweisung vor Versand'),
		($1, 'en-US', 'Bank Transfer', 'Bank transfer before shipment'),
		($2, 'de-DE', 'Kreditkarte',   'Visa, Mastercard oder American Express'),
		($2, 'en-US', 'Credit Card',   'Visa, Mastercard or American Express')`,
		pmVorkasseID, pmKreditkarteID,
	); err != nil {
		return fmt.Errorf("payment_method_translations: %w", err)
	}

	// ── Customers ────────────────────────────────────────────────────────────
	s.step("customers")
	pwHash, err := auth.HashPassword("Demo1234!")
	if err != nil {
		return fmt.Errorf("hashing demo password: %w", err)
	}
	custMariaID := uuid.New()
	custThomasID := uuid.New()
	// $1,$2=customer IDs, $3=password hash, $4=now
	if err := exec(ctx, tx, `
		INSERT INTO customers (id, email, password_hash, first_name, last_name, active, custom_fields, created_at, updated_at) VALUES
		($1, 'maria@example.com',  $3, 'Maria',  'Müller',  true, '{}', $4, $4),
		($2, 'thomas@example.com', $3, 'Thomas', 'Schmidt', true, '{}', $4, $4)`,
		custMariaID, custThomasID, pwHash, now,
	); err != nil {
		return fmt.Errorf("customers: %w", err)
	}
	addrMariaID := uuid.New()
	addrThomasID := uuid.New()
	// $1,$2=address IDs, $3,$4=customer IDs, $5=now
	if err := exec(ctx, tx, `
		INSERT INTO customer_addresses (id, customer_id, first_name, last_name, street, city, zip, country_code, phone, created_at, updated_at) VALUES
		($1, $3, 'Maria',  'Müller',  'Musterstraße 42', 'Berlin',  '10115', 'DE', '+49 30 12345678', $5, $5),
		($2, $4, 'Thomas', 'Schmidt', 'Musterallee 10',  'München', '80333', 'DE', '+49 89 98765432', $5, $5)`,
		addrMariaID, addrThomasID, custMariaID, custThomasID, now,
	); err != nil {
		return fmt.Errorf("customer_addresses: %w", err)
	}
	// Set default addresses
	if err := exec(ctx, tx, `
		UPDATE customers SET default_billing_address_id=$1, default_shipping_address_id=$1 WHERE id=$2`,
		addrMariaID, custMariaID,
	); err != nil {
		return fmt.Errorf("customer default address (maria): %w", err)
	}
	if err := exec(ctx, tx, `
		UPDATE customers SET default_billing_address_id=$1, default_shipping_address_id=$1 WHERE id=$2`,
		addrThomasID, custThomasID,
	); err != nil {
		return fmt.Errorf("customer default address (thomas): %w", err)
	}

	// ── Discounts ────────────────────────────────────────────────────────────
	s.step("discounts")
	validFrom := now
	validUntil := now.AddDate(1, 0, 0) // valid for 1 year
	if err := exec(ctx, tx, `
		INSERT INTO discounts (id, code, type, value, min_order_value, max_uses, used_count, valid_from, valid_until, active, conditions, created_at, updated_at) VALUES
		($1, 'DEMO10',   'percentage', 1000,  NULL, 100,  0, $4, $5, true, '{}', $4, $4),
		($2, 'SOMMER20', 'percentage', 2000,  5000,  50,  0, $4, $5, true, '{}', $4, $4),
		($3, 'FIXED5',   'fixed',       500,  2000,  NULL, 0, $4, $5, true, '{}', $4, $4)`,
		uuid.New(), uuid.New(), uuid.New(), validFrom, validUntil,
	); err != nil {
		return fmt.Errorf("discounts: %w", err)
	}

	// ── Orders ───────────────────────────────────────────────────────────────
	s.step("orders")

	mariaAddr := `{"first_name":"Maria","last_name":"Müller","street":"Musterstraße 42","city":"Berlin","zip":"10115","country_code":"DE","phone":"+49 30 12345678"}`
	thomasAddr := `{"first_name":"Thomas","last_name":"Schmidt","street":"Musterallee 10","city":"München","zip":"80333","country_code":"DE","phone":"+49 89 98765432"}`

	// Order 1 – Maria, 1× Smartphone Pro Max Schwarz, confirmed
	// subtotal_net=67143, subtotal_gross=79900, shipping_cost=399
	// tax: (79900-67143)+(399-335) = 12757+64 = 12821
	// total: 79900+399 = 80299
	order1ID := uuid.New()
	if err := exec(ctx, tx, `
		INSERT INTO orders (id, order_number, customer_id, status, currency,
		                    subtotal_net, subtotal_gross, shipping_cost, tax_total, total,
		                    billing_address, shipping_address,
		                    payment_method_id, shipping_method_id,
		                    notes, custom_fields, created_at, updated_at)
		VALUES ($1, 'ORD-20260301-DEMO1', $2, 'confirmed', 'EUR',
		        67143, 79900, 399, 12821, 80299,
		        $3, $3,
		        $4, $5,
		        NULL, '{}', $6, $6)`,
		order1ID, custMariaID, mariaAddr, pmKreditkarteID, shipStandardID, now,
	); err != nil {
		return fmt.Errorf("order 1: %w", err)
	}
	if err := exec(ctx, tx, `
		INSERT INTO order_items (id, order_id, product_id, variant_id, sku, name, quantity,
		                         unit_price_net, unit_price_gross, total_net, total_gross, tax_rate)
		VALUES ($1, $2, $3, $4, 'PHONE-001-SW', 'Smartphone Pro Max – Schwarz',
		        1, 67143, 79900, 67143, 79900, 1900)`,
		uuid.New(), order1ID, prodPhoneID, varPhoneSchwarzID,
	); err != nil {
		return fmt.Errorf("order_items 1: %w", err)
	}
	// Status history: pending → confirmed
	if err := exec(ctx, tx, `
		INSERT INTO order_status_history (id, order_id, from_status, to_status, comment, created_at) VALUES
		($1, $3, NULL,      'pending',   'Bestellung eingegangen', $5),
		($2, $3, 'pending', 'confirmed', 'Zahlung per Kreditkarte bestätigt', $5)`,
		uuid.New(), uuid.New(), order1ID, now, now,
	); err != nil {
		return fmt.Errorf("order_status_history 1: %w", err)
	}
	if err := exec(ctx, tx, `
		INSERT INTO payment_transactions (id, order_id, payment_method_id, status, amount, currency, provider_reference, created_at)
		VALUES ($1, $2, $3, 'completed', 80299, 'EUR', 'TXN-DEMO-001', $4)`,
		uuid.New(), order1ID, pmKreditkarteID, now,
	); err != nil {
		return fmt.Errorf("payment_transactions 1: %w", err)
	}

	// Order 2 – Thomas, 2× T-Shirt Basic S, pending (Vorkasse)
	// subtotal_net=2*2521=5042, subtotal_gross=2*2999=5998, shipping_cost=399
	// tax: (5998-5042)+(399-335) = 956+64 = 1020
	// total: 5998+399 = 6397
	order2ID := uuid.New()
	if err := exec(ctx, tx, `
		INSERT INTO orders (id, order_number, customer_id, status, currency,
		                    subtotal_net, subtotal_gross, shipping_cost, tax_total, total,
		                    billing_address, shipping_address,
		                    payment_method_id, shipping_method_id,
		                    notes, custom_fields, created_at, updated_at)
		VALUES ($1, 'ORD-20260301-DEMO2', $2, 'pending', 'EUR',
		        5042, 5998, 399, 1020, 6397,
		        $3, $3,
		        $4, $5,
		        'Bitte schnell liefern!', '{}', $6, $6)`,
		order2ID, custThomasID, thomasAddr, pmVorkasseID, shipStandardID, now,
	); err != nil {
		return fmt.Errorf("order 2: %w", err)
	}
	if err := exec(ctx, tx, `
		INSERT INTO order_items (id, order_id, product_id, variant_id, sku, name, quantity,
		                         unit_price_net, unit_price_gross, total_net, total_gross, tax_rate)
		VALUES ($1, $2, $3, $4, 'SHIRT-001-S', 'T-Shirt Basic – S',
		        2, 2521, 2999, 5042, 5998, 1900)`,
		uuid.New(), order2ID, prodShirtID, varShirtSID,
	); err != nil {
		return fmt.Errorf("order_items 2: %w", err)
	}
	if err := exec(ctx, tx, `
		INSERT INTO order_status_history (id, order_id, from_status, to_status, comment, created_at)
		VALUES ($1, $2, NULL, 'pending', 'Bestellung eingegangen', $3)`,
		uuid.New(), order2ID, now,
	); err != nil {
		return fmt.Errorf("order_status_history 2: %w", err)
	}

	return nil
}

// insertProduct inserts a product row and its translations.
// translations, descriptions and slugs are parallel slices of [locale, value] pairs.
func (s *Seeder) insertProduct(
	ctx context.Context, tx pgx.Tx, now time.Time,
	id uuid.UUID, sku string, taxRuleID uuid.UUID,
	priceNet, priceGross, stock, weightGrams int,
	names [][2]string,
	descriptions [][2]string,
	slugs [][2]string,
) error {
	if err := exec(ctx, tx, `
		INSERT INTO products (id, sku, active, price_net, price_gross, currency, tax_rule_id, stock, weight, custom_fields, metadata, created_at, updated_at)
		VALUES ($1, $2, true, $3, $4, 'EUR', $5, $6, $7, '{}', '{}', $8, $8)`,
		id, sku, priceNet, priceGross, taxRuleID, stock, weightGrams, now,
	); err != nil {
		return fmt.Errorf("product row: %w", err)
	}
	for i, name := range names {
		locale := name[0]
		desc := ""
		if i < len(descriptions) {
			desc = descriptions[i][1]
		}
		slug := ""
		if i < len(slugs) {
			slug = slugs[i][1]
		}
		if err := exec(ctx, tx, `
			INSERT INTO product_translations (product_id, locale, name, description, slug, meta_title, meta_description)
			VALUES ($1, $2, $3, $4, $5, $3, $4)`,
			id, locale, name[1], desc, slug,
		); err != nil {
			return fmt.Errorf("product_translations [%s]: %w", locale, err)
		}
	}
	return nil
}
