import { api } from './client';
import type { ApiResponse, APIKey, APIKeyCreateResponse } from '$lib/types';

const BASE = '/admin/api-keys';

export const apiKeysApi = {
	list: (all?: boolean) => api.get<ApiResponse<APIKey[]>>(`${BASE}${all ? '?all=true' : ''}`),
	create: (data: { name: string; permissions: string[] }) =>
		api.post<ApiResponse<APIKeyCreateResponse>>(BASE, data),
	revoke: (id: string) => api.delete<ApiResponse<{ message: string }>>(`${BASE}/${id}`)
};
