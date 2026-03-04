import { api, type ApiResponse } from './client';

export interface Customer {
	id: string;
	email: string;
	first_name: string;
	last_name: string;
	active: boolean;
}

export interface RegisterRequest {
	email: string;
	password: string;
	first_name: string;
	last_name: string;
}

export const customersApi = {
	register(data: RegisterRequest): Promise<ApiResponse<Customer>> {
		return api.post<Customer>('/store/register', data);
	},

	getAccount(): Promise<ApiResponse<Customer>> {
		return api.get<Customer>('/store/account');
	},

	updateAccount(data: Partial<{ first_name: string; last_name: string }>): Promise<ApiResponse<Customer>> {
		return api.put<Customer>('/store/account', data);
	}
};
