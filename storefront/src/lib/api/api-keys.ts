import { api, type ApiResponse } from './client';

export interface StoreAPIKey {
	id: string;
	name: string;
	key_type: string;
	permissions: string[];
	active: boolean;
	customer_id: string;
	last_used_at: string | null;
	created_at: string;
}

export interface CreateKeyResponse extends StoreAPIKey {
	key: string;
}

export const apiKeysApi = {
	list(): Promise<ApiResponse<StoreAPIKey[]>> {
		return api.get<StoreAPIKey[]>('/store/api-keys');
	},
	create(name: string, permissions?: string[]): Promise<ApiResponse<CreateKeyResponse>> {
		return api.post<CreateKeyResponse>('/store/api-keys', { name, permissions });
	},
	revoke(id: string): Promise<ApiResponse<{ message: string }>> {
		return api.delete<{ message: string }>(`/store/api-keys/${id}`);
	}
};
