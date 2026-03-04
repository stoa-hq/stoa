import { api, type ApiResponse } from './client';

export interface CategoryTranslation {
	locale: string;
	name: string;
	description: string;
	slug: string;
}

export interface Category {
	id: string;
	parent_id?: string;
	position: number;
	active: boolean;
	translations: CategoryTranslation[];
	children?: Category[];
}

export function getCategoryName(cat: Category, locale = 'de-DE'): string {
	const t = cat.translations?.find((t) => t.locale === locale) ?? cat.translations?.[0];
	return t?.name ?? cat.id;
}

export const categoriesApi = {
	tree(): Promise<ApiResponse<Category[]>> {
		return api.get<Category[]>('/store/categories/tree');
	}
};
