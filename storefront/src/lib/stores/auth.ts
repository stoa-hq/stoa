import { writable } from 'svelte/store';
import { getAccessToken, getRefreshToken, setTokens, clearTokens } from '$lib/api/client';

interface AuthState {
	accessToken: string | null;
	refreshToken: string | null;
	user: { email: string; firstName?: string } | null;
}

function parseJwtPayload(token: string): Record<string, unknown> | null {
	try {
		const b64url = token.split('.')[1];
		const b64 = b64url.replace(/-/g, '+').replace(/_/g, '/');
		const padded = b64.padEnd(b64.length + (4 - (b64.length % 4)) % 4, '=');
		return JSON.parse(atob(padded));
	} catch {
		return null;
	}
}

function createAuthStore() {
	const initial: AuthState = { accessToken: null, refreshToken: null, user: null };

	if (typeof localStorage !== 'undefined') {
		const at = getAccessToken();
		const rt = getRefreshToken();
		if (at) {
			initial.accessToken = at;
			initial.refreshToken = rt;
			const payload = parseJwtPayload(at);
			if (payload) {
				initial.user = { email: (payload.email as string) ?? '' };
			}
		}
	}

	const { subscribe, set } = writable<AuthState>(initial);

	return {
		subscribe,
		login(accessToken: string, refreshToken: string) {
			setTokens(accessToken, refreshToken);
			const payload = parseJwtPayload(accessToken);
			const user = payload ? { email: (payload.email as string) ?? '' } : null;
			set({ accessToken, refreshToken, user });
		},
		logout() {
			clearTokens();
			set({ accessToken: null, refreshToken: null, user: null });
		},
		isAuthenticated(): boolean {
			const at = getAccessToken();
			if (!at) return false;
			const payload = parseJwtPayload(at);
			if (!payload) return false;
			return Date.now() / 1000 < (payload.exp as number);
		}
	};
}

export const authStore = createAuthStore();
