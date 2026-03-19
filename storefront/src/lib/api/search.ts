import { api, type ApiResponse } from './client';

export interface SearchResult {
	id: string;
	type: string;
	score: number;
	title: string;
	description?: string;
	slug?: string;
	data?: Record<string, unknown>;
}

export const searchApi = {
	search(params: {
		q: string;
		locale?: string;
		page?: number;
		limit?: number;
		type?: string;
	}): Promise<ApiResponse<SearchResult[]>> {
		const query = new URLSearchParams();
		query.set('q', params.q);
		if (params.locale) query.set('locale', params.locale);
		if (params.page) query.set('page', String(params.page));
		if (params.limit) query.set('limit', String(params.limit));
		if (params.type) query.set('type', params.type);
		return api.get<SearchResult[]>(`/store/search?${query.toString()}`);
	}
};
