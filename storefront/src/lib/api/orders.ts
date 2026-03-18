import { api, type ApiResponse } from './client';

export interface OrderItem {
	id: string;
	sku: string;
	name: string;
	quantity: number;
	unit_price_gross: number;
	total_gross: number;
}

export interface Order {
	id: string;
	order_number: string;
	status: string;
	currency: string;
	subtotal_gross: number;
	shipping_cost: number;
	total: number;
	created_at: string;
	items?: OrderItem[];
}

export interface CheckoutItem {
	product_id: string;
	variant_id?: string;
	quantity: number;
}

export interface CheckoutRequest {
	currency: string;
	billing_address: Record<string, string>;
	shipping_address: Record<string, string>;
	shipping_method_id?: string;
	payment_method_id?: string;
	notes?: string;
	items: CheckoutItem[];
}

export const ordersApi = {
	checkout(data: CheckoutRequest): Promise<ApiResponse<Order>> {
		return api.post<Order>('/store/checkout', data);
	},

	myOrders(): Promise<ApiResponse<Order[]>> {
		return api.get<Order[]>('/store/account/orders');
	}
};
