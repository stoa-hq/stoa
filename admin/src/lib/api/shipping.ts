import { api } from './client';
import type { ApiResponse, ShippingMethod, PaginationParams } from '$lib/types';

const BASE = '/admin/shipping-methods';

export const shippingApi = {
	list: (params: PaginationParams = {}) => {
		const q = new URLSearchParams();
		if (params.limit) q.set('limit', String(params.limit ?? 100));
		return api.get<ApiResponse<ShippingMethod[]>>(`${BASE}?${q}`);
	},
	get: (id: string) => api.get<ApiResponse<ShippingMethod>>(`${BASE}/${id}`),
	create: (data: Partial<ShippingMethod>) => api.post<ApiResponse<ShippingMethod>>(BASE, data),
	update: (id: string, data: Partial<ShippingMethod>) =>
		api.put<ApiResponse<ShippingMethod>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`)
};
