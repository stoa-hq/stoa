import { api, type ApiResponse } from './client';

export interface ProductTranslation {
	locale: string;
	name: string;
	description: string;
	slug: string;
	meta_title: string;
	meta_description: string;
}

export interface ProductVariantOption {
	id: string;
	group_id: string;
	color_hex?: string;
	position: number;
	translations?: { locale: string; name: string }[];
}

export interface ProductVariant {
	id: string;
	sku: string;
	price_net?: number;
	price_gross?: number;
	stock: number;
	active: boolean;
	options?: ProductVariantOption[];
}

export interface Product {
	id: string;
	sku: string;
	active: boolean;
	price_net: number;
	price_gross: number;
	currency: string;
	stock: number;
	has_variants: boolean;
	translations: ProductTranslation[];
	variants?: ProductVariant[];
	media?: { media_id: string; position: number; url?: string }[];
	categories?: string[];
	tags?: string[];
}

export function getTranslation(product: Product, locale = 'de-DE'): ProductTranslation {
	return (
		product.translations.find((t) => t.locale === locale) ??
		product.translations.find((t) => t.locale === 'de-DE') ??
		product.translations[0] ?? { locale: '', name: product.sku, description: '', slug: product.sku, meta_title: '', meta_description: '' }
	);
}

export const productsApi = {
	list(params: { page?: number; limit?: number; search?: string; category_id?: string } = {}): Promise<ApiResponse<{ items: Product[] }>> {
		const q = new URLSearchParams();
		if (params.page) q.set('page', String(params.page));
		if (params.limit) q.set('limit', String(params.limit));
		if (params.search) q.set('search', params.search);
		if (params.category_id) q.set('category_id', params.category_id);
		const qs = q.toString();
		return api.get<{ items: Product[] }>(`/store/products${qs ? `?${qs}` : ''}`);
	},

	getBySlug(slug: string): Promise<ApiResponse<Product>> {
		return api.get<Product>(`/store/products/${slug}`);
	},

	getById(id: string): Promise<ApiResponse<Product>> {
		return api.get<Product>(`/store/products/id/${id}`);
	}
};
