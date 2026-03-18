import { api } from './client';
import type { ApiResponse } from '$lib/types';

export interface StoreSettings {
	store_name: string;
	store_description: string;
	logo_url: string | null;
	favicon_url: string | null;
	contact_email: string | null;
	currency: string;
	country: string | null;
	timezone: string;
	copyright_text: string;
	maintenance_mode: boolean;
	created_at: string;
	updated_at: string;
}

export type UpdateSettingsRequest = Omit<StoreSettings, 'created_at' | 'updated_at'>;

const BASE = '/admin/settings';

export const settingsApi = {
	get: () => api.get<ApiResponse<StoreSettings>>(BASE),
	update: (data: UpdateSettingsRequest) => api.put<ApiResponse<StoreSettings>>(BASE, data)
};
