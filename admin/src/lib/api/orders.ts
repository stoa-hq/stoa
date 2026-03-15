import { api } from './client';
import type { ApiResponse, Order, PaymentTransaction, PaginationParams } from '$lib/types';

const BASE = '/admin/orders';

export const ordersApi = {
	list: (params: PaginationParams & { status?: string; customer_id?: string; search?: string } = {}) => {
		const q = new URLSearchParams();
		if (params.page) q.set('page', String(params.page));
		if (params.limit) q.set('limit', String(params.limit));
		if (params.sort) q.set('sort', params.sort);
		if (params.order) q.set('order', params.order);
		if (params.status) q.set('status', params.status);
		if (params.customer_id) q.set('customer_id', params.customer_id);
		if (params.search) q.set('search', params.search);
		return api.get<ApiResponse<Order[]>>(`${BASE}?${q}`);
	},
	get: (id: string) => api.get<ApiResponse<Order>>(`${BASE}/${id}`),
	updateStatus: (id: string, status: string, comment?: string) =>
		api.put<ApiResponse<Order>>(`${BASE}/${id}/status`, { status, comment }),
	getTransactions: (orderId: string) =>
		api.get<ApiResponse<PaymentTransaction[]>>(`${BASE}/${orderId}/transactions`)
};
