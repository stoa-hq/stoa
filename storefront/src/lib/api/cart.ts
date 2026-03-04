import { api, type ApiResponse } from './client';

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

export const cartApi = {
	create(currency = 'EUR'): Promise<ApiResponse<Cart>> {
		return api.post<Cart>('/store/cart', { currency, session_id: crypto.randomUUID() });
	},

	get(id: string): Promise<ApiResponse<Cart>> {
		return api.get<Cart>(`/store/cart/${id}`);
	},

	addItem(cartId: string, productId: string, quantity: number, variantId?: string): Promise<ApiResponse<Cart>> {
		return api.post<Cart>(`/store/cart/${cartId}/items`, {
			product_id: productId,
			variant_id: variantId ?? null,
			quantity
		});
	},

	updateItem(cartId: string, itemId: string, quantity: number): Promise<ApiResponse<Cart>> {
		return api.put<Cart>(`/store/cart/${cartId}/items/${itemId}`, { quantity });
	},

	removeItem(cartId: string, itemId: string): Promise<ApiResponse<Cart>> {
		return api.delete<Cart>(`/store/cart/${cartId}/items/${itemId}`);
	}
};
