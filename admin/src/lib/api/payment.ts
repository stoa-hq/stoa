import { api } from './client';
import type { ApiResponse, PaymentMethod, PaginationParams } from '$lib/types';

const BASE = '/admin/payment-methods';

export const paymentApi = {
	list: (params: PaginationParams = {}) => {
		const q = new URLSearchParams();
		if (params.limit) q.set('limit', String(params.limit ?? 100));
		return api.get<ApiResponse<PaymentMethod[]>>(`${BASE}?${q}`);
	},
	get: (id: string) => api.get<ApiResponse<PaymentMethod>>(`${BASE}/${id}`),
	create: (data: Partial<PaymentMethod>) => api.post<ApiResponse<PaymentMethod>>(BASE, data),
	update: (id: string, data: Partial<PaymentMethod>) =>
		api.put<ApiResponse<PaymentMethod>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`)
};
