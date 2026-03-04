import { api } from './client';
import type { ApiResponse, Discount, PaginationParams } from '$lib/types';

const BASE = '/admin/discounts';

export const discountsApi = {
	list: (params: PaginationParams = {}) => {
		const q = new URLSearchParams();
		if (params.page) q.set('page', String(params.page));
		if (params.limit) q.set('limit', String(params.limit));
		return api.get<ApiResponse<Discount[]>>(`${BASE}?${q}`);
	},
	get: (id: string) => api.get<ApiResponse<Discount>>(`${BASE}/${id}`),
	create: (data: Partial<Discount>) => api.post<ApiResponse<Discount>>(BASE, data),
	update: (id: string, data: Partial<Discount>) =>
		api.put<ApiResponse<Discount>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`)
};
