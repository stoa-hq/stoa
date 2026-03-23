import { api } from './client';
import type { ApiResponse } from '$lib/types';

export interface PropertyGroupTranslationInput {
	locale: string;
	name: string;
}

export interface CreatePropertyGroupRequest {
	identifier: string;
	position?: number;
	translations: PropertyGroupTranslationInput[];
}

export interface PropertyOptionTranslationInput {
	locale: string;
	name: string;
}

export interface CreatePropertyOptionRequest {
	position?: number;
	color_hex?: string;
	translations: PropertyOptionTranslationInput[];
}

export interface PropertyOptionTranslation {
	locale: string;
	name: string;
}

export interface PropertyOption {
	id: string;
	group_id: string;
	color_hex?: string;
	position: number;
	translations?: PropertyOptionTranslation[];
}

export interface PropertyGroupTranslation {
	locale: string;
	name: string;
}

export interface PropertyGroup {
	id: string;
	identifier: string;
	position: number;
	created_at: string;
	updated_at: string;
	translations?: PropertyGroupTranslation[];
	options?: PropertyOption[];
}

const BASE = '/admin/property-groups';

export const propertyGroupsApi = {
	list: () => api.get<ApiResponse<PropertyGroup[]>>(BASE),
	get: (id: string) => api.get<ApiResponse<PropertyGroup>>(`${BASE}/${id}`),
	create: (data: CreatePropertyGroupRequest) =>
		api.post<ApiResponse<PropertyGroup>>(BASE, data),
	update: (id: string, data: CreatePropertyGroupRequest) =>
		api.put<ApiResponse<PropertyGroup>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`),

	createOption: (groupId: string, data: CreatePropertyOptionRequest) =>
		api.post<ApiResponse<PropertyOption>>(`${BASE}/${groupId}/options`, data),
	updateOption: (groupId: string, optId: string, data: CreatePropertyOptionRequest) =>
		api.put<ApiResponse<PropertyOption>>(`${BASE}/${groupId}/options/${optId}`, data),
	deleteOption: (groupId: string, optId: string) =>
		api.delete<void>(`${BASE}/${groupId}/options/${optId}`),
};
