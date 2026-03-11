import { api, type ApiResponse } from './client';

export interface PaymentMethodTranslation {
	locale: string;
	name: string;
	description: string;
}

export interface PaymentMethod {
	id: string;
	provider: string;
	active: boolean;
	translations: PaymentMethodTranslation[];
}

export function getPaymentName(m: PaymentMethod, locale = 'de-DE'): string {
	const t = m.translations.find((t) => t.locale === locale) ?? m.translations.find((t) => t.locale === 'de-DE') ?? m.translations[0];
	return t?.name ?? m.provider;
}

export const paymentApi = {
	list(): Promise<ApiResponse<PaymentMethod[]>> {
		return api.get<PaymentMethod[]>('/store/payment-methods');
	}
};
