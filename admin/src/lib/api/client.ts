import { authStore } from '$lib/stores/auth';
import { get } from 'svelte/store';

const API_BASE = '/api/v1';

export class ApiClientError extends Error {
	constructor(
		public status: number,
		public errors: { code: string; detail: string; field?: string }[]
	) {
		super(errors[0]?.detail ?? `HTTP ${status}`);
		this.name = 'ApiClientError';
	}
}

const MUTATING_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

function readCsrfCookie(): string | null {
	if (typeof document === 'undefined') return null;
	const m = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]+)/);
	return m ? decodeURIComponent(m[1]) : null;
}

// Fetches /health once to let the server set the csrf_token cookie.
let csrfPrimingPromise: Promise<void> | null = null;
async function ensureCsrfToken(): Promise<string | null> {
	const token = readCsrfCookie();
	if (token) return token;
	if (!csrfPrimingPromise) {
		csrfPrimingPromise = fetch(`${API_BASE}/health`)
			.catch(() => {})
			.then(() => {});
	}
	await csrfPrimingPromise;
	return readCsrfCookie();
}

async function request<T>(
	method: string,
	path: string,
	body?: unknown,
	options: RequestInit = {}
): Promise<T> {
	const auth = get(authStore);
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(options.headers as Record<string, string>)
	};

	if (auth.accessToken) {
		// Bearer token → CSRF check is bypassed by the server.
		headers['Authorization'] = `Bearer ${auth.accessToken}`;
	} else if (MUTATING_METHODS.has(method)) {
		// No Bearer token → server applies CSRF Double-Submit-Cookie check.
		// Ensure the cookie exists (fetches /health once if not yet set), then echo it.
		const csrfToken = await ensureCsrfToken();
		if (csrfToken) headers['X-CSRF-Token'] = csrfToken;
	}

	const response = await fetch(`${API_BASE}${path}`, {
		method,
		headers,
		body: body !== undefined ? JSON.stringify(body) : undefined,
		...options
	});

	// Token expired – try refresh once
	if (response.status === 401 && auth.refreshToken) {
		const refreshed = await tryRefresh(auth.refreshToken);
		if (refreshed) {
			headers['Authorization'] = `Bearer ${get(authStore).accessToken}`;
			const retry = await fetch(`${API_BASE}${path}`, {
				method,
				headers,
				body: body !== undefined ? JSON.stringify(body) : undefined
			});
			return handleResponse<T>(retry);
		}
		authStore.logout();
		if (typeof window !== 'undefined') window.location.href = '/admin/login';
		throw new ApiClientError(401, [{ code: 'unauthorized', detail: 'Session expired' }]);
	}

	return handleResponse<T>(response);
}

async function handleResponse<T>(response: Response): Promise<T> {
	const text = await response.text();
	const json = text ? JSON.parse(text) : {};

	if (!response.ok) {
		const errors = json.errors ?? [{ code: 'error', detail: `HTTP ${response.status}` }];
		throw new ApiClientError(response.status, errors);
	}

	return json as T;
}

async function tryRefresh(refreshToken: string): Promise<boolean> {
	try {
		// Refresh is a POST without Bearer → include CSRF token.
		const csrfToken = readCsrfCookie();
		const headers: Record<string, string> = { 'Content-Type': 'application/json' };
		if (csrfToken) headers['X-CSRF-Token'] = csrfToken;

		const res = await fetch(`${API_BASE}/auth/refresh`, {
			method: 'POST',
			headers,
			body: JSON.stringify({ refresh_token: refreshToken })
		});
		if (!res.ok) return false;
		const data = await res.json();
		authStore.setTokens(data.data.access_token, data.data.refresh_token);
		return true;
	} catch {
		return false;
	}
}

export const api = {
	get: <T>(path: string) => request<T>('GET', path),
	post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
	put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
	delete: <T>(path: string) => request<T>('DELETE', path),

	// Multipart upload (for media)
	upload: async <T>(path: string, formData: FormData): Promise<T> => {
		const auth = get(authStore);
		const headers: Record<string, string> = {};
		if (auth.accessToken) headers['Authorization'] = `Bearer ${auth.accessToken}`;

		const response = await fetch(`${API_BASE}${path}`, {
			method: 'POST',
			headers,
			body: formData
		});
		return handleResponse<T>(response);
	}
};
