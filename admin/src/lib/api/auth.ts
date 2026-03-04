import { api } from './client';
import type { LoginResponse } from '$lib/types';

export const authApi = {
	login: (email: string, password: string) =>
		api.post<{ data: LoginResponse }>('/auth/login', { email, password }),

	refresh: (refreshToken: string) =>
		api.post<{ data: LoginResponse }>('/auth/refresh', { refresh_token: refreshToken }),

	logout: () => api.post<void>('/auth/logout')
};
