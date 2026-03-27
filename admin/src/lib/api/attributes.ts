import { api } from './client';
import type { ApiResponse } from '$lib/types';

export interface AttributeTranslation {
	locale: string;
	name: string;
	description?: string;
}

export interface AttributeOptionTranslation {
	locale: string;
	name: string;
}

export interface AttributeOption {
	id: string;
	attribute_id: string;
	position: number;
	translations?: AttributeOptionTranslation[];
}

export type AttributeType = 'text' | 'number' | 'select' | 'multi_select' | 'boolean';

export interface Attribute {
	id: string;
	identifier: string;
	type: AttributeType;
	unit?: string;
	position: number;
	filterable: boolean;
	required: boolean;
	created_at: string;
	updated_at: string;
	translations?: AttributeTranslation[];
	options?: AttributeOption[];
}

export interface CreateAttributeRequest {
	identifier: string;
	type: AttributeType;
	unit?: string;
	position?: number;
	filterable?: boolean;
	required?: boolean;
	translations: AttributeTranslation[];
}

export interface UpdateAttributeRequest {
	identifier: string;
	type: AttributeType;
	unit?: string;
	position?: number;
	filterable?: boolean;
	required?: boolean;
	translations: AttributeTranslation[];
}

export interface CreateAttributeOptionRequest {
	position?: number;
	translations: AttributeOptionTranslation[];
}

const BASE = '/admin/attributes';

export const attributesApi = {
	list: () => api.get<ApiResponse<Attribute[]>>(BASE),
	get: (id: string) => api.get<ApiResponse<Attribute>>(`${BASE}/${id}`),
	create: (data: CreateAttributeRequest) => api.post<ApiResponse<Attribute>>(BASE, data),
	update: (id: string, data: UpdateAttributeRequest) =>
		api.put<ApiResponse<Attribute>>(`${BASE}/${id}`, data),
	delete: (id: string) => api.delete<void>(`${BASE}/${id}`),

	createOption: (attributeId: string, data: CreateAttributeOptionRequest) =>
		api.post<ApiResponse<AttributeOption>>(`${BASE}/${attributeId}/options`, data),
	updateOption: (attributeId: string, optId: string, data: CreateAttributeOptionRequest) =>
		api.put<ApiResponse<AttributeOption>>(`${BASE}/${attributeId}/options/${optId}`, data),
	deleteOption: (attributeId: string, optId: string) =>
		api.delete<void>(`${BASE}/${attributeId}/options/${optId}`),
};
