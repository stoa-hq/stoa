import { api } from './client';
import type { ApiResponse, Warehouse, WarehouseStock, PaginationParams } from '$lib/types';

const BASE = '/admin/warehouses';

export const warehousesApi = {
	list: (params: PaginationParams = {}) => {
		const q = new URLSearchParams();
		if (params.limit) q.set('limit', String(params.limit ?? 100));
		if (params.page) q.set('page', String(params.page));
		return api.get<ApiResponse<Warehouse[]>>(`${BASE}?${q}`);
	},
	get: (id: string) => api.get<ApiResponse<Warehouse>>(`${BASE}/${id}`),
	create: (data: Partial<Warehouse>) => api.post<ApiResponse<Warehouse>>(BASE, data),
	update: (id: string, data: Partial<Warehouse>) =>
		api.put<ApiResponse<Warehouse>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`),
	getStock: (id: string) => api.get<ApiResponse<WarehouseStock[]>>(`${BASE}/${id}/stock`),
	setStock: (id: string, items: { product_id: string; variant_id?: string; quantity: number; reference?: string }[]) =>
		api.put<ApiResponse<WarehouseStock[]>>(`${BASE}/${id}/stock`, { items }),
	removeStock: (warehouseId: string, stockId: string) =>
		api.delete<void>(`${BASE}/${warehouseId}/stock/${stockId}`),
	getProductStock: (productId: string) =>
		api.get<ApiResponse<WarehouseStock[]>>(`/admin/products/${productId}/stock`),
};
