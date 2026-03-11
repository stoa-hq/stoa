import { get } from 'svelte/store';
import { api } from './client';
import { authStore } from '$lib/stores/auth';
import type { ApiResponse, Product, ProductListResponse, CreateProductRequest, ProductVariant, PaginationParams, BulkRequest, BulkResponse } from '$lib/types';

const BASE = '/admin/products';

export interface ProductFilters extends PaginationParams {
	active?: boolean;
	search?: string;
}

export const productsApi = {
	list: (params: ProductFilters = {}) => {
		const q = new URLSearchParams();
		if (params.page) q.set('page', String(params.page));
		if (params.limit) q.set('limit', String(params.limit));
		if (params.sort) q.set('sort', params.sort);
		if (params.order) q.set('order', params.order);
		if (params.active !== undefined) q.set('filter[active]', String(params.active));
		if (params.search) q.set('q', params.search);
		return api.get<ApiResponse<ProductListResponse>>(`${BASE}?${q}`);
	},
	get: (id: string) => api.get<ApiResponse<Product>>(`${BASE}/${id}`),
	create: (data: CreateProductRequest) => api.post<ApiResponse<Product>>(BASE, data),
	update: (id: string, data: Partial<Product>) => api.put<ApiResponse<Product>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`),
	createVariant: (productId: string, data: Partial<ProductVariant>) =>
		api.post<ApiResponse<ProductVariant>>(`${BASE}/${productId}/variants`, data),
	updateVariant: (productId: string, variantId: string, data: Partial<ProductVariant>) =>
		api.put<ApiResponse<ProductVariant>>(`${BASE}/${productId}/variants/${variantId}`, data),
	deleteVariant: (productId: string, variantId: string) =>
		api.delete<void>(`${BASE}/${productId}/variants/${variantId}`),
	bulk: (data: BulkRequest) =>
		api.post<ApiResponse<BulkResponse>>(`${BASE}/bulk`, data),
	importCSV: (file: File) => {
		const formData = new FormData();
		formData.append('file', file);
		return api.upload<ApiResponse<BulkResponse>>('/admin/products/import', formData);
	},
	downloadTemplate: () =>
		fetch('/api/v1/admin/products/import/template', {
			headers: { Authorization: `Bearer ${get(authStore).accessToken ?? ''}` }
		}),
};
