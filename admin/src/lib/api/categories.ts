import { api } from './client';
import type { ApiResponse, Category, CreateCategoryRequest, PaginationParams } from '$lib/types';

const BASE = '/admin/categories';

export const categoriesApi = {
	list: (params: PaginationParams = {}) => {
		const q = new URLSearchParams();
		if (params.page) q.set('page', String(params.page));
		if (params.limit) q.set('limit', String(params.limit ?? 100));
		return api.get<ApiResponse<Category[]>>(`${BASE}?${q}`);
	},
	get: (id: string) => api.get<ApiResponse<Category>>(`${BASE}/${id}`),
	create: (data: CreateCategoryRequest) => api.post<ApiResponse<Category>>(BASE, data),
	update: (id: string, data: CreateCategoryRequest) =>
		api.put<ApiResponse<Category>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`)
};
