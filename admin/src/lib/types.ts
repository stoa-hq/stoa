// ── Shared ───────────────────────────────────────────────────────────────────

export interface Meta {
	total: number;
	page: number;
	limit: number;
	pages: number;
}

export interface ApiResponse<T> {
	data: T;
	meta?: Meta;
	errors?: ApiError[];
}

export interface ApiError {
	code: string;
	detail: string;
	field?: string;
}

// ── Auth ─────────────────────────────────────────────────────────────────────

export interface LoginResponse {
	access_token: string;
	refresh_token: string;
	expires_in: number;
	token_type: string;
}

export interface AdminUser {
	id: string;
	email: string;
	role: string;
}

// ── Product ───────────────────────────────────────────────────────────────────

export interface ProductTranslation {
	locale: string;
	name: string;
	description: string;
	slug: string;
	meta_title: string;
	meta_description: string;
}

export interface Product {
	id: string;
	sku: string;
	active: boolean;
	price_net: number;
	price_gross: number;
	currency: string;
	stock: number;
	weight: number;
	tax_rule_id?: string;
	custom_fields?: Record<string, unknown>;
	created_at: string;
	updated_at: string;
	translations: ProductTranslation[];
	variants?: ProductVariant[];
	categories?: string[];
	tags?: string[];
}

export interface ProductListResponse {
	items: Product[];
}

export interface CreateProductRequest {
	sku?: string;
	active?: boolean;
	price_net?: number;
	price_gross?: number;
	currency: string;
	stock?: number;
	weight?: number;
	tax_rule_id?: string;
	translations: {
		locale: string;
		name: string;
		slug: string;
		description?: string;
	}[];
	category_ids?: string[];
	tag_ids?: string[];
}

export interface ProductVariant {
	id: string;
	product_id: string;
	sku: string;
	price_net: number | null;
	price_gross: number | null;
	stock: number;
	active: boolean;
}

export interface BulkImportOptionInput {
	group_name: string;
	option_name: string;
	locale: string;
}

export interface BulkImportVariantInput {
	sku: string;
	active: boolean;
	stock: number;
	price_net?: number;
	price_gross?: number;
	options: BulkImportOptionInput[];
}

export interface BulkCreateProductRequest extends CreateProductRequest {
	variants?: BulkImportVariantInput[];
}

export interface BulkRequest {
	products: BulkCreateProductRequest[];
}

export interface BulkResult {
	index: number;
	sku?: string;
	success: boolean;
	id?: string;
	errors?: string[];
}

export interface BulkResponse {
	results: BulkResult[];
	total: number;
	succeeded: number;
	failed: number;
}

// ── Category ─────────────────────────────────────────────────────────────────

export interface CategoryTranslation {
	locale: string;
	name: string;
	description: string;
	slug: string;
}

export interface Category {
	id: string;
	parent_id?: string;
	position: number;
	active: boolean;
	translations: CategoryTranslation[];
	children?: Category[];
	created_at: string;
	updated_at: string;
}

export interface CreateCategoryRequest {
	parent_id?: string | null;
	position?: number;
	active?: boolean;
	translations: { locale: string; name: string; slug: string; description?: string }[];
}

// ── Customer ──────────────────────────────────────────────────────────────────

export interface Customer {
	id: string;
	email: string;
	first_name: string;
	last_name: string;
	active: boolean;
	created_at: string;
	updated_at: string;
}

// ── Order ─────────────────────────────────────────────────────────────────────

export interface Order {
	id: string;
	order_number: string;
	customer_id?: string;
	status: string;
	currency: string;
	subtotal_net: number;
	subtotal_gross: number;
	shipping_cost: number;
	tax_total: number;
	total: number;
	payment_method_id?: string;
	shipping_method_id?: string;
	notes?: string;
	created_at: string;
	updated_at: string;
}

export interface OrderItem {
	id: string;
	order_id: string;
	product_id: string;
	variant_id?: string;
	sku: string;
	name: string;
	quantity: number;
	unit_price_gross: number;
	total_gross: number;
}

// ── Media ─────────────────────────────────────────────────────────────────────

export interface Media {
	id: string;
	filename: string;
	mime_type: string;
	size: number;
	storage_path: string;
	alt_text?: string;
	url?: string;
	created_at: string;
}

// ── Tax ───────────────────────────────────────────────────────────────────────

export interface TaxRule {
	id: string;
	name: string;
	rate: number; // basis points: 1900 = 19.00%
	country_code?: string;
	type: string;
	created_at: string;
	updated_at: string;
}

// ── Shipping ─────────────────────────────────────────────────────────────────

export interface ShippingMethodTranslation {
	locale: string;
	name: string;
	description: string;
}

export interface ShippingMethod {
	id: string;
	active: boolean;
	price_net: number;
	price_gross: number;
	translations: ShippingMethodTranslation[];
	created_at: string;
	updated_at: string;
}

// ── Payment ───────────────────────────────────────────────────────────────────

export interface PaymentMethodTranslation {
	locale: string;
	name: string;
	description: string;
}

export interface PaymentMethod {
	id: string;
	provider: string;
	active: boolean;
	translations: PaymentMethodTranslation[];
	created_at: string;
	updated_at: string;
}

// ── Discount ─────────────────────────────────────────────────────────────────

export interface Discount {
	id: string;
	code: string;
	type: 'percentage' | 'fixed';
	value: number;
	min_order_value?: number;
	max_uses?: number;
	used_count: number;
	valid_from?: string;
	valid_until?: string;
	active: boolean;
	created_at: string;
	updated_at: string;
}

// ── Tag ───────────────────────────────────────────────────────────────────────

export interface Tag {
	id: string;
	name: string;
	slug: string;
	created_at: string;
	updated_at: string;
}

// ── Audit ─────────────────────────────────────────────────────────────────────

export interface AuditLog {
	id: string;
	user_id: string;
	user_type: string;
	action: string;
	entity_type: string;
	entity_id: string;
	changes?: Record<string, unknown>;
	ip_address?: string;
	created_at: string;
}

// ── Pagination ────────────────────────────────────────────────────────────────

export interface PaginationParams {
	page?: number;
	limit?: number;
	sort?: string;
	order?: 'asc' | 'desc';
}
