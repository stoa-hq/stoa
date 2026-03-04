import { api, type ApiResponse } from './client';

export interface LoginResponse {
	access_token: string;
	refresh_token: string;
}

export const authApi = {
	login(email: string, password: string): Promise<ApiResponse<LoginResponse>> {
		return api.post<LoginResponse>('/auth/login', { email, password });
	},

	logout(): Promise<ApiResponse<null>> {
		return api.post<null>('/auth/logout', {});
	},

	refresh(refreshToken: string): Promise<ApiResponse<LoginResponse>> {
		return api.post<LoginResponse>('/auth/refresh', { refresh_token: refreshToken });
	}
};
