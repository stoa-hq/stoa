import { request, type ApiResponse } from './client';

export interface CartItem {
	id: string;
	cart_id: string;
	product_id: string;
	variant_id?: string;
	quantity: number;
}

export interface Cart {
	id: string;
	currency: string;
	session_id: string;
	items: CartItem[];
}

const SESSION_ID_KEY = 'storefront_session_id';

function getSessionId(): string {
	if (typeof localStorage === 'undefined') return '';
	return localStorage.getItem(SESSION_ID_KEY) ?? '';
}

function saveSessionId(id: string) {
	if (typeof localStorage !== 'undefined') {
		localStorage.setItem(SESSION_ID_KEY, id);
	}
}

function cartRequest<T>(method: string, path: string, body?: unknown): Promise<ApiResponse<T>> {
	return request<T>(method, path, body, { extraHeaders: { 'X-Session-ID': getSessionId() } });
}

export const cartApi = {
	async create(currency = 'EUR'): Promise<ApiResponse<Cart>> {
		const sessionId = crypto.randomUUID();
		saveSessionId(sessionId);
		return request<Cart>('POST', '/store/cart', { currency, session_id: sessionId }, {
			extraHeaders: { 'X-Session-ID': sessionId }
		});
	},

	get(id: string): Promise<ApiResponse<Cart>> {
		return cartRequest<Cart>('GET', `/store/cart/${id}`);
	},

	addItem(cartId: string, productId: string, quantity: number, variantId?: string): Promise<ApiResponse<Cart>> {
		return cartRequest<Cart>('POST', `/store/cart/${cartId}/items`, {
			product_id: productId,
			variant_id: variantId ?? null,
			quantity
		});
	},

	updateItem(cartId: string, itemId: string, quantity: number): Promise<ApiResponse<Cart>> {
		return cartRequest<Cart>('PUT', `/store/cart/${cartId}/items/${itemId}`, { quantity });
	},

	removeItem(cartId: string, itemId: string): Promise<ApiResponse<Cart>> {
		return cartRequest<Cart>('DELETE', `/store/cart/${cartId}/items/${itemId}`);
	}
};
