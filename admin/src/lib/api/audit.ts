import { api } from './client';
import type { ApiResponse, AuditLog, PaginationParams } from '$lib/types';

export const auditApi = {
	list: (params: PaginationParams & { entity_type?: string } = {}) => {
		const q = new URLSearchParams();
		if (params.page) q.set('page', String(params.page));
		if (params.limit) q.set('limit', String(params.limit));
		if (params.entity_type) q.set('filter[entity_type]', params.entity_type);
		return api.get<ApiResponse<AuditLog[]>>(`/admin/audit-log?${q}`);
	}
};
