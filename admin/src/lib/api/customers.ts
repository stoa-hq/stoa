import { api } from './client';
import type { ApiResponse, Customer, PaginationParams } from '$lib/types';

const BASE = '/admin/customers';

export const customersApi = {
	list: (params: PaginationParams = {}) => {
		const q = new URLSearchParams();
		if (params.page) q.set('page', String(params.page));
		if (params.limit) q.set('limit', String(params.limit));
		if (params.sort) q.set('sort', params.sort);
		if (params.order) q.set('order', params.order);
		return api.get<ApiResponse<Customer[]>>(`${BASE}?${q}`);
	},
	get: (id: string) => api.get<ApiResponse<Customer>>(`${BASE}/${id}`),
	create: (data: Partial<Customer>) => api.post<ApiResponse<Customer>>(BASE, data),
	update: (id: string, data: Partial<Customer>) =>
		api.put<ApiResponse<Customer>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`)
};
