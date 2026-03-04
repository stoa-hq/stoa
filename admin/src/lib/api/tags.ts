import { api } from './client';
import type { ApiResponse, Tag, PaginationParams } from '$lib/types';

const BASE = '/admin/tags';

export const tagsApi = {
	list: (params: PaginationParams = {}) => {
		const q = new URLSearchParams();
		if (params.limit) q.set('limit', String(params.limit ?? 200));
		return api.get<ApiResponse<Tag[]>>(`${BASE}?${q}`);
	},
	get: (id: string) => api.get<ApiResponse<Tag>>(`${BASE}/${id}`),
	create: (data: Partial<Tag>) => api.post<ApiResponse<Tag>>(BASE, data),
	update: (id: string, data: Partial<Tag>) =>
		api.put<ApiResponse<Tag>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`)
};
