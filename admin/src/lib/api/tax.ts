import { api } from './client';
import type { ApiResponse, TaxRule, PaginationParams } from '$lib/types';

const BASE = '/admin/tax-rules';

export const taxApi = {
	list: (params: PaginationParams = {}) => {
		const q = new URLSearchParams();
		if (params.page) q.set('page', String(params.page));
		if (params.limit) q.set('limit', String(params.limit ?? 100));
		return api.get<ApiResponse<TaxRule[]>>(`${BASE}?${q}`);
	},
	get: (id: string) => api.get<ApiResponse<TaxRule>>(`${BASE}/${id}`),
	create: (data: Partial<TaxRule>) => api.post<ApiResponse<TaxRule>>(BASE, data),
	update: (id: string, data: Partial<TaxRule>) =>
		api.put<ApiResponse<TaxRule>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`)
};
