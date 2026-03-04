import { api, type ApiResponse } from './client';

export interface ShippingMethodTranslation {
	locale: string;
	name: string;
	description: string;
}

export interface ShippingMethod {
	id: string;
	active: boolean;
	price_gross: number;
	currency: string;
	translations: ShippingMethodTranslation[];
}

export function getShippingName(m: ShippingMethod, locale = 'de-DE'): string {
	const t = m.translations.find((t) => t.locale === locale) ?? m.translations[0];
	return t?.name ?? m.id;
}

export const shippingApi = {
	list(): Promise<ApiResponse<ShippingMethod[]>> {
		return api.get<ShippingMethod[]>('/store/shipping-methods');
	}
};
