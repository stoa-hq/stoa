import { api } from './client';
import type { ApiResponse, Media, PaginationParams } from '$lib/types';

const BASE = '/admin/media';

export const mediaApi = {
	list: (params: PaginationParams = {}) => {
		const q = new URLSearchParams();
		if (params.page) q.set('page', String(params.page));
		if (params.limit) q.set('limit', String(params.limit));
		return api.get<ApiResponse<Media[]>>(`${BASE}?${q}`);
	},
	upload: (file: File, altText?: string) => {
		const fd = new FormData();
		fd.append('file', file);
		if (altText) fd.append('alt_text', altText);
		return api.upload<ApiResponse<Media>>(BASE, fd);
	},
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`)
};
